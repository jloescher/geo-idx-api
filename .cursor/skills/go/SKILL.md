---
name: go
description: Go 1.25+ Fiber API, pgx, PostgreSQL queue workers, and idx-api internal packages. Use when implementing handlers, services, jobs, migrations (goose), config, or tests in cmd/ and internal/.
---

# Go (idx-api)

## Stack

- **Go 1.25+**, **Fiber v2**, **pgx/v5** + **sqlx**, **goose** migrations
- **PostgreSQL** queue (`jobs` table, `{"type":"bridge.fetch_page","args":{...}}` payloads)
- **Argon2id** passwords (`internal/auth/password`); legacy bcrypt verified + upgraded on login
- **API tokens**: SHA-256 hashed in `personal_access_tokens.token` (re-issue after cutover)

## Layout

| Path | Role |
|------|------|
| `cmd/api` | HTTP server (:8000) |
| `cmd/worker` | Queue consumer (`WORKER_QUEUES`) |
| `cmd/scheduler` | Cron → enqueue jobs |
| `cmd/seed` | `make seed-admin` (ADMIN_SEED_*) |
| `internal/api` | Routes, middleware (`domain.token`, `mls.access`) |
| `internal/handler/` | bridge, gis, images, dashboard, auth, marketing |
| `internal/service/` | sync, search, gis, cache, mls, crypto |
| `internal/job` | Queue handler registry |
| `internal/queue` | Enqueue, reserve, batches |
| `internal/repository` | DB access |
| `internal/web` | Embedded CSS/JS + HTML helpers |
| `migrations/` | Goose SQL (`00001_initial.sql`) |

## Commands

```bash
make migrate          # GOOSE_DBSTRING required
make seed-admin
make run-api
make run-worker
make run-scheduler
GOFLAGS=-mod=mod go test ./...
go build ./cmd/...
```

## Patterns

- Config: `internal/config` from `.env` (see `.env.example`)
- Handlers: thin; delegate to `internal/service`
- Queue types: `internal/queue/payload.go` constants (`bridge.fetch_page`, `spark.persist_chunk`, …)
- Never embed MLS page JSON in job payloads — use `replica_pages`
- Revenue impact: comment monetization-sensitive logic

## Auth

- Domain header / Referer host **or** Bearer PAT + domain binding
- Abilities: `idx:access` (teaser) vs `idx:full`
- Dashboard: session cookie after `/login`

## Tests

- `go test ./internal/...`
- Integration: `TEST_DATABASE_URL` for DB tests
- HTTP: `net/http/httptest` or Fiber test helpers; fake upstream MLS/GIS in handler tests

## Do not

- Reintroduce Laravel/PHP/Composer for this service
- Use `php artisan` or Eloquent in new code
- Store plaintext API tokens

See [README.md](../../README.md), [docs/go-cutover.md](../../docs/go-cutover.md), [AGENTS.md](../../AGENTS.md).
