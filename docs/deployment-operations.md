# Deployment & operations

**Go idx-api** — Docker, Coolify/Dokploy, PostgreSQL queue, goose migrations. See also **[Coolify deployment](coolify-deployment.md)** (including [multi-DC NYC + ATL](coolify-deployment.md#8-multi-dc-production-nyc--atl)) and **[go-cutover.md](go-cutover.md)**.

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

**Image cache:** set `IMAGE_CACHE_PATH` (default `/var/cache/geoidx/images`); writable by container user. Per-API instance in multi-DC (not shared).

---

## Environment (all replicas)

| Variable | Purpose |
|----------|---------|
| `DB_*` | PostgreSQL (`DB_SSLMODE=require` for remote hosts) |
| `BRIDGE_API_KEY`, `SPARK_ACCESS_TOKEN` | MLS upstream credentials |
| `WORKER_QUEUES` | Comma-separated queue names for `cmd/worker` |
| `SCHEDULER_LEADER_LOCK_ID`, `SCHEDULER_STANDBY_POLL_SECONDS` | Cluster scheduler leadership (multi-DC) |
| `IDX_PLATFORM_URL`, `IDX_API_PUBLIC_URL`, `IDX_IMAGES_PUBLIC_URL` | Public URLs |
| `ADMIN_SEED_*` | `make seed-admin` only (not read at runtime by API) |

**Worker queues (typical):**

```env
WORKER_QUEUES=default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

Bridge fetch/persist: `BRIDGE_SYNC_FETCH_QUEUE`, `BRIDGE_SYNC_PERSIST_QUEUE`  
Spark fetch/persist: `SPARK_SYNC_FETCH_QUEUE`, `SPARK_SYNC_PERSIST_QUEUE`  
Mirror payload / expand: `MLS_SYNC_EXPAND`, `BRIDGE_SYNC_EXPAND`, `BRIDGE_SYNC_FULL_PROPERTY` — see [listings-mirror.md](listings-mirror.md).

---

## Migrations & admin seed

```bash
export GOOSE_DBSTRING="postgres://..."
make migrate
make seed-admin
```

Run **once** per schema change against the shared database (Patroni primary in multi-DC). Do not run migrations from every replica on every deploy unless `migrations/` changed.

**Legacy queue purge** (after Laravel cutover):

```sql
DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
DELETE FROM jobs WHERE payload LIKE '%mls.listings_cache_refresh%';
```

---

## Worker

Process: `cmd/worker` (or `make run-worker` locally).

- Polls `jobs` with `FOR UPDATE SKIP LOCKED`
- Job types: `internal/queue/payload.go` (`bridge.fetch_page`, `spark.persist_chunk`, `mls.replication_kickoff`, `mls.proxy_cache_purge`, …)
- Unknown/legacy payloads are discarded; purge old rows (see go-cutover)

Scale: four workers across two DCs share the same queues, or split fetch vs persist during replication catch-up.

---

## Scheduler

Process: `cmd/scheduler` (or `make run-scheduler` locally).

Enqueues periodic work: replication kickoff, **`mls.proxy_cache_purge`**, CoinGecko pricing, GIS probe, replica/closed purges. **Requires workers** to execute jobs.

**Single host:** one scheduler process is enough.

**Multi-DC (two schedulers):** uses **`pg_try_advisory_lock`** on a dedicated DB connection (`SCHEDULER_LEADER_LOCK_ID`, default `913374211`). Only the leader runs cron; the peer logs `scheduler standby`. See [Coolify §7](coolify-deployment.md#7-scheduler-cluster-leadership-required-for-2-schedulers).

| Log line | Meaning |
|----------|---------|
| `scheduler leader acquired` | This instance holds the lock and runs cron |
| `scheduler standby, waiting for leader lock` | Peer instance; safe to leave running for failover |

---

## idx-images

Nginx proxies `/images/*` → **`idx-api:8000`** on the same Docker network. Set API container alias **`idx-api`** per host. Same image for staging and production.

---

## Multi-DC checklist

1. Tailscale on both Coolify servers → Patroni primary (`./scripts/verify-patroni-connectivity.sh`).
2. Ten Coolify apps (2× API, 4× worker, 2× scheduler, 2× idx-images) — [app matrix](coolify-deployment.md#coolify-app-matrix).
3. Shared env → **primary** DSN only (phase 1).
4. Cloudflare geo LB for `idx-api` and `idx-images` hostnames.
5. Start order: workers → schedulers (verify one leader) → APIs → idx-images.

---

## Local Compose

```bash
docker compose -f docker-compose.dev.yml up --build
./scripts/docker-dev.sh up-watch   # optional tunnel/watch helpers
```

---

## Troubleshooting

| Symptom | Check |
|---------|--------|
| `unknown job type` / empty `type` | Legacy Laravel rows in `jobs`; purge SQL above |
| Duplicate replication kickoff every minute | Two schedulers without advisory lock; check leader logs |
| Spark jobs not running | `WORKER_QUEUES` includes `spark-sync-fetch`, `spark-sync-persist` |
| Login fails after cutover | `make seed-admin`; passwords are Argon2id |
| API tokens rejected | Re-issue PATs from dashboard (SHA-256 storage; not legacy `id\|secret`) |
| 502 on `/images/*` | idx-images → `idx-api` network alias, port 8000 |
| `readyz` fails from ATL | Patroni/Tailscale latency or PostGIS extension on DB |

---

## Related

- [README.md](../README.md)
- [AGENTS.md](../AGENTS.md)
- [coolify-deployment.md](coolify-deployment.md)
- [database-migrations.md](database-migrations.md)
