# Quantyra GeoIDX — Documentation Index

Central index for **idx-api** (Go 1.25+). Implementation lives in `cmd/` and `internal/`; guides live under `docs/`.

---

## Quick links

| Document | Description |
|----------|-------------|
| [HTTP API overview](api.md) | `/api/v1`, GIS, auth (domain + PAT), dashboard tokens |
| [Go cutover runbook](go-cutover.md) | Laravel → Go migration, queue purge, API key re-issue |
| [OpenAPI spec](yaak-api-collection.json) | OpenAPI 3.1 document (update when routes change) |
| [Bridge / MLS API](bridge-api-documentation.md) | Bridge Data Output upstream reference |
| [Spark Platform (Beaches MLS)](spark/README.md) | Integration, RESO, compliance, fixtures |
| [Spark — idx-api integration](spark/idx-api-integration.md) | Replication, dual hosts, queues, hybrid search |
| [IDX-API Bridge proxy](idx-api-bridge-proxy.md) | Proxy auth, cache, mirror, search, images, env |
| [Comps API](comps-api.md) | `POST /api/v1/comps/run` |
| [GIS API](gis-api.md) | Parcel/geometry proxy |
| [Database migrations](database-migrations.md) | Goose SQL, PostGIS, schema inventory |
| [Deployment & operations](deployment-operations.md) | Docker, queues, scheduler, migrations |
| [Coolify deployment](coolify-deployment.md) | Production/staging: api, worker, scheduler, idx-images |
| [README](../README.md) | Local dev, `make` targets, build & test |

---

## Project layout

| Path | Role |
|------|------|
| `cmd/api`, `cmd/worker`, `cmd/scheduler`, `cmd/seed` | Binaries |
| `internal/` | Handlers, services, queue, repository, config |
| `migrations/` | Goose SQL schema |
| `internal/web/static/` | Embedded dashboard/marketing CSS/JS |
| `docs/` | Product and operations documentation |
| `Dockerfile` | Targets: `api`, `worker`, `scheduler` |
| `Dockerfile.idx-images` | Nginx edge for `/images/*` |

---

## Dev commands

```bash
cp .env.example .env
export GOOSE_DBSTRING="postgres://..."
make migrate
make seed-admin
make run-api          # :8000
make run-worker       # WORKER_QUEUES in .env
make run-scheduler
go test ./...
```

Docker: `docker compose -f docker-compose.dev.yml up --build`
