# Deployment & operations

**Go idx-api** — Docker, Coolify/Dokploy, PostgreSQL queue, goose migrations. See also **[Coolify deployment](coolify-deployment.md)** and **[go-cutover.md](go-cutover.md)**.

---

## Docker images

Build from project root (context `.`).

| Service | Dockerfile | Target | Port |
|---------|------------|--------|------|
| **idx-api (web)** | [`Dockerfile`](../Dockerfile) | `api` | **8000** |
| **idx-api (worker)** | same | `worker` | — |
| **idx-api (scheduler)** | same | `scheduler` | — |
| **idx-images** | [`Dockerfile.idx-images`](../Dockerfile.idx-images) | — | **8080** |

```bash
docker build -f Dockerfile --target api -t quantyra/idx-api:latest .
docker build -f Dockerfile --target worker -t quantyra/idx-api-worker:latest .
docker build -f Dockerfile --target scheduler -t quantyra/idx-api-scheduler:latest .
docker build -f Dockerfile.idx-images -t quantyra/idx-images:latest .
```

**Health:** `GET /healthz` (liveness), `GET /readyz` (Postgres + PostGIS).

**Image cache:** set `IMAGE_CACHE_PATH` (default `/var/cache/geoidx/images`); writable by container user.

---

## Environment (all replicas)

| Variable | Purpose |
|----------|---------|
| `DB_*` | PostgreSQL (`DB_SSLMODE=require` for remote hosts) |
| `BRIDGE_API_KEY`, `SPARK_ACCESS_TOKEN` | MLS upstream credentials |
| `WORKER_QUEUES` | Comma-separated queue names for `cmd/worker` |
| `IDX_PLATFORM_URL`, `IDX_API_PUBLIC_URL`, `IDX_IMAGES_PUBLIC_URL` | Public URLs |
| `ADMIN_SEED_*` | `make seed-admin` only (not read at runtime by API) |

**Worker queues (typical):**

```env
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

Bridge fetch/persist: `BRIDGE_SYNC_FETCH_QUEUE`, `BRIDGE_SYNC_PERSIST_QUEUE`  
Spark fetch/persist: `SPARK_SYNC_FETCH_QUEUE`, `SPARK_SYNC_PERSIST_QUEUE`

---

## Migrations & admin seed

```bash
export GOOSE_DBSTRING="postgres://..."
make migrate
make seed-admin
```

Run once per deploy when `migrations/` changes.

---

## Worker

Process: `/usr/local/bin/worker` (or `make run-worker` locally).

- Polls `jobs` with `FOR UPDATE SKIP LOCKED`
- Job types: `internal/queue/payload.go` (`bridge.fetch_page`, `spark.persist_chunk`, `mls.replication_kickoff`, …)
- Discard legacy Laravel payloads or purge `jobs` table (see go-cutover)

Scale: separate replicas for fetch vs persist during replication catch-up.

---

## Scheduler

Process: `/usr/local/bin/scheduler` (or `make run-scheduler`).

Enqueues periodic work (listings cache refresh, replication kickoff, GIS probe, crypto pricing). **Requires workers** to execute jobs.

---

## idx-images

Nginx proxies `/images/*` → idx-api:8000. Same image for staging and production.

---

## Local Compose

```bash
docker compose -f docker-compose.dev.yml up --build
./scripts/docker-dev.sh up-watch   # if using tunnel/watch helpers
```

---

## Troubleshooting

| Symptom | Check |
|---------|--------|
| `unknown job type type=""` | Laravel jobs in `jobs`; run purge SQL from go-cutover |
| Spark jobs not running | `WORKER_QUEUES` includes `spark-sync-fetch`, `spark-sync-persist` |
| Login fails after cutover | `make seed-admin`; passwords are Argon2id |
| API tokens rejected | Re-issue PATs from dashboard (SHA-256 storage) |
| 502 on `/images/*` | idx-images → idx-api network, port 8000 |

---

## Related

- [README.md](../README.md)
- [AGENTS.md](../AGENTS.md)
- [coolify-deployment.md](coolify-deployment.md)
