# Quantyra GeoIDX — Documentation Index

Central index for all documentation in this project. Implementation code lives in this repository root, and **reference and integration guides** live under `docs/`.

---

## Quick links

| Document | Description |
|----------|-------------|
| [idx-api HTTP API overview](api.md) | `/api/v1` and GIS entrypoints; how dashboard Sanctum keys relate to `domain.token`. |
| [OpenAPI spec document](yaak-api-collection.json) | Canonical OpenAPI `3.1.0` document used by Swagger UI (`/swagger`) and served at `/openapi.json`. |
| [Bridge / MLS API](bridge-api-documentation.md) | Bridge Data Output API reference (Stellar MLS proxy usage). |
| [Spark Beaches MLS](spark-api-documentation.md) | Spark RESO replication (`spark_beaches` / mirror `beaches`), dual-bound incremental sync, `spark-sync-*` queues, live RESO proxy, image rewrite. |
| [IDX-API Bridge proxy](idx-api-bridge-proxy.md) | Secured Bridge proxy: `/api/v1/*`, `?domain=`, auth (verified **domains** + Sanctum PATs with **`idx:access` or `idx:full`**, **dashboard API keys**), full MLS-shaped JSON for authenticated traffic, listings cache (15m), **search cache**, **PostGIS mirror** (Active/Pending replication + **`bridge-sync-fetch`** / **`bridge-sync-persist`** queues), **hybrid search** (AP local, Closed live Bridge, mixed merge), `/api/v1/bridge/stats`, listing pricing enrichment (`pricing` + `pricing_converted`), queued CoinGecko quote refresh, JSON photo URL rewrite to **idx-images**, CloudFront URL normalization, OData cursor pagination, `/images/*` streaming proxy + immutable CDN headers, audit, env, Docker. |
| [Comps API](comps-api.md) | `POST /api/v1/comps/run` for A–E comps, investor modes (`rent_hold_cashflow`, `flip_vs_hold`, `appraiser_simulation`), BPO mode (`bpo`) with market-derived adjustments, and **home value estimator** (`home_value`) with Google Maps geocoding, condition overlay, and market-scaled renovation credits. |
| [Database migrations](database-migrations.md) | Inventory of `database/migrations/`, PostGIS, legacy cleanup migration, deploy notes. |
| [Deployment & operations](deployment-operations.md) | Docker, docker-compose, Dokploy, migrations, queues, scheduling (non–Coolify-specific layout). |
| [Coolify deployment](coolify-deployment.md) | **Production and staging** on Coolify: four apps per env, Dockerfile targets, ports 8000/8080, PostgreSQL queue, env checklist, post-deploy, `idx-api` / `idx-images` networking, CPU/RAM (staging workers **768M** PHP / **1024 MB** container). |
| [Docker builds](../README.md) | Production (`Dockerfile.production`), staging (`Dockerfile.staging`), image edge (`Dockerfile.idx-images`) — project-root build context. |

---

## Project layout summary

| Path | Role |
|------|------|
| `app/`, `routes/`, `config/`, `database/` | Laravel 13 + Octane: **secured Bridge MLS proxy** (`/api/v1/*`, images) and supporting services. |
| `docs/` | Product, integration, deployment, and operations documentation. |
| `tests/` | Feature and unit test coverage for Bridge and platform flows. |
| `Dockerfile.production`, `Dockerfile.staging`, `Dockerfile.idx-images` | Production API, staging API (FrankenPHP/Octane + workers), and Nginx image edge (same idx-images image for staging and production). |

For a full product overview, see the root [README.md](../README.md). **Coolify:** [coolify-deployment.md](coolify-deployment.md). **Docker / Dokploy:** [deployment-operations.md](deployment-operations.md) and the root [README.md](../README.md).

## Dev run commands

- Docker dev up/watch: `./scripts/docker-dev.sh up-watch`
- Docker dev down: `./scripts/docker-dev.sh down`