# Quantyra IDX API (Go)

High-performance MLS proxy and image delivery service for Quantyra IDX, written in **Go 1.25+** with **Fiber**, **PostgreSQL + PostGIS**, and a **PostgreSQL-backed job queue** (no Redis).

## Features

- Bridge Data Output + Spark MLS proxy under `/api/v1/*`
- Multi-MLS catalog (`bridge_stellar`, `spark_beaches`; public `?dataset=` param)
- Domain + API token auth, MLS feed allowlists, audit logging
- Image proxy `/images/*` with NVMe filesystem cache
- PostGIS listings mirror + replication workers (fetch → staged `replica_pages` → chunk persist → finalize)
- Hybrid `POST /api/v1/search` (PostGIS / live MLS / split)
- GIS parcel proxy `/api/v1/gis`
- Invite-only dashboard (`/dashboard`) for domains and API keys
- Scheduler: listings cache refresh (15m), replication kickoff, GIS probe, crypto pricing

## Project layout

```text
idx-api/
├── cmd/api/           # HTTP server (:8000)
├── cmd/worker/        # Queue consumer (WORKER_QUEUES)
├── cmd/scheduler/     # Cron dispatcher
├── internal/          # Application code
├── migrations/        # Goose SQL schema
├── Dockerfile         # Targets: api | worker | scheduler
├── docker-compose.dev.yml
└── docs/
```

## Local development

**Prerequisites:** Go 1.25+, PostgreSQL with PostGIS, `.env` from `.env.example`.

```bash
cp .env.example .env
# Edit DB_*, BRIDGE_API_KEY, SPARK_ACCESS_TOKEN, etc.

# Database (migrations — no global goose required)
export GOOSE_DBSTRING="postgres://postgres:postgres@127.0.0.1:5432/idx_api?sslmode=disable"
make migrate
make seed-admin   # ADMIN_SEED_EMAIL / ADMIN_SEED_PASSWORD in .env
# Or install goose on PATH:
#   make migrate-install
#   export GOOSE_DRIVER=postgres GOOSE_DBSTRING="postgres://..."
#   goose -dir migrations up

# Run API
go run ./cmd/api

# Worker (separate terminal)
export WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
go run ./cmd/worker

# Scheduler
go run ./cmd/scheduler
```

### Docker Compose (Go stack)

```bash
docker compose -f docker-compose.dev.yml up --build
```

## Build & test

```bash
GOFLAGS=-mod=mod go build ./cmd/...
GOFLAGS=-mod=mod go test ./...
go fmt ./...
golangci-lint run   # optional
```

## Deployment (Coolify / Dokploy)

| Service | Dockerfile target | Port |
|---------|-------------------|------|
| idx-api-web | `api` | 8000 |
| idx-api-worker-fetch | `worker` | — |
| idx-api-worker-persist | `worker` | — |
| idx-api-scheduler | `scheduler` | — |
| idx-images | `Dockerfile.idx-images` | 8080 |

Environment variables match [`.env.example`](.env.example). See **[docs/go-cutover.md](docs/go-cutover.md)** for migration from Laravel and API key re-issue.

## API documentation

- [docs/idx-api-bridge-proxy.md](docs/idx-api-bridge-proxy.md)
- [docs/gis-api.md](docs/gis-api.md)
- [docs/database-migrations.md](docs/database-migrations.md)

## Health

- `GET /healthz` — liveness
- `GET /readyz` — Postgres + PostGIS check
