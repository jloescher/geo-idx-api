# Quantyra IDX API

**Go 1.25+** service (Fiber, pgx, PostgreSQL queue) powering Quantyra's Bridge MLS proxy, Spark Beaches MLS proxy, GIS parcel/geometry proxy, authenticated user dashboard (domains, API keys, MLS feed scope), and secured image proxy delivery. The service sits between real estate MLS data (Bridge Data Output / Stellar MLS), public ArcGIS parcel sources, and customer tooling. Three public surfaces: **idx.quantyralabs.cc** (app/marketing), **idx-api.quantyralabs.cc** (API), **idx-images.quantyralabs.cc** (image proxy).

> **Note:** This repository was rewritten from Laravel 13 + Octane. Use `cmd/api`, `cmd/worker`, `cmd/scheduler`, and [README.md](README.md) for build/run instructions.

## Tech Stack

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| Runtime | Go | 1.25+ | Server language |
| HTTP | Fiber | 2.x | API router and middleware |
| Database | PostgreSQL + PostGIS | — | pgx + sqlx |
| Queue | PostgreSQL `jobs` | — | `cmd/worker` + LISTEN/NOTIFY |
| Migrations | goose | 3.x | `migrations/*.sql` |
| Dashboard UI | Embedded HTML/CSS | — | `internal/web` (`//go:embed`) |
| Auth | Session + PAT | — | Argon2id passwords; SHA-256 token hashes |
| Edge images | Nginx | 1.27 | `Dockerfile.idx-images` → Go API |

## Quick Start

```bash
# Prerequisites: Go 1.25+, PostgreSQL with PostGIS

cp .env.example .env
# Edit DB_*, BRIDGE_API_KEY, SPARK_ACCESS_TOKEN, ADMIN_SEED_*, WORKER_QUEUES

export GOOSE_DBSTRING="postgres://user:pass@127.0.0.1:5432/idx_api?sslmode=disable"
make migrate
make seed-admin

make run-api        # :8000
make run-worker     # separate terminal
make run-scheduler  # separate terminal

go test ./...
```

## Project Structure

```
idx-api/
├── cmd/api/              # HTTP server
├── cmd/worker/           # Queue consumer
├── cmd/scheduler/        # Cron → enqueue
├── cmd/seed/             # make seed-admin
├── internal/
│   ├── api/              # Routes, middleware
│   ├── handler/          # bridge, gis, images, dashboard, auth, marketing
│   ├── service/          # sync, search, gis, cache, comps, mls, crypto, auth
│   ├── job/              # Queue handler registry
│   ├── queue/            # PostgreSQL queue client
│   ├── repository/       # DB access
│   ├── config/           # .env loading
│   ├── auth/password/    # Argon2id
│   └── web/              # Embedded static + HTML layout
├── migrations/           # Goose SQL
├── docs/
├── Dockerfile            # api | worker | scheduler
└── docker-compose.dev.yml
```

## Architecture Overview

The service has three primary subsystems:

### 1. Bridge MLS Proxy (`/api/v1/*`)

Proxies Bridge Data Output with domain-based or PAT authentication. Key behaviors:
- **domain.token** + **mls.access** middleware resolve identity and feed access
- **Image URL rewriting**: Bridge photo URLs rewritten to `idx-images` public URLs
- **On-demand proxy cache**: repeat identical live requests stored in `mls_search_cache` (`X-IDX-Cache: HIT`/`MISS`); scheduler purges stale rows
- **Audit logging**: `mls_proxy_audit_logs` including optional `cache_hit`

### 2. GIS Parcel/Geometry Proxy (`/api/v1/gis`, `/api/v1/mls/{mlsCode}/gis`)

Public ArcGIS feature server proxy for Florida parcel data. Three-tier caching with generation-based invalidation, source failover, teaser mode for non-full tokens, and bbox limits.

### 3. Platform & user dashboard

- **Marketing home** (`/`) — `internal/handler/marketing`
- **User dashboard** (`/dashboard`) — domains, DNS TXT verify, API keys (`internal/handler/dashboard`)

