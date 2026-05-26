# Go cutover runbook

## Pre-cutover

1. Deploy Go `api`, `worker`, and `scheduler` images against the **same** PostgreSQL database.
2. Run goose migrations: `goose -dir migrations postgres "$GOOSE_DBSTRING" up` (idempotent on existing Laravel-era schema).
3. Notify customers to **re-issue API keys** from `/dashboard` (Go uses SHA-256 token hashes; legacy Sanctum `id|secret` tokens are not accepted).

## Cutover

1. Point `idx-api` traffic to Go `api` service (port 8000).
2. Scale workers:
   - Fetch: `WORKER_QUEUES=default,bridge-sync-fetch,spark-sync-fetch`
   - Persist: `WORKER_QUEUES=bridge-sync-persist,spark-sync-persist`
3. Run scheduler container (`cmd/scheduler`).
4. Keep `idx-images` Nginx proxy unchanged (upstream Go API).

## Post-cutover

1. Purge leftover **Laravel** rows in `jobs` (Go expects `{"type":"bridge.fetch_page",...}`; PHP jobs show `CallQueuedHandler` and log `discarded legacy Laravel queue job`):
   ```sql
   DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
   DELETE FROM jobs WHERE payload LIKE '%mls.listings_cache_refresh%';
   ```
   Or truncate `jobs` on a disposable staging DB after cutover. The scheduler now enqueues **`mls.proxy_cache_purge`** (renamed from `mls.listings_cache_refresh`).
2. Verify `/healthz`, `/readyz`, `POST /api/v1/search`, `/images/*`.
3. Monitor replication lag via `GET /api/v1/bridge/stats`.

## Rollback

Rollback requires a prior Laravel/Octane deployment artifact; the current repository builds **Go-only** images (`Dockerfile` targets `api`, `worker`, `scheduler`). Fresh `00001_initial.sql` is **Go-only** (Laravel `sessions`, `cache`, `password_reset_tokens` removed).
