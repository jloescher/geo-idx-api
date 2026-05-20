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

# Worker (separate terminal) — only processes rows the scheduler (or API) enqueues
export WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
go run ./cmd/worker

# Scheduler (separate terminal) — required for replication kickoff, cache refresh, etc.
go run ./cmd/scheduler
```

**Scheduler + worker:** The worker stays idle when the `jobs` table is empty. The scheduler enqueues `mls.replication_kickoff` every minute at **:00**; the worker runs kickoff, which enqueues `bridge.fetch_page` / `spark.fetch_page` on their queues. Run **both** processes against the same `DB_*` as the API. First kickoff log appears on the next minute boundary after `scheduler started`.

**Inspecting the database:** Completed jobs are **deleted** from `jobs` after success (normal queue behavior). Look at `replica_pages` during fetch, then `listings` after persist. Ensure `DB_*` in `.env` matches `GOOSE_DBSTRING` used for `make migrate`.

Bridge replication (`bridge.fetch_page`) and Spark beaches replication (`spark.fetch_page`) call upstream OData and write `replica_pages` → `listings` via the worker. Requires `SPARK_ACCESS_TOKEN` and replication host vars (see `docs/spark/idx-api-integration.md`).

Replicated rows split payload across `raw_data`, `media`, `unit`, `room`, `open_house`, and `custom_fields` (see [docs/listings-mirror.md](docs/listings-mirror.md)). Mirror-backed search returns flat RESO Property JSON.

After replication, verify indexed columns (not only `raw_data`):

```sql
SELECT COUNT(*) AS total,
       COUNT(list_price) AS with_price,
       COUNT(flood_zone_code) AS with_flood,
       COUNT(estimated_total_monthly_fees) AS with_fees,
       COUNT(coordinates) AS with_geom
FROM listings WHERE dataset_slug = 'stellar';
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
