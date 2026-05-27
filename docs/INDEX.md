# Quantyra GeoIDX — Documentation Index

Central index for **idx-api** (Go 1.25+). Implementation lives in `cmd/` and `internal/`; guides live under `docs/`.

---

## Quick links

| Document | Description |
|----------|-------------|
| [HTTP API overview](api.md) | Current route groups, auth model, and API surface summary |
| [Route reference (code-aligned)](routes-reference.md) | Full method/path/auth map from Fiber route registration |
| [Go cutover runbook](go-cutover.md) | Laravel → Go migration, queue purge, API key re-issue |
| [OpenAPI spec](yaak-api-collection.json) | OpenAPI 3.1 source for JSON/API endpoints (`/api`, `/api/v1`, `/images`, infra) |
| [Bridge / MLS API](bridge-api-documentation.md) | Bridge Data Output upstream reference |
| [Spark Platform (Beaches MLS)](spark/README.md) | Integration, RESO, compliance, fixtures |
| [Spark — idx-api integration](spark/idx-api-integration.md) | Replication, dual hosts, queues, hybrid search |
| [IDX-API Bridge proxy](idx-api-bridge-proxy.md) | Proxy auth, cache, mirror, search, images, env |
| [Comps API](comps-api.md) | `POST /api/v1/comps/run` (BPO, home value, investor modes) |
| [GIS API](gis-api.md) | Parcel/geometry proxy, teaser for `idx:access`-only PATs |
| [GIS sources](gis-sources.md) | County parcel REST catalog, FDOR/FDOT findings, MLS coverage, probes |
| [Database migrations](database-migrations.md) | Goose SQL, PostGIS, schema inventory |
| [Listings mirror](listings-mirror.md) | Payload split, `$expand`, replication kickoff gating, hybrid search merge |
| [FEMA flood enrichment](fema-flood-enrichment.md) | NFHL Layer 28 jobs, `fema_flood_zone_code`, FEMA-backed `low_risk_flood_zone_yn` |
| [Deployment & operations](deployment-operations.md) | Docker, queues, scheduler leader lock, migrations |
| [Coolify deployment](coolify-deployment.md) | Single-host and **multi-DC (NYC + ATL)** runbooks |
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

---

## Scheduled jobs (Go)

`cmd/scheduler` enqueues (workers in `cmd/worker` execute):

| Cron (approx.) | Queue job type | Purpose |
|----------------|----------------|---------|
| Every minute | `mls.replication_kickoff` on `sync-kickoff` | Bridge/Spark replication kickoff (deduped; see [listings-mirror](listings-mirror.md)) |
| Every 15 min | `mls.proxy_cache_purge` | Purge expired `mls_search_cache` rows |
| Every 10 min | `crypto.refresh_pricing` | CoinGecko snapshot refresh |
| Daily 03:05 | `mls.purge_closed_listings` | Closed + rolling-window trim on `listings` |
| Daily 04:15 | `mls.purge_replica_pages` | Stale `replica_pages` staging |
| Monday 06:30 | `gis.probe_sources` | ArcGIS metadata probe |

**Multi-DC:** deploy two schedulers only with PostgreSQL advisory lock (`SCHEDULER_LEADER_LOCK_ID`) — see [Coolify deployment §7](coolify-deployment.md#7-scheduler-cluster-leadership-required-for-2-schedulers).

---

## Dev commands

```bash
cp .env.example .env
export GOOSE_DBSTRING="postgres://..."
make migrate
make seed-admin
make run-api          # :8000
make run-worker       # WORKER_QUEUES (include sync-kickoff + fetch/persist queues)
make run-scheduler
go test ./...
```

Docker: `docker compose -f docker-compose.dev.yml up --build`
