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
MLS_SYNC_KICKOFF_QUEUE=sync-kickoff
WORKER_QUEUES=default,sync-kickoff,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
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

**One-time data backfills** (after `00006`, before `00008`): use Patroni leader **:5432**, not HAProxy `:5000`. See [production-data-backfill.md](production-data-backfill.md) for listings field promote and GIS city/county expand runners under `docs/scripts/`.

**Legacy queue purge** (after Laravel cutover):

```sql
DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
DELETE FROM jobs WHERE payload LIKE '%mls.listings_cache_refresh%';
```

---

## Worker

Process: `cmd/worker` (or `make run-worker` locally).

- Polls `jobs` with `FOR UPDATE SKIP LOCKED`; **fair queue rotation** when `WORKER_QUEUES` lists multiple names (Bridge vs Spark fetch parity). When both fetch and persist queues are configured, workers **alternate pools** (fetch vs persist) so one pool cannot starve the other.
- `mls.replication_kickoff` runs on **`MLS_SYNC_KICKOFF_QUEUE`** (default `sync-kickoff`) — include that queue on the default/kickoff worker, not only `default`.
- Job types: `internal/queue/payload.go` (`bridge.fetch_page`, `spark.persist_chunk`, `mls.replication_kickoff`, `mls.proxy_cache_purge`, `mls.geocode_listings_kickoff`, `mls.geocode_listings_batch`, …)
- Unknown/legacy payloads are discarded; purge old rows (see go-cutover)

**Topology:** one combined worker is fine for dev. Production catch-up: split **fetch** (`bridge-sync-fetch,spark-sync-fetch`) from **persist** (`bridge-sync-persist,spark-sync-persist`) and keep **`default`** on a small worker for kickoff/GIS/crypto. See [Coolify §2](coolify-deployment.md#2-worker-configuration).

**Replication:** kickoff must not stack replication fetches while a chain is active; only finalize enqueues the next page. Tune `MLS_REPLICATION_FRESHNESS_MINUTES`, `BRIDGE_SYNC_*`, `SPARK_SYNC_*`, and optional `MLS_BEACHES_PERSIST_CHUNK_SIZE` per [listings-mirror.md](listings-mirror.md).

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

## Replication monitoring (catch-up)

Run against the Patroni primary during heavy replication:

**Queue depth by queue**

```sql
SELECT queue, COUNT(*) AS pending
FROM jobs
WHERE reserved_at IS NULL AND available_at <= EXTRACT(EPOCH FROM NOW())::bigint
GROUP BY queue
ORDER BY pending DESC;
```

**Duplicate kickoff backlog**

```sql
SELECT queue, payload->>'type' AS job_type, COUNT(*) AS pending
FROM jobs
WHERE reserved_at IS NULL
GROUP BY 1, 2
HAVING COUNT(*) > 5
ORDER BY pending DESC;
```

**Active replica pages (expect ≤1 pending/processing per provider+dataset)**

```sql
SELECT provider, dataset_slug, status, COUNT(*)
FROM replica_pages
WHERE status IN ('pending', 'processing')
GROUP BY 1, 2, 3;
```

**Cursor state**

```sql
SELECT dataset_slug, replication_in_progress, replication_next_url IS NOT NULL AS has_next, last_sync_finished_at
FROM listing_sync_cursors
ORDER BY dataset_slug;
```

**Maintenance window** (after large catch-up): `VACUUM (ANALYZE) listings;` and `VACUUM (ANALYZE) replica_pages;`

**Local smoke:** `make run-scheduler` + `make run-worker` with both feeds enabled; logs should show interleaved `enqueued fetch` for `stellar` and `beaches`, not long Bridge-only bursts. API: `GET /api/v1/bridge/stats` for `replication_in_progress` per dataset.

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
| Bridge dominates logs, Spark idle | Combined worker + global job order; use fair multi-queue worker or split fetch workers; confirm kickoff not enqueueing during `replication_in_progress` |
| Spark jobs not running | `WORKER_QUEUES` includes `spark-sync-fetch`, `spark-sync-persist` |
| Login fails after cutover | `make seed-admin`; passwords are Argon2id |
| API tokens rejected | Re-issue PATs from dashboard (SHA-256 storage; not legacy `id\|secret`) |
| 502 on `/images/*` | idx-images → `idx-api` network alias, port 8000 |
| `readyz` fails from ATL | Patroni/Tailscale latency or PostGIS extension on DB |
| `geocode enrich kickoff` … `column "finished_at" does not exist` | Old worker binary; redeploy worker with fix that dedupes geocode jobs via `jobs.reserved_at` (see [listings-mirror.md](listings-mirror.md#schema)) |
| `/swagger` shows “No layout defined for StandaloneLayout” | Blocked or missing `swagger-ui-standalone-preset.js` from unpkg; redeploy API; see [swagger-ui-testing.md](swagger-ui-testing.md) |

---

## Related

- [README.md](../README.md)
- [AGENTS.md](../AGENTS.md)
- [coolify-deployment.md](coolify-deployment.md)
- [database-migrations.md](database-migrations.md)
