# Running the Enterprise ATS in VS Code

A step-by-step guide from zero to a running API. Commands are shown for
macOS/Linux and Windows (PowerShell) where they differ.

---

## 0. What you'll have at the end

- **API** on `http://localhost:8080` (Swagger UI at `/docs`)
- **Worker** processing background jobs (emails, reminders, resume parsing)
- **Postgres + Redis + Mailpit** running in Docker

---

## 1. Install the tools (one-time)

| Tool | Why | Install |
|------|-----|---------|
| **Go 1.23+** | builds & runs the app | https://go.dev/dl/ — then check `go version` |
| **Docker Desktop** | runs Postgres/Redis/Mailpit | https://www.docker.com/products/docker-desktop/ |
| **VS Code** | editor | https://code.visualstudio.com/ |
| **golang-migrate** | applies DB migrations | `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |

After installing Go, make sure Go's bin folder is on your PATH so `migrate`
works:
- **macOS/Linux:** add `export PATH="$PATH:$(go env GOPATH)/bin"` to `~/.zshrc` or `~/.bashrc`
- **Windows:** add `%USERPROFILE%\go\bin` to your PATH (search "Edit environment variables")

> `make` is optional. This guide uses direct commands; `make <target>` is just a
> shortcut (macOS/Linux, or Windows with `make` installed).

---

## 2. Open the project in VS Code

1. Open **VS Code** → **File → Open Folder…** → select the `ats` folder.
2. VS Code will suggest **recommended extensions** (Go, Docker, REST Client) —
   click **Install**. If not prompted, install the **Go** extension by Google.
3. The Go extension will offer to install its tools (gopls, dlv, etc.) — click
   **Install All**. Wait for it to finish (bottom-right notifications).

---

## 3. Fetch dependencies

Open a terminal in VS Code (**Terminal → New Terminal**) and run:

```bash
go mod tidy
```

This downloads every dependency (Gin, pgx, Asynq, the PDF library, the AWS SDK
for the S3 driver, etc.) and creates `go.sum`. **Run this before anything else.**

If it fails to resolve a version, run these and re-try `go mod tidy`:

```bash
go get github.com/ledongthuc/pdf@latest
go get github.com/aws/aws-sdk-go-v2/config@latest
go get github.com/aws/aws-sdk-go-v2/service/s3@latest
```

---

## 4. Start the infrastructure (Docker)

Make sure **Docker Desktop is running**, then:

```bash
docker compose up -d
```

This starts three containers:

| Service | Port | Notes |
|---------|------|-------|
| Postgres | 5432 | user `ats`, password `ats`, db `ats` |
| Redis | 6379 | queue + health |
| Mailpit | 8025 | test inbox — open http://localhost:8025 |

Check they're healthy: `docker compose ps`.

---

## 5. Apply the database migrations

```bash
migrate -path migrations -database "postgres://ats:ats@localhost:5432/ats?sslmode=disable" up
```

You should see it apply migrations `1` through `8`. (In VS Code you can instead
run **Terminal → Run Task… → db: migrate up**.)

---

## 6. Run the API and worker

### Option A — Press F5 (recommended)

1. Open the **Run and Debug** panel (the ▷ bug icon in the left bar, or `Ctrl/Cmd+Shift+D`).
2. Pick **Run API + Worker** from the dropdown, then press the green ▶ (or **F5**).
   Both processes start with breakpoints enabled.

You can also run **Run API** and **Run Worker** separately.

### Option B — Terminal

Two terminals:

```bash
# terminal 1
go run ./cmd/api

# terminal 2  (route emails to Mailpit)
#   macOS/Linux:
ATS_SMTP_HOST=localhost go run ./cmd/worker
#   Windows PowerShell:
$env:ATS_SMTP_HOST="localhost"; go run ./cmd/worker
```

When the API prints `http server listening ... :8080`, it's up.

---

## 7. Try it out

### Open the API docs
Go to **http://localhost:8080/docs** — interactive Swagger UI for every endpoint.

### Create your first admin
Sign up a user, then promote it to admin (the first admin must be seeded).
Using the Postgres container (no extra tools needed):

```bash
docker compose exec postgres psql -U ats -d ats -c "UPDATE users SET role='admin' WHERE email='jane@example.com';"
```

### Send some requests
Open **`api/requests.http`** in VS Code and click **"Send Request"** above each
block (needs the REST Client extension). Or use curl:

```bash
# sign up
curl -X POST localhost:8080/api/v1/auth/signup -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","password":"password123","full_name":"Jane Doe"}'

# log in -> copy tokens.access_token from the response
curl -X POST localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","password":"password123"}'

# use the token
curl localhost:8080/api/v1/me -H "Authorization: Bearer <access_token>"
```

Actions that send email (signup follow-ups, offers, reminders) show up in the
**Mailpit inbox at http://localhost:8025**.

---

## 8. Run the tests

```bash
go test ./... -race -cover
```

Or **Terminal → Run Task… → test**. These are pure unit tests — they don't need
the database or Docker.

---

## 9. Everyday workflow (quick reference)

```bash
docker compose up -d        # start infra (once per session)
go run ./cmd/api            # or press F5 -> "Run API + Worker"
go run ./cmd/worker
# ... code, save (auto-formats), test ...
go test ./...
docker compose down         # stop infra when done
```

---

## 10. Troubleshooting

| Symptom | Fix |
|--------|-----|
| `go: command not found` | Go isn't installed or not on PATH — reinstall, reopen terminal |
| `connect postgres: ... connection refused` | Docker isn't running, or `docker compose up -d` wasn't run |
| `migrate: command not found` | Go bin dir not on PATH (see step 1) |
| `go mod tidy` fails on a version | run the `go get ... @latest` commands in step 3 |
| API starts but `/readyz` returns 503 | Postgres or Redis is down — check `docker compose ps` |
| Emails don't appear in Mailpit | start the worker with `ATS_SMTP_HOST=localhost` (the F5 "Run Worker" config already sets this) |
| Port 5432/6379/8080 already in use | stop the other process, or change the port via `ATS_SERVER_PORT` / edit `docker-compose.yml` |
| Uploaded resume never gets parsed | the **worker** must be running and share the same `storage.dir` as the API (default `./storage`, fine when both run from the project root) |

---

## Notes

- The module path is `github.com/faizan/ats`. You don't need to change it to run
  locally. If you push to your own GitHub repo, update `module` in `go.mod` and
  the imports to match.
- All settings have working defaults (they match `docker-compose.yml`), so no
  config file is required. To override anything, copy `config.example.yaml` to
  `config.yaml` or set `ATS_*` environment variables.