```
┌──────────────┐        ┌──────────────┐        ┌──────────────┐
│  idx.*       │        │  idx-api.*   │        │ idx-images.* │
│  (App/Mktg)  │        │  (API)       │        │  (Nginx)     │
├──────────────┤        ├──────────────┤        ├──────────────┤
│ User dashboard │        │ Bridge Proxy │        │ Edge cache   │
│ (Go HTML)      │        │ GIS Proxy    │   ──▶  │ -> idx-api   │
│                │        │              │        │ /images/*    │
└──────────────┘        └──────┬───────┘        └──────────────┘
                               │
                               ▼
                       ┌──────────────┐        ┌──────────────┐
                       │  Bridge MLS  │        │  ArcGIS      │
                       │              │        │  Parcel Src  │
                       └──────────────┘        └──────────────┘
```

### Key modules

| Module | Location | Purpose |
|--------|----------|---------|
| Bridge proxy | `internal/handler/bridge` | MLS RESO proxy, search, stats |
| GIS proxy | `internal/handler/gis` | ArcGIS parcel proxy |
| Images | `internal/handler/images` | `/images/*` streaming cache |
| Domain/token auth | `internal/api/middleware/domain_token.go` | Domain slug and/or PAT |
| MLS access | `internal/api/middleware/mls_access.go` | Feed allowlists |
| Sync/replication | `internal/service/sync` | Bridge + Spark mirror jobs |
| Comps / BPO | `internal/service/comps` | Modes A–E, investor, BPO (14-line URAR), `home_value` |
| Dashboard | `internal/handler/dashboard` | Login, domains, API keys, invitations |
| Queue | `internal/queue`, `internal/job` | Job types and workers |

## Development Guidelines

### Code Style (Go)
- **Formatting**: `gofmt` / `make fmt`; optional `golangci-lint run ./...`
- **Layout**: standard Go project layout under `cmd/` and `internal/`
- **Naming**: PascalCase exported types; camelCase unexported; package names are short, lowercase (`comps`, `sync`, `mlspoxy`)
- **Handlers**: thin Fiber handlers in `internal/handler/*`; business logic in `internal/service/*`
- **Config**: loaded once in `internal/config` from `.env` (see `.env.example`)
- **Revenue impact comments**: mark monetization-sensitive paths (cache, access tiers) where applicable

### Import Order (Go)
1. Standard library
2. Third-party modules (`github.com/gofiber/...`, `github.com/jackc/...`)
3. `github.com/quantyralabs/idx-api/...` packages

### Database Conventions
- **Migrations**: goose SQL in `migrations/` — currently **`00001_initial.sql`** only (consolidated fresh schema)
- **Access**: `internal/repository` (pgx pool + sqlx where helpful)
- **PostGIS**: `listings.coordinates`; partial indexes for Active/Pending search
- **Queue**: PostgreSQL `jobs` table; payload JSON `{"type":"...","args":{...}}`

### Testing Patterns
- **Colocated tests**: `*_test.go` next to packages (`go test ./internal/service/comps/...`)
- **Full suite**: `make test` or `go test ./...`
- **External APIs**: use `httptest.Server` or stub upstream in unit tests; no live Bridge/Spark in CI by default
- **Database tests**: use a disposable Postgres DB; set `GOOSE_DBSTRING` before `make migrate` for integration-style tests

### Scheduled Tasks (`cmd/scheduler` → PostgreSQL `jobs`)
| Job type | Schedule | Queue | Purpose |
|----------|----------|-------|---------|
| `mls.replication_kickoff` | Every minute | default | Enqueue Bridge/Spark sync per dataset |
| `mls.proxy_cache_purge` | Every 15 min | default | **Purge** stale `mls_search_cache` rows (on-demand proxy cache; does not pre-warm Active/Pending) |
| `crypto.refresh_pricing` | Every 10 min | `COINGECKO_QUEUE` | CoinGecko snapshots |
| `mls.purge_replica_pages` | Daily 04:15 | default | Completed + failed `replica_pages` retention |
| `mls.purge_closed_listings` | Daily 03:05 | default | Mirror rolling window purge |
| `gis.probe_sources` | Monday 06:30 | `GIS_QUEUE` | ArcGIS metadata fingerprint / generation bump |

