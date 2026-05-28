# Quantyra GeoIDX — Documentation Index

Central index for **idx-api** (Go 1.25+). Implementation lives in `cmd/` and `internal/`; guides live under `docs/`.

---

## Quick links

| Document | Description |
|----------|-------------|
| [HTTP API overview](api.md) | Current route groups, auth model, and API surface summary |
| [Route reference (code-aligned)](routes-reference.md) | Full method/path/auth map from Fiber route registration |
| [Go cutover runbook](go-cutover.md) | Laravel → Go migration, queue purge, API key re-issue |
| [OpenAPI spec](yaak-api-collection.json) | OpenAPI 3.1 source for JSON/API endpoints; served at `/openapi.json`, explorer at `/swagger` |
| [Swagger UI testing](swagger-ui-testing.md) | Manual test steps for `/swagger`, auth, GIS autocomplete, and search |
| [Bridge / MLS API](bridge-api-documentation.md) | Bridge Data Output upstream reference |
| [Spark Platform (Beaches MLS)](spark/README.md) | Integration, RESO, compliance, fixtures |
| [Spark — idx-api integration](spark/idx-api-integration.md) | Replication, dual hosts, queues, hybrid search |
| [IDX-API Bridge proxy](idx-api-bridge-proxy.md) | Proxy auth, cache, mirror, search, images, env |
| [Comps API](comps-api.md) | `POST /api/v1/comps/run` (BPO, home value, investor modes) |
| [GIS API](gis-api.md) | Parcel/geometry proxy, teaser for `idx:access`-only PATs |
| [GIS sources](gis-sources.md) | County parcel REST catalog, FDOR/FDOT findings, MLS coverage, probes |
| [Database migrations](database-migrations.md) | Goose SQL, PostGIS, schema inventory |
| [Production data backfill](production-data-backfill.md) | Patroni scripts: listings field promote + GIS city/county expand |
| [Listings mirror](listings-mirror.md) | Payload split, `$expand`, replication kickoff gating, hybrid search merge |
| [FEMA flood enrichment](fema-flood-enrichment.md) | NFHL Layer 28 jobs, `fema_flood_zone_code`, FEMA-backed `low_risk_flood_zone_yn` |
| [Deployment & operations](deployment-operations.md) | Docker, queues, scheduler leader lock, migrations |
| [Coolify deployment](coolify-deployment.md) | Single-host and **multi-DC (NYC + ATL)** runbooks |
| [Coolify env by app](coolify-env-by-app.md) | Production worker split, shared vs role-specific variables |
| [README](../README.md) | Local dev, `make` targets, build & test |

---

## Project layout

| Path | Role |
|------|------|
| `cmd/api`, `cmd/worker`, `cmd/scheduler`, `cmd/seed` | Binaries |
| `internal/` | Handlers, services, queue, repository, config |
| `migrations/` | Goose SQL schema |
| `internal/web/static/` | Embedded dashboard/marketing assets |
| `docs/` | Product and operations documentation |
| `Dockerfile` | Targets: `api`, `worker`, `scheduler` |
| `Dockerfile.idx-images` | Nginx edge for `/images/*` |
| `scripts/verify-patroni-connectivity.sh` | Multi-DC DB smoke (`psql` + optional `/readyz`) |
| `docs/scripts/run_listings_field_promote_backfill.sh` | Listings IDX/facet column backfill (Patroni `:5432`) |
| `docs/scripts/run_gis_cities_county_expand.sh` | GIS multi-county `gis_cities` expand before migration 00008 |

---

## Scheduled jobs (Go)

`cmd/scheduler` enqueues (workers in `cmd/worker` execute). Cron uses **seconds** (`robfig/cron/v3`). Leader-only when two schedulers run — see [Coolify §7](coolify-deployment.md#7-scheduler-cluster-leadership-required-for-2-schedulers).

| Cron (UTC) | Job type | Queue (typical) | Purpose |
|------------|----------|-----------------|---------|
| Every minute `:00` | `mls.replication_kickoff` | `sync-kickoff` | Bridge/Spark replication kickoff (deduped; [listings-mirror](listings-mirror.md)) |
| `MLS_REPLICATION_RESUME_CRON` (default `0 */2 * * * *`) | `mls.replication_resume` | `sync-kickoff` | Stalled replication resume |
| Every 10 min | `crypto.refresh_pricing` | `COINGECKO_QUEUE` (`default`) | CoinGecko snapshot |
| Every 15 min | `mls.proxy_cache_purge` | `default` | Purge expired `mls_search_cache` |
| Daily 03:05 | `mls.purge_closed_listings` | `default` | Closed + rolling-window trim |
| Daily 04:00 | `mls.mirror_key_reconcile` | `sync-kickoff` | Mirror key reconciliation |
| Daily 04:15 | `mls.purge_replica_pages` | `default` | Stale `replica_pages` staging |
| Daily 04:30 | `fema.flood_enrich_kickoff` | `FEMA_ENRICH_QUEUE` (`default`) | FEMA NFHL flood zone backfill ([fema-flood-enrichment](fema-flood-enrichment.md)) |
| Daily 05:15 | `mls.geocode_listings_kickoff` | `GEOCODE_QUEUE` (`default`) | Listings geocode backfill ([listings-mirror](listings-mirror.md)) |
| Monday 06:30 | `gis.probe_sources` | `GIS_QUEUE` | ArcGIS metadata probe |
| 1st of month 02:00 | `gis.monthly_parcel_refresh` | `GIS_SYNC_QUEUE` | Parcel layer refresh |
| Jan 1 03:00 | `gis.annual_boundaries_refresh` | `GIS_SYNC_QUEUE` | Boundary layer refresh |
| Every 6 h at `:15` | GIS bootstrap actions | `GIS_SYNC_QUEUE` | Enqueue missing parcel/boundary sync when layers empty |

**Multi-DC:** deploy two schedulers only with PostgreSQL advisory lock (`SCHEDULER_LEADER_LOCK_ID`) — see [Coolify deployment §7](coolify-deployment.md#7-scheduler-cluster-leadership-required-for-2-schedulers).

---

## Dev commands

```bash
cp .env.example .env
export GOOSE_DBSTRING="postgres://..."
make migrate
make seed-admin
make run-api          # :8000 — Swagger UI at /swagger
make openapi-sync     # after editing docs/yaak-api-collection.json
make run-worker       # WORKER_QUEUES — see coolify-env-by-app.md for production split
make run-scheduler
go test ./...
```

Docker: `docker compose -f docker-compose.dev.yml up --build`
