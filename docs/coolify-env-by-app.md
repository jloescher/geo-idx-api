# Coolify environment — per-app reference

Production uses **split workers** (fetch vs persist vs `default`) so Bridge backlog cannot starve Spark. Paste templates for Quantyra’s live stack live in **`temp/`** at the repo root (secrets — **do not commit**; add `temp/` to `.gitignore` if needed).

**Canonical variable list:** [`.env.example`](../.env.example) and [`internal/config/config.go`](../internal/config/config.go).

**Deploy runbook:** [coolify-deployment.md](coolify-deployment.md) · **Operations:** [deployment-operations.md](deployment-operations.md)

---

## App matrix

| Coolify app | Dockerfile target | `WORKER_QUEUES` | Consumes jobs on |
|-------------|-------------------|-----------------|------------------|
| **idx-api-web** | `api` | *(none — HTTP only)* | — |
| **idx-api-scheduler** | `scheduler` | *(none — enqueue only)* | — |
| **idx-api-worker 1** | `worker` | `default,sync-kickoff` | `default`, `sync-kickoff`, `GIS_SYNC_QUEUE`, `FEMA_ENRICH_QUEUE`, `GEOCODE_QUEUE`, `COINGECKO_QUEUE` |
| **idx-api-worker 2** | `worker` | `bridge-sync-fetch,spark-sync-fetch` | MLS fetch HTTP |
| **idx-api-worker 3–4** | `worker` | `bridge-sync-persist,spark-sync-persist` | MLS persist (Postgres) |

Multi-DC: replicate the same four worker roles per datacenter (NYC + ATL). All processes use the **Patroni primary** for `DB_*` / `DB_RW_DSN` and the shared `jobs` table.

**idx-images:** separate Dockerfile; no MLS/FEMA/geocode vars.

**idx-api-mcp:** dedicated Coolify app for the remote MCP (Streamable HTTP). Uses Dockerfile target `mcp-monitor`. Exposes port 3000. Supports both raw `mcp_...` keys (for Cursor / local) and OAuth 2.1 + PKCE (for Grok Web Custom Connectors). See `docs/mcp-monitoring.md`.

---

## Shared environment (all six apps)

Set on **web**, **scheduler**, and **every worker** unless noted.

| Group | Variables |
|-------|-----------|
| **App** | `APP_NAME`, `APP_ENV=production`, `APP_DEBUG=false`, `APP_PORT=8000`, `LOG_FORMAT=json` |
| **Database** | `DB_*`, `DB_SSLMODE=require`, `DB_RW_DSN`, `DB_QUEUE_TABLE`, `DB_QUEUE_RETRY_AFTER`, `DB_QUEUE_RESERVATION_TIMEOUT`, `QUEUE_NOTIFY_CHANNEL=idx_jobs_wakeup` |
| **Patroni / reads** | `PATRONI_ENDPOINTS`; API also: `DB_READONLY_BASE_DSN` (phase-2 read replicas) |
| **Public URLs** | `IDX_PLATFORM_URL`, `IDX_API_PUBLIC_URL`, `IDX_IMAGES_PUBLIC_URL`, `IDX_PLATFORM_HOSTS`, `IDX_API_HOSTS` |
| **Bridge** | `BRIDGE_API_KEY`, `BRIDGE_HOST`, `BRIDGE_DATASET`, `BRIDGE_DATASETS`, `BRIDGE_PATH_PREFIX`, `BRIDGE_RESO_ROOT`, `BRIDGE_TIMEOUT`, `LISTINGS_CACHE_TTL`, `MLS_PROXY_CACHE_RETENTION_DAYS`, `MLS_LOOKUP_CACHE_TTL`, sync queue names, `BRIDGE_SYNC_REPLICATION_TOP`, `BRIDGE_SYNC_INCREMENTAL_TOP`, persist/upsert chunks, rate limits |
| **Spark** | `SPARK_ACCESS_TOKEN`, `SPARK_REPLICATION_HOST`, `SPARK_REPLICATION_RESO_ROOT` (e.g. `Version/3/Reso/OData`), `SPARK_API_HOST`, `SPARK_API_VERSION`, `SPARK_LIVE_RESO_ROOT`, `SPARK_DATASETS`, `SPARK_TIMEOUT=120`, `SPARK_SYNC_REPLICATION_TOP`, `SPARK_SYNC_INCREMENTAL_TOP`, fetch/persist queues and chunks |
| **MLS mirror** | `MLS_LISTINGS_CACHE_RETENTION_DAYS`, `MLS_REPLICATION_FRESHNESS_MINUTES`, `MLS_LOCAL_MIRROR_ROLLING_MONTHS`, `MLS_STELLAR_PERSIST_CHUNK_SIZE`, `MLS_BEACHES_PERSIST_CHUNK_SIZE`, `MLS_PERSIST_CHUNK_TIMEOUT_SECONDS`, `MLS_REPLICA_PAGE_RETENTION_HOURS`, `MLS_REPLICA_PAGE_FAILED_RETENTION_DAYS`, `MLS_MIRROR_KEY_RECONCILE_RETRY_MINUTES`, `MLS_SYNC_KICKOFF_QUEUE` |
| **GIS** | `GIS_SYNC_PAGE_SIZE`, `GIS_SYNC_UPSERT_CHUNK`, `GIS_HTTP_TIMEOUT`, `GIS_SYNC_QUEUE`, `GIS_QUEUE`, `GIS_IMPORT_QUEUE`, `GIS_EDGE_CACHE_TTL`, `GIS_ORIGIN_MAX_DAYS_*`, `GIS_MAX_BBOX_SPAN_DEG`, `GIS_MAX_FEATURES`, `GIS_TEASER_*`, `GIS_FLORIDA_MLS_CODES`, `GIS_BOUNDARY_STALE_DAYS`, `GIS_IMPORT_PATH`, `GIS_IMPORT_MAX_BYTES` |
| **Images** | `IMAGE_CACHE_PATH`, `IMAGE_CACHE_TTL` |
| **Dashboard** | `CLOUDFLARE_TURNSTILE_*`, `SESSION_LIFETIME`, `IDX_INVITATION_TTL_HOURS` |