## Environment Variables

### Core

| Variable | Required | Description |
|----------|----------|-------------|
| `APP_PORT` | No | HTTP listen port (default `8000`) |
| `APP_URL` / `IDX_API_PUBLIC_URL` | Yes | Public API base URL |
| `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`, `DB_SSLMODE` | Yes | PostgreSQL connection (see `internal/config`) |
| `GOOSE_DBSTRING` | Migrate | DSN for `make migrate` (can mirror `DB_*`) |
| `ADMIN_SEED_EMAIL`, `ADMIN_SEED_PASSWORD` | Seed | `make seed-admin` bootstrap user |
| `WORKER_QUEUES` | Worker | Comma-separated queues (see `.env.example`) |

### Public URLs

| Variable | Required | Description |
|----------|----------|-------------|
| `IDX_PLATFORM_URL` | Yes | Public app URL (idx.quantyralabs.cc) |
| `IDX_API_PUBLIC_URL` | Yes | Public API URL (defaults to APP_URL) |
| `IDX_IMAGES_PUBLIC_URL` | Yes | Public image proxy URL (idx-images.quantyralabs.cc) |
| `IDX_PLATFORM_HOSTS` | Dev | Comma-separated allowed hosts for platform |
| `IDX_API_HOSTS` | Dev | Comma-separated allowed hosts for API |

### MLS (provider-agnostic)

Use **`MLS_*`** for mirror retention, replication scheduling, proxy-cache purge, and per-feed tuning (`MLS_STELLAR_*`, `MLS_BEACHES_*`). Keep **`BRIDGE_*`** / **`SPARK_*`** for upstream hosts, credentials, RESO paths, and sync queues. Consumers: `internal/service/sync`, `internal/service/search`, `internal/service/cache`.

| Variable | Required | Description |
|----------|----------|-------------|
| `MLS_LOCAL_MIRROR_ROLLING_MONTHS` | No | Active/Pending mirror window for purge, PostGIS search, listings-cache OData (default **12**; staging often **3**) |
| `MLS_REPLICA_PAGE_RETENTION_HOURS` | No | Completed replication staging page TTL (default 24) |
| `MLS_REPLICA_PAGE_FAILED_RETENTION_DAYS` | No | Failed staging page retention (default 7) |
| `MLS_REPLICATION_FRESHNESS_MINUTES` | No | Catch-up vs steady incremental threshold (default 15) |
| `MLS_LISTINGS_SYNC_*` | No | Listings collection cache pagination caps |
| `MLS_STELLAR_*` / `MLS_BEACHES_*` | No | Per-feed replication `$top`, persist/upsert chunks, rate limit, API key overrides |

Deprecated aliases (one release): `BRIDGE_LOCAL_MIRROR_ROLLING_MONTHS`, `SPARK_LOCAL_MIRROR_ROLLING_MONTHS`, `BRIDGE_REPLICA_PAGE_*`, `SPARK_REPLICA_PAGE_*`.

### Bridge MLS (platform)

| Variable | Required | Description |
|----------|----------|-------------|
| `BRIDGE_API_KEY` | Yes | Bridge Data Output API key |
| `BRIDGE_HOST` | Yes | Bridge API base URL (default: api.bridgedataoutput.com) |
| `BRIDGE_DATASET` | No | MLS dataset (default: `stellar`) |
| `BRIDGE_PATH_PREFIX` | No | e.g. `api/v2` |
| `BRIDGE_RESO_ROOT` | No | e.g. `reso/odata` |
| `BRIDGE_LISTING_PHOTO_PATH` | No | Path template for photos |
| `BRIDGE_IMAGE_REWRITE_HOSTS` | No | Extra hostnames for URL rewriting |
| `BRIDGE_TIMEOUT` | No | HTTP timeout (default: 30) |
| `LISTINGS_CACHE_TTL` | No | On-demand `mls_search_cache` TTL in seconds (default: 900) |
| `MLS_LOOKUP_CACHE_TTL` | No | Lookup partition TTL (default ~30 days) |
| `MLS_PROXY_CACHE_RETENTION_DAYS` | No | Purge horizon for stale proxy cache rows |
| `GOOGLE_MAPS_GEOCODING_API_KEY` | Home value | Address geocoding for `home_value` mode |
| `IMAGE_CACHE_PATH` | No | Image storage root (Docker: /var/cache/geoidx/images) |
| `IMAGE_CACHE_TTL` | No | Origin re-fetch TTL (default: 86400) |

