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
| `DB_*`, `DB_RW_DSN` | PostgreSQL primary (`DB_SSLMODE=require` for remote hosts) |
| `QUEUE_NOTIFY_CHANNEL` | `pg_notify` channel for worker wakeup (default `idx_jobs_wakeup`) |
| `BRIDGE_API_KEY`, `SPARK_ACCESS_TOKEN` | MLS upstream credentials |
| `SPARK_REPLICATION_HOST`, `SPARK_REPLICATION_RESO_ROOT` | Spark replication OData base (not live API host) |
| `WORKER_QUEUES` | Comma-separated queue names for `cmd/worker` — **per worker app** in production |
| `SCHEDULER_LEADER_LOCK_ID`, `SCHEDULER_STANDBY_POLL_SECONDS` | Cluster scheduler leadership (multi-DC) |
| `IDX_PLATFORM_URL`, `IDX_API_PUBLIC_URL`, `IDX_IMAGES_PUBLIC_URL` | Public URLs |
| `FEMA_ENRICH_QUEUE`, `FEMA_*` | NFHL flood enrichment — **default worker** — [fema-flood-enrichment.md](fema-flood-enrichment.md) |
| `GEOCODE_QUEUE`, `GOOGLE_MAPS_GEOCODING_API_KEY`, `GEOCODE_*` | Listings geocode backfill — **default worker** |
| `GIS_SYNC_QUEUE`, `GIS_QUEUE` | GIS parcel/boundary jobs (default `default`) |
| `ADMIN_SEED_*` | `make seed-admin` only (not read at runtime by API) |

**Worker queues:**

| Topology | `WORKER_QUEUES` |
|----------|-----------------|
| **Local / dev** (single worker) | `default,sync-kickoff,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist` |
| **Production split** | See [coolify-env-by-app.md](coolify-env-by-app.md) |

