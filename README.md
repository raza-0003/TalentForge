# Enterprise ATS — Backend

A simplified applicant tracking system (Greenhouse/Lever style) built with Go.
The **full schema** (all phases) and the **core application** — auth, jobs,
applications with an audit timeline, interviews, feedback, and admin — are
implemented. See [Status](#status) for what's built vs. planned, and
[docs/SCHEMA.md](docs/SCHEMA.md) for the data model.

## Tech stack

| Concern         | Choice                          |
| --------------- | ------------------------------- |
| HTTP            | Gin                             |
| Database        | PostgreSQL (via `pgx/v5`)       |
| Cache / queue   | Redis (via `go-redis/v9`)       |
| Config          | Viper                           |
| Logging         | Zap                             |
| Migrations      | `golang-migrate`                |
| Container       | Docker (multi-stage)            |
| CI              | GitHub Actions                  |

## Project layout

```
cmd/
  api/            # HTTP server entrypoint
  worker/         # Asynq worker entrypoint (Phase 4 — planned)
internal/
  config/         # Viper configuration loading
  logger/         # Zap logger construction
  middleware/     # Gin middleware: request logging, recovery, JWT auth, RBAC
  database/       # Postgres + Redis clients
  domain/         # entities + sentinel errors (no external deps)
  auth/           # password hashing, JWT, refresh tokens
  httputil/       # shared JSON response + pagination helpers
  repository/     # pgx data access, one file per aggregate
  service/        # business logic (auth, jobs, applications, interviews, admin)
  handler/        # Gin HTTP handlers + router wiring
migrations/       # SQL migrations (golang-migrate), all phases
docs/SCHEMA.md    # ER diagram + schema rationale
.github/workflows # CI pipeline
```

Request flow: **handler → service → repository → Postgres**. Handlers never touch
SQL; repositories never know about HTTP.

> The module path is `github.com/faizan/ats`. Rename it in `go.mod` and the
> imports in `cmd/api/main.go` to your own path before pushing.

## Prerequisites

- Go 1.23+
- Docker (for Postgres + Redis)
- [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI (for migrations)
- [`golangci-lint`](https://golangci-lint.run) (optional, for `make lint`)

## Getting started

```bash
# 1. Resolve dependencies (also generates go.sum).
make tidy          # or: go mod tidy

# 2. Start Postgres + Redis.
make up

# 3. Apply migrations.
make migrate-up

# 4. Run the API.
make run
```

The server listens on `:8080` by default.

## Health endpoints

```bash
curl localhost:8080/healthz
# {"status":"ok"}

curl localhost:8080/readyz
# {"status":"ready","checks":{"postgres":"ok","redis":"ok"}}
```

`/healthz` is a liveness probe (process is up). `/readyz` is a readiness probe
that pings Postgres and Redis and returns `503` if either is unreachable.

## API docs

Interactive **Swagger UI** is served at `http://localhost:8080/docs`, backed by
the OpenAPI 3.0 spec at `/openapi.yaml` (embedded into the binary via `go:embed`
from `internal/docs/openapi.yaml`). Use the **Authorize** button to paste a
bearer token and try endpoints live.

## Testing

Unit tests are pure — no database or network — so they run anywhere:

```bash
make test    # go test ./... -race -cover
```

They cover resume field extraction & skill canonicalization (`parser`), the PDF
renderer (`pdf`), password hashing + JWT round-trip/expiry (`auth`), enum
validation (`domain`), and money formatting (`service`). CI (GitHub Actions)
runs `go vet`, build, the test suite with the race detector, and `golangci-lint`
on every push and PR.

## Configuration

Configuration resolves in this order (later wins):

1. Built-in defaults (see `internal/config/config.go`)
2. `config.yaml` in the working directory (optional — copy from `config.example.yaml`)
3. Environment variables, prefixed `ATS_` with `.` replaced by `_`

Examples:

```bash
ATS_SERVER_PORT=9090
ATS_POSTGRES_PASSWORD=super-secret
ATS_LOG_LEVEL=debug
ATS_ENV=prod            # switches Zap to JSON output and Gin to release mode
```

The default Postgres/Redis settings match `docker-compose.yml`, so a fresh
`make up && make run` works with no config file.

**File storage** is pluggable behind the `storage.Storage` interface. The default
`local` driver writes under `storage.dir` (the API and worker must share it). Set
`ATS_STORAGE_DRIVER=s3` with `ATS_STORAGE_BUCKET` / `ATS_STORAGE_REGION` (and
`ATS_STORAGE_ENDPOINT` for MinIO) to use S3; credentials come from the standard
AWS environment (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, or an IAM role).

## API

Base path: `/api/v1`. Send `Authorization: Bearer <access_token>` for protected
routes. Roles: **candidate** (self-signup), **recruiter** & **admin**
(provisioned by an admin).

### Auth (public)
| Method | Path | Body |
|--------|------|------|
| POST | `/auth/signup` | `{email, password, full_name}` — creates a candidate |
| POST | `/auth/login` | `{email, password}` → `{user, tokens}` |
| POST | `/auth/refresh` | `{refresh_token}` → new token pair (rotates) |
| POST | `/auth/logout` | `{refresh_token}` — revokes the token |
| GET | `/me` | *(auth)* current user |

### Jobs
| Method | Path | Access |
|--------|------|--------|
| GET | `/jobs?status=open&q=go&limit=&offset=` | public |
| GET | `/jobs/:id` | public |
| POST | `/jobs` | recruiter/admin |
| PUT | `/jobs/:id` | owner recruiter/admin |
| POST | `/jobs/:id/close` | owner recruiter/admin |

### Applications
| Method | Path | Access |
|--------|------|--------|
| POST | `/jobs/:id/apply` | candidate — `{resume_id?, cover_letter?}` |
| GET | `/candidate/applications` | candidate — own applications |
| GET | `/applications/:id` | owner candidate or recruiter/admin |
| GET | `/applications/:id/timeline` | owner candidate or recruiter/admin |
| GET | `/jobs/:id/applications?status=` | recruiter/admin |
| PATCH | `/applications/:id/status` | recruiter/admin — `{status, note?}` |

### Candidate profile
| Method | Path |
|--------|------|
| GET / PUT | `/candidate/profile` — `{phone, headline, location, links{}, skills[]}` |

### Resumes
| Method | Path | Access |
|--------|------|--------|
| POST | `/candidate/resumes` | candidate — multipart `file` (+ `primary=true`); queued for parsing |
| GET | `/candidate/resumes` | candidate — own resumes (incl. `parsed_data`) |
| POST | `/candidate/resumes/:id/primary` | candidate — set primary |
| GET | `/resumes/:id/download` | owner candidate or recruiter/admin |

Uploads (≤10 MB; pdf/doc/docx/txt/md/rtf) are stored via the storage driver and
a `resume:parse` task extracts emails, phones, and skills into `parsed_data`.
Text and DOCX use the standard library; **PDF** uses `ledongthuc/pdf` (panic-safe
— scanned/image-only or encrypted PDFs are stored and flagged `pdf_unreadable`
rather than failing). Attach a resume when applying by passing `resume_id` to
`POST /jobs/:id/apply`.

### Candidate search (recruiter/admin)
| Method | Path |
|--------|------|
| GET | `/candidates?skills=Go,PostgreSQL&match=all&q=<name>&limit=&offset=` |

Skill matching is index-backed (GIN on `candidate_profiles.skills`): `match=any`
(default) returns candidates with any listed skill, `match=all` requires all of
them. Skills are canonicalized on both sides, so `go` matches a stored `Go`.
`q` additionally filters by name/headline.

### Interviews & feedback (recruiter/admin)
| Method | Path | Body |
|--------|------|------|
| POST | `/applications/:id/interviews` | `{interviewer_id, scheduled_at (RFC3339), duration_minutes?, mode?, location?}` |
| GET | `/applications/:id/interviews` | — |
| PATCH | `/interviews/:id` | reschedule / set status |
| POST | `/interviews/:id/feedback` | `{rating 1-5, recommendation, strengths?, weaknesses?, comments?}` |
| GET | `/interviews/:id/feedback` | — |

### Offers
| Method | Path | Access |
|--------|------|--------|
| POST | `/applications/:id/offers` | recruiter/admin — `{position_title?, salary_amount?, salary_currency?, start_date? (YYYY-MM-DD), expires_at? (RFC3339)}` |
| GET | `/applications/:id/offers` | recruiter/admin |
| POST | `/offers/:id/send` | recruiter/admin — notifies candidate, moves app to `offer` |
| POST | `/offers/:id/rescind` | recruiter/admin |
| GET | `/candidate/offers` | candidate — own offers |
| POST | `/offers/:id/accept` | candidate — moves app to `hired` |
| POST | `/offers/:id/decline` | candidate |
| GET | `/offers/:id/download` | owner candidate or recruiter/admin — the PDF |

Creating an offer renders a one-page PDF letter (generated in-process, no
external library) and stores it via the storage driver. Lifecycle:
`draft → sent → accepted / declined / rescinded`.

### Admin (admin only)
| Method | Path |
|--------|------|
| POST | `/admin/recruiters` — provision a recruiter |
| GET | `/admin/recruiters` |
| PATCH | `/admin/users/:id/active` — `{is_active}` |
| GET | `/admin/analytics` — job/application counts by status |

> The first admin must be seeded directly, e.g.:
> `UPDATE users SET role='admin' WHERE email='you@example.com';`

### Quick smoke test
```bash
curl -X POST localhost:8080/api/v1/auth/signup \
  -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","password":"password123","full_name":"Jane Doe"}'

curl -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","password":"password123"}'
# → copy tokens.access_token, then:
curl localhost:8080/api/v1/me -H "Authorization: Bearer <access_token>"
```

## Background jobs (Phase 4)

The API never sends email on the request path. Instead it writes a row to the
`notifications` outbox and enqueues a task on Redis (via Asynq). A separate
**worker** process drains the queue and delivers the mail, marking each outbox
row `sent` or `failed`. Scheduling an interview also queues a reminder to run
24h beforehand.

The worker also runs an Asynq **Scheduler** that fires a **pending-notification
sweeper** on an interval (`worker.sweepcron`, default every 5 min). The sweep
re-delivers any outbox rows still `pending` past `worker.sweepaftermin` (default
15 min) — i.e. rows whose original enqueue was lost — so no notification is
silently dropped. Each swept row leaves `pending` (→ `sent`/`failed`), so the
sweep always converges.

Run the worker alongside the API (both share the same config and codebase):

```bash
make up          # postgres + redis + mailpit
make run         # API   (terminal 1)
make worker      # worker (terminal 2)
```

By default emails are **logged, not sent**. To deliver them to the local Mailpit
inbox (web UI at http://localhost:8025), point the worker at it:

```bash
ATS_SMTP_HOST=localhost make worker
```

For a real provider (e.g. SendGrid), set `ATS_SMTP_HOST`, `ATS_SMTP_PORT`,
`ATS_SMTP_USERNAME`, `ATS_SMTP_PASSWORD`, and `ATS_SMTP_FROM`.

## Status

**Implemented:** full schema (all phases); JWT auth with refresh-token rotation
& RBAC; candidate profiles; jobs; applications with an append-only timeline;
interviews; feedback; admin (recruiter management + analytics); the
**Phase 4 background worker** (outbox-backed email over SMTP, scheduled interview
reminders, pending-notification sweeper); **resume upload + background parsing**
(text/DOCX/PDF field extraction, resumes attachable to applications);
**index-backed candidate skill search**; and **PDF offer-letter generation**
with a full accept/decline lifecycle.

Also included: **OpenAPI/Swagger docs** at `/docs`, a pluggable **S3 storage
driver**, and a **pure unit-test suite** wired into CI.

**Possible next steps:**
- Richer analytics (time-to-hire, funnel conversion) and a metrics endpoint.
- A page-break/multi-page PDF renderer (the built-in one is single-page).
- Integration tests against a throwaway Postgres container (e.g. testcontainers).