### Spark MLS (Beaches — platform)

| Variable | Required | Description |
|----------|----------|-------------|
| `SPARK_ACCESS_TOKEN` | Yes (Beaches) | Bearer for Spark RESO replication and live proxy |
| `SPARK_API_FEED_ID` | No | Spark dashboard API Feed ID (audit/logging) |
| `SPARK_REPLICATION_HOST` | No | Replication OData host (default: `https://replication.sparkapi.com`) — sync only |
| `SPARK_REPLICATION_RESO_ROOT` | No | Replication RESO path (default: `Reso/OData`) |
| `SPARK_API_HOST` | No | Live API host (default: `https://sparkapi.com`) — proxy/search/images |
| `SPARK_API_VERSION` | No | Live API version prefix (default: `v1`) |
| `SPARK_LIVE_RESO_ROOT` | No | Live RESO path under version (default: `Reso/OData`) |
| `SPARK_RESO_BASE_URL` | No | Legacy override for replication base only |
| `SPARK_DATASETS` | No | Mirror/catalog slugs (default: `beaches`) |
| `SPARK_SYNC_FETCH_QUEUE` | No | Fetch queue (default: `spark-sync-fetch`) |
| `SPARK_SYNC_PERSIST_QUEUE` | No | Persist queue (default: `spark-sync-persist`) |

Catalog key `spark_beaches`; mirror partition `beaches`. See @docs/spark/README.md (integration, RESO reference, compliance).

### GIS Parcel Proxy

| Variable | Required | Description |
|----------|----------|-------------|
| `GIS_EDGE_CACHE_TTL` | No | GIS edge cache TTL (default: 900) |
| `GIS_ORIGIN_MAX_DAYS_PRIMARY` | No | Postgres origin max age for statewide (default: 90) |
| `GIS_ORIGIN_MAX_DAYS_COUNTY` | No | Postgres origin max age for county (default: 30) |
| `GIS_METADATA_TIMEOUT` | No | Metadata probe HTTP timeout (default: 12) |
| `GIS_QUEUE` | No | Queue for GIS jobs (default: default) |
| `GIS_QUEUE_BACKUP_WRITES` | No | Async filesystem backup (default: true) |
| `GIS_TEASER_MAX_FEATURES` | No | Feature cap for non-full-access (default: 40) |
| `GIS_TEASER_COORD_DECIMALS` | No | Coordinate precision for teaser (default: 4, ~11m) |
| `GIS_MAX_BBOX_SPAN_DEG` | No | Max bbox span to prevent abuse (default: 0.35) |
| `GIS_FLORIDA_MLS_CODES` | No | Comma-separated MLS codes (default: stellar) |

### Internal / Ops

| Variable | Required | Description |
|----------|----------|-------------|
| `IMAGE_CACHE_PATH` | Prod | Local NVMe path for image bytes (default under `/var/cache/geoidx/images`) |
| `SESSION_LIFETIME` | No | Dashboard session TTL (hours) |

## Available Commands

| Command | Description |
|---------|-------------|
| `make build` | Build `bin/api`, `bin/worker`, `bin/scheduler` |
| `make test` | `go test ./...` |
| `make fmt` | `gofmt` on `cmd/` and `internal/` |
| `make migrate` | Goose up (`GOOSE_DBSTRING` required) |
| `make seed-admin` | Bootstrap admin user from `ADMIN_SEED_*` |
| `make run-api` | `go run ./cmd/api` (:8000) |
| `make run-worker` | `go run ./cmd/worker` |
| `make run-scheduler` | `go run ./cmd/scheduler` |
| `docker compose -f docker-compose.dev.yml up --build` | API + idx-images edge in Compose |
| Dashboard PAT | Issue tokens from `/dashboard` (`idx:full`) or `POST /api/auth/token` |