`ADMIN_SEED_*` is for **`make seed-admin` only** — not required on runtime API/worker/scheduler env.

---

## Role-specific additions

### Web (`api`)

- `BRIDGE_LISTING_PHOTO_PATH` (image URL rewrite)
- `SPARK_SYNC_MAX_REQUESTS_PER_SECOND`, `SPARK_SYNC_MAX_REQUESTS_PER_5MIN` (live Spark proxy throttling)
- `MLS_SYNC_RATE_LIMIT_RETRY_SECONDS`, `MLS_SYNC_TIMEOUT_RETRY_SECONDS`, `MLS_SYNC_RATE_LIMIT_MAX_ATTEMPTS`
- `COMPS_CLOSED_CACHE_DAYS`, `COMPS_CLOSED_CACHE_MIN_HITS`
- `COINGECKO_API_KEY` (if pricing endpoints read fresh quotes from worker-populated cache)
- `GIS_IMPORT_PATH=/var/cache/geoidx/gis-imports` — **writes** shapefile uploads from admin dashboard/API; mount a **shared volume** at this path (see below)

### Scheduler

- `SCHEDULER_LEADER_LOCK_ID=913374211`, `SCHEDULER_STANDBY_POLL_SECONDS=15`
- `MLS_REPLICATION_RESUME_STALL_MINUTES`, `MLS_REPLICATION_RESUME_CRON`
- `FEMA_ENRICH_QUEUE=default`, `GEOCODE_QUEUE=default` (cron targets — workers must consume these queues)

### Worker 1 (`default,sync-kickoff,gis-import`)

**Required** for background enrichment (not optional on a split stack):

```env
WORKER_QUEUES=default,sync-kickoff,gis-import
MLS_REPLICATION_RESUME_STALL_MINUTES=3
MLS_REPLICATION_RESUME_CRON=0 */2 * * * *
MLS_SYNC_RATE_LIMIT_RETRY_SECONDS=300
MLS_SYNC_TIMEOUT_RETRY_SECONDS=60
MLS_SYNC_RATE_LIMIT_MAX_ATTEMPTS=50

# FEMA NFHL — see fema-flood-enrichment.md
FEMA_ENRICH_QUEUE=default
FEMA_FLOOD_ENRICH_BATCH_SIZE=2000
FEMA_FLOOD_STALE_DAYS=30
FEMA_MAX_REQUESTS_PER_SECOND=8
FEMA_HTTP_TIMEOUT=15s

# Geocode backfill — see listings-mirror.md
GOOGLE_MAPS_GEOCODING_API_KEY=...
GEOCODING_TIMEOUT=12s
GEOCODE_QUEUE=default
GEOCODE_BATCH_SIZE=200
GEOCODE_MAX_REQUESTS_PER_SECOND=5

COINGECKO_API_KEY=...
COINGECKO_QUEUE=default

# GIS shapefile import — same GIS_IMPORT_PATH mount as idx-api-web
GIS_IMPORT_PATH=/var/cache/geoidx/gis-imports
GIS_IMPORT_MAX_BYTES=536870912
GIS_IMPORT_QUEUE=gis-import
```

Also runs: `mls.replication_kickoff` (via `sync-kickoff`), GIS sync on `GIS_SYNC_QUEUE`, `gis.shapefile_import` on **`GIS_IMPORT_QUEUE`** (`gis-import`), purge jobs, `crypto.refresh_pricing`.

**Shapefile queue:** Enqueue uploads to `GIS_IMPORT_QUEUE` (default `gis-import`). Only **idx-api-worker 1** should list `gis-import` in `WORKER_QUEUES` (both NYC and ATL replicas). Workers 2–4 omit `gis-import` so replication jobs stay isolated; both DC worker-1 instances may consume `gis-import` for redundancy — ensure **`DB_QUEUE_RESERVATION_TIMEOUT` matches on every replica** so one DC does not stale-release another’s long ogr2ogr run.

