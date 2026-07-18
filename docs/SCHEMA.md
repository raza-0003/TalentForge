# Database Schema

The full schema spans all project phases. Migrations live in `migrations/` and
apply in order via `make migrate-up`.

## Entity-relationship overview

```
                         ┌──────────────┐
                         │    users     │  role: candidate | recruiter | admin
                         └──────┬───────┘
        ┌───────────────┬───────┼───────────────┬──────────────────┐
        │ 1:1           │ 1:N   │ 1:N           │ 1:N              │ 1:N (as interviewer)
 ┌──────▼─────────┐ ┌───▼────┐ ┌▼──────────┐ ┌──▼──────────┐  ┌────▼───────┐
 │candidate_      │ │resumes │ │  jobs     │ │refresh_     │  │ interviews │
 │profiles        │ │        │ │(created_  │ │tokens       │  │(interviewer│
 │(skills, links) │ │        │ │ by)       │ │             │  │ _id)       │
 └────────────────┘ └───┬────┘ └────┬──────┘ └─────────────┘  └────────────┘
                        │           │ 1:N
                        │      ┌────▼────────────┐
              resume_id │      │  applications   │  status: applied → … → hired/rejected
              (SET NULL)└─────▶│ (job_id,        │
                               │  candidate_id)  │
                               └───┬─────────┬───┘
                    1:N            │         │  1:N
              ┌────────────────────▼──┐   ┌──▼──────────────┐
              │  application_events    │   │  interviews     │ 1:1  ┌─────────────────────┐
              │  (audit / timeline)    │   │  (application_  │─────▶│ interview_feedback  │
              └────────────────────────┘   │   id)           │      │ (rating 1-5, rec.)  │
                                           └──┬──────────────┘      └─────────────────────┘
                                              │ 1:N
                                        ┌─────▼────────┐
                                        │ offer_letters │  status: draft → sent → accepted
                                        └──────────────┘

  notifications  ── outbox/email log, optionally linked to users(recipient_user_id)
```

## Tables by phase

| Phase | Tables | Purpose |
|-------|--------|---------|
| 1 — Auth | `users`, `refresh_tokens` | Accounts with roles; hashed, revocable refresh tokens |
| 2 — Candidate | `candidate_profiles`, `resumes` | Profile (skills as `text[]`, links as `jsonb`) + uploaded resumes |
| 2 — Recruiting | `jobs`, `applications`, `application_events` | Job postings, one application per candidate/job, append-only timeline |
| 3 — Interviews | `interviews`, `interview_feedback` | Scheduling + structured 1–5 feedback with a recommendation |
| 4 — Automation | `notifications` | Email outbox the background worker drains |
| 5 — Offers | `offer_letters` | Offer records + generated PDF object key |

## Design decisions

- **Enums over free-text status** — `application_status`, `job_status`, `interview_status`, etc. are Postgres `ENUM` types, so invalid states are rejected at the database, not just in Go.
- **`bigint` identity PKs** — every ID scans cleanly into a Go `int64`. If you later want opaque, non-enumerable public IDs, add a `public_id uuid DEFAULT gen_random_uuid()` column per table (a one-migration change) and expose that in URLs.
- **Append-only `application_events`** — status changes never overwrite history; the candidate timeline and recruiter audit both read from this table.
- **Hashed refresh tokens** — only a SHA-256 hash is stored, so a database leak can't be replayed. Rotation + revocation happen by row.
- **`updated_at` via trigger** — a single `set_updated_at()` trigger function keeps timestamps correct without application code remembering to set them.
- **Search-ready indexes** — GIN index on `candidate_profiles.skills` for skill search; btree indexes on every foreign key and status column used for filtering and analytics aggregation.
- **`ON DELETE` policies** — child rows cascade with their parent (e.g. applications with a job); references that are merely informational (`resume_id`, `actor_id`) use `SET NULL` so history survives.