**GIS probe:** enqueued as `gis.probe_sources` (scheduler Monday 06:30) or trigger via worker after manual enqueue.

## Docker Deployment

### Production and staging (Coolify / VPS)

Multi-stage **[`Dockerfile`](Dockerfile)** builds three binaries from Go 1.25:

| Target | Process | Port |
|--------|---------|------|
| `api` | HTTP server (`cmd/api`) | 8000 |
| `worker` | Queue consumer (`cmd/worker`) | — |
| `scheduler` | Cron dispatcher (`cmd/scheduler`) | — |

**idx-images:** **[`Dockerfile.idx-images`](Dockerfile.idx-images)** — Nginx 1.27 → upstream `idx-api:8000` for `/images/*`.

```bash
docker build -f Dockerfile --target api -t quantyra/idx-api:latest .
docker build -f Dockerfile --target worker -t quantyra/idx-api-worker:latest .
docker build -f Dockerfile --target scheduler -t quantyra/idx-api-scheduler:latest .
docker build -f Dockerfile.idx-images -t quantyra/idx-images:latest .
```

See **[docs/coolify-deployment.md](docs/coolify-deployment.md)** for four-app layout (web, worker, scheduler, idx-images), env, and resource hints.

### Development

```bash
docker compose -f docker-compose.dev.yml up --build
```

Run **worker** and **scheduler** on the host (or separate Coolify services) against the same `DB_*` as the API. See [README.md](README.md).

## Testing

- Package tests: `go test ./internal/...`
- Prefer `httptest` for HTTP handlers and upstream stubs
- After schema changes: `make migrate` on a throwaway database
- Smoke: replication kickoff at minute boundary when scheduler + worker are running

## Additional Resources

- @docs/INDEX.md — Documentation index
- @docs/spark/README.md — Spark Platform (BeachesMLS): integration, RESO reference, compliance
- @docs/database-migrations.md — Migration inventory, PostGIS, legacy drops
- @docs/coolify-deployment.md — Coolify production & staging (four apps, env, networking, resources)
- @docs/idx-api-bridge-proxy.md — Bridge proxy architecture, auth flow, cache strategy, image rewrite
- @docs/bridge-api-documentation.md — Bridge Data Output upstream API reference
- @docs/gis-api.md — GIS parcel/geometry proxy documentation
- @README.md — Project overview and Docker build instructions

## Skill Usage Guide

When working on tasks involving these technologies, invoke the corresponding skill from [`.cursor/skills/`](.cursor/skills/) (see the [skills index](.cursor/skills/README.md)):

| Skill | Invoke When |
|-------|-------------|
| **go** | Handlers, services, queue jobs, config, `cmd/*`, tests |
| postgresql | Goose migrations, PostGIS queries |
| docker | `Dockerfile`, Compose, Coolify deploy |
| frontend-design | Dashboard/marketing HTML + `internal/web/static` CSS |
| nginx | idx-images reverse proxy |
| crafting-empty-states | Dashboard empty states |
| designing-inapp-guidance | Onboarding copy and flows |
| inspecting-search-coverage | MLS/GIS search filters and docs |
| `.cursor/skills/_legacy/` (laravel, php, vite, tailwind, …) | **Do not use** for new backend work (pre–Go cutover) |

## Agent Notes (Go)

- Prefer `IDX_API_PUBLIC_URL` / `APP_URL` for absolute URLs; do not hardcode production hostnames in code.
- Add or update `*_test.go` for behavior changes; run `go test` on touched packages before finishing.
- Schema changes: edit `migrations/00001_initial.sql` for **fresh** databases only; do not add a new goose file unless upgrading existing deployments is required.
- Documentation: update `docs/` when API contracts change; only create new doc files when asked.