```env
MLS_SYNC_KICKOFF_QUEUE=sync-kickoff
# Dev only — production uses split workers:
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
- Job types: `internal/queue/payload.go` (`bridge.fetch_page`, `spark.persist_chunk`, `mls.replication_kickoff`, `mls.proxy_cache_purge`, `fema.flood_enrich_kickoff`, `fema.flood_enrich_batch`, `mls.geocode_listings_kickoff`, `mls.geocode_listings_batch`, …)
- **Default-queue worker** must list `FEMA_ENRICH_QUEUE` and `GEOCODE_QUEUE` (usually `default`) in `WORKER_QUEUES` and supply the matching API keys — see [coolify-env-by-app.md](coolify-env-by-app.md).
- Unknown/legacy payloads are discarded; purge old rows (see go-cutover)

**Topology:** one combined worker is fine for dev. Production catch-up: split **fetch** (`bridge-sync-fetch,spark-sync-fetch`) from **persist** (`bridge-sync-persist,spark-sync-persist`) and keep **`default`** on a small worker for kickoff/GIS/crypto. See [Coolify §2](coolify-deployment.md#2-worker-configuration).

**Replication:** kickoff must not stack replication fetches while a chain is active; only finalize enqueues the next page. Tune `MLS_REPLICATION_FRESHNESS_MINUTES`, `BRIDGE_SYNC_*`, `SPARK_SYNC_*`, and optional `MLS_BEACHES_PERSIST_CHUNK_SIZE` per [listings-mirror.md](listings-mirror.md).

Scale: four workers across two DCs share the same queues, or split fetch vs persist during replication catch-up.

---

## Scheduler

Process: `cmd/scheduler` (or `make run-scheduler` locally).

Enqueues periodic work: replication kickoff, **`mls.proxy_cache_purge`**, CoinGecko pricing, FEMA flood enrich, geocode backfill, GIS probe/refresh, replica/closed purges. **Requires workers** to execute jobs. Full cron table: [INDEX.md § Scheduled jobs](INDEX.md#scheduled-jobs-go).

**Single host:** one scheduler process is enough.

**Multi-DC (two schedulers):** uses **`pg_try_advisory_lock`** on a dedicated DB connection (`SCHEDULER_LEADER_LOCK_ID`, default `913374211`). Only the leader runs cron; the peer logs `scheduler standby`. See [Coolify §7](coolify-deployment.md#7-scheduler-cluster-leadership-required-for-2-schedulers).

| Log line | Meaning |
|----------|---------|
| `scheduler leader acquired` | This instance holds the lock and runs cron |
| `scheduler standby, waiting for leader lock` | Peer instance; safe to leave running for failover |

### Scheduler incident troubleshooting (NYC + ATL)

If the monitoring dashboard shows **Scheduler leader not detected** while API and workers are healthy:

1. **Confirm both scheduler apps exist and are Running** — not stopped, not built with target `worker` or `api`:
   - `idx-scheduler-nyc` on **re-db** (NYC)
   - `idx-scheduler-atl` on **re-node-02** (ATL)
2. **Shared env on both** (same Patroni primary): `DB_RW_DSN`, `DB_HOST`/`DB_PORT`, `SCHEDULER_LEADER_LOCK_ID=913374211`, `MLS_SYNC_KICKOFF_QUEUE=sync-kickoff`.
3. **Tailscale from ATL** — scheduler on re-node-02 must reach the Patroni primary (same DSN as NYC web).
4. **Logs** — one container: `scheduler leader acquired`; the other: `scheduler standby`. If neither appears, the process is not connected or is crash-looping (check `database` / `config` errors on startup).
5. **Read-only SQL on primary** (expect one granted advisory lock while leader is up):

```sql
SELECT pid, granted, classid, objid FROM pg_locks
WHERE locktype = 'advisory' AND classid = 0 AND objid = 913374211;
```

6. **Workers before schedulers on fresh deploy** — kickoff jobs need a worker with `sync-kickoff` in `WORKER_QUEUES` (worker 1 on each DC).

**Monitoring probe bug (fixed):** Older API builds called `pg_try_advisory_lock` on one pool connection and `pg_advisory_unlock` on another. Session locks leaked onto idle API pool connections and **prevented schedulers from acquiring leadership** (both stuck in standby). After deploying the fix, **restart NYC + ATL API containers** once to drop leaked locks on existing pool sessions. Schedulers use a dedicated acquired connection for lock + unlock ([`internal/scheduler/leader.go`](../internal/scheduler/leader.go)) — that path was always correct.

**Scheduler `DB_RW_DSN` and HAProxy :5000:** The leader holds a long-lived session advisory lock on a dedicated connection. HAProxy idle timeouts (~60–120s) can drop that TCP session and release the lock while cron still runs. Prefer **Patroni primary :5432** on Tailscale for `DB_RW_DSN` on scheduler apps, or append libpq keepalives (`keepalives=1&keepalives_idle=30&keepalives_interval=10&keepalives_count=5`). The scheduler binary also pings the leader connection every 30s.

**Post-deploy verification (read-only):** After restarting APIs and schedulers, confirm one row in `pg_locks` for `objid = 913374211`, Infrastructure tab shows `holder_pid` and recent `last_enqueue_at`, and Incidents clears the scheduler critical.

### Failed jobs incident (historical vs active)

The dashboard warns on **`failed_jobs` rows from the last 7 days**. After cutover, rows such as `mls.replication_resume` (unknown handler at failure time) or Spark HTTP 400 are often **historical**. Once workers register all job types, purge resolved rows on the primary:

```sql
-- Review first
SELECT queue, payload::jsonb->>'type' AS type, COUNT(*), MAX(failed_at) AS last_failed
FROM failed_jobs GROUP BY 1, 2 ORDER BY COUNT(*) DESC;

-- Example: remove pre-cutover replication_resume failures (run only after confirming handler is deployed)
DELETE FROM failed_jobs
WHERE queue = 'sync-kickoff'
  AND payload::jsonb->>'type' = 'mls.replication_resume';
```

Do not purge production queue data without an explicit ops decision.

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
| Listings missing `fema_flood_zone_code` after “enrichment” | See [fema-flood-enrichment.md § Interpreting gaps](fema-flood-enrichment.md#interpreting-missing-fema_flood_zone_code); confirm worker 1 has `FEMA_*` and `default` in `WORKER_QUEUES` |
| Geocode never runs | `GOOGLE_MAPS_GEOCODING_API_KEY` on default worker; `GEOCODE_QUEUE` in `WORKER_QUEUES` |

---

## Related

- [README.md](../README.md)
- [AGENTS.md](../AGENTS.md)
- [coolify-deployment.md](coolify-deployment.md)
- [coolify-env-by-app.md](coolify-env-by-app.md)
- [database-migrations.md](database-migrations.md)