**Shapefile volume:** In Coolify, attach the **same** persistent volume to **idx-api-web** and **idx-api-worker 1** at `/var/cache/geoidx/gis-imports` (or your `GIS_IMPORT_PATH`). Redeploy **both** after Dockerfile worker changes (`gdal-tools`, directory permissions). Smoke-test the worker image: `make docker-gis-smoke`.

**Multi-server (re-db + re-node-02):** Coolify routes `upload.idx.quantyralabs.cc` to the **additional server** while worker 1 on the **primary** may consume import jobs. Per-host bind mounts are not shared — use **NFS over Tailscale** so both hosts see the same `/data/coolify/gis-imports`. Step-by-step: [gis-import-nfs-setup.md](gis-import-nfs-setup.md). Verify with `./scripts/verify-gis-import-nfs.sh` on each host.

### Worker 2 (fetch)

```env
WORKER_QUEUES=bridge-sync-fetch,spark-sync-fetch
SPARK_TIMEOUT=120
SPARK_SYNC_MAX_REQUESTS_PER_SECOND=4
SPARK_SYNC_MAX_REQUESTS_PER_5MIN=1200
MLS_SYNC_RATE_LIMIT_RETRY_SECONDS=300
MLS_SYNC_TIMEOUT_RETRY_SECONDS=60
MLS_SYNC_RATE_LIMIT_MAX_ATTEMPTS=50
```

Cluster rate limits use PostgreSQL `sync_rate_budget` — identical fetch env on **every** fetch worker in every DC.

### Workers 3–4 (persist)

```env
WORKER_QUEUES=bridge-sync-persist,spark-sync-persist
# MLS_PERSIST_CHUNK_TIMEOUT_SECONDS=900   # wall clock per persist chunk (default 900)
```

Need `DB_RW_DSN` and MLS persist chunk vars; **no** FEMA/geocode/Google keys unless you collapse roles into one worker.

**Redeploy persist workers** after changes to persist chunk handling, reservation release, or `RowsForChunk` — API-only deploys do not pick up worker job handlers.

---

## Dev / single-worker shortcut

Local and small staging can use one process:

```env
WORKER_QUEUES=default,sync-kickoff,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
```

Include FEMA, geocode, and CoinGecko vars on that same worker.

### idx-api-mcp (remote MCP for AI agents / Grok Web)

This is a separate Coolify application using Dockerfile target `mcp-monitor`.

**Core runtime variables (required for both raw keys and OAuth flow):**

```env
MCP_HTTP_ENABLED=true
MCP_PUBLIC_URL=https://mcp.quantyralabs.cc/mcp
OAUTH_AUTH_SERVER=https://idx.quantyralabs.cc
```

These power:
- Correct RFC 9728 Protected Resource Metadata (`/.well-known/oauth-protected-resource`)
- The `WWW-Authenticate: Bearer resource_metadata="..."` header on 401 challenges
- Routing unauthenticated clients into the OAuth 2.1 + PKCE consent flow on the main web app

**Important production note (June 2026):** A malformed PRM URL (`https://.well-known/oauth-protected-resource`) was caused by a string-splitting bug in `buildResourceMetadataURL()`, not by the `MCP_PUBLIC_URL` value itself. The helper now uses `net/url.Parse` to derive `scheme://host` safely. Use the live debug endpoint after deploy:

```
GET https://mcp.quantyralabs.cc/debug/oauth-config
```

Expected value with production env:

```json
{
  "process_mcp_public_url": "https://mcp.quantyralabs.cc/mcp",
  "produced_resource_metadata_url": "https://mcp.quantyralabs.cc/.well-known/oauth-protected-resource"
}
```

If `produced_resource_metadata_url` is still malformed, verify the app is running the latest image/revision and fully redeployed.

See `docs/mcp-monitoring.md` → "Production Gotchas & Live Debugging Tools" for full troubleshooting context.

---

## After updating Coolify env

1. Redeploy **worker 1** after adding `GOOGLE_MAPS_GEOCODING_API_KEY` or `FEMA_*`.
2. Redeploy **web + worker 1** after changing `GIS_IMPORT_PATH` or shapefile upload wiring.
3. Confirm logs: `fema flood enrich kickoff`, `geocode listings kickoff` (no SQL errors on `jobs`).
4. Optional manual kickoff: `POST /api/v1/admin/flood-enrich` or `POST /api/v1/admin/geocode/kickoff` (session auth) — [fema-flood-enrichment.md](fema-flood-enrichment.md), [listings-mirror.md](listings-mirror.md).

---

## Related

- [FEMA flood enrichment](fema-flood-enrichment.md) — column semantics and NFHL ops
- [Listings mirror](listings-mirror.md) — geocode jobs and replication
- [Production data backfill](production-data-backfill.md) — post-`00006` SQL on Patroni `:5432`
