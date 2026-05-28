# Admin dashboard

Invite-only Fiber dashboard with section navigation for domains (onboarding, API keys, DNS verify) and live ops monitoring.

## Navigation

| Route | Section | Description |
|-------|---------|-------------|
| `/dashboard` | — | Redirects to `/dashboard/monitoring` |
| `/dashboard/monitoring` | Monitoring | Live ops metrics, 30s auto-refresh |
| `/dashboard/domains` | Domains | Add domain bundles, per-hostname API keys, TXT verify, delete |
| `/dashboard/invite` | Invite user | Admin only |

Legacy URLs redirect to Domains (query string preserved):

- `GET /dashboard/setup` → `/dashboard/domains`
- `GET /dashboard/api-keys` → `/dashboard/domains`

## Domains flow (add domain + verify)

1. Open **Domains** → fill hostname + MLS dataset checkboxes → **Add domain & verify DNS**
2. `POST /dashboard/domains` provisions in one PostgreSQL transaction:
   - Production domain (`example.com`)
   - Staging domain (`staging.example.com`, `parent_domain_id` → production)
   - One **Production** API token (`domain_id` → production row)
   - One **Staging** API token (`domain_id` → staging row)
   - `allowed_mls_datasets` JSON from checkbox selections; first checked feed becomes default `mls_dataset`
3. Redirect to `/dashboard/domains#verify` with a one-time session flash (tokens + TXT)
4. Publish TXT records at your DNS host
5. Click **Verify TXT** per hostname (`POST /dashboard/domains/:id/verify-txt`)

### API keys on the Domains page

- Each hostname shows a masked key (`idx_••••…`). **Show** / **Copy** work only when plaintext is in the current page flash (after provision or **Regenerate API key**).
- `POST /dashboard/domains/:id/regenerate-token` replaces the key for that domain (old key stops working immediately).
- `POST /dashboard/domains/:id/delete` removes the production domain; CASCADE deletes staging and both tokens.

DNS TXT verification is required before API auth (`domains.verification_status`). Bearer tokens with `domain_id` set must be used with `X-Domain-Slug` matching that domain.

### MLS dataset checkboxes

Catalog codes come from `mls.Resolver.Catalog()` (e.g. `bridge_stellar`, `spark_beaches`). At least one dataset must be selected.

## Monitoring

| Endpoint | Auth |
|----------|------|
| `GET /dashboard/monitoring` | Dashboard session (HTML) |
| `GET /dashboard/monitoring/data` | Dashboard session (JSON snapshot) |
| `GET /api/v1/admin/monitoring` | Same session middleware (JSON) |
| `POST /api/v1/admin/flood-enrich` | Enqueue FEMA NFHL flood enrichment ([fema-flood-enrichment.md](fema-flood-enrichment.md)) |
| `POST /api/v1/admin/gis/probe` | Admin: probe ArcGIS metadata (`{ "source_key": "optional" }`) |
| `POST /api/v1/admin/gis/sync` | Admin: enqueue parcel sync (`source_key`, `force`) |
| `GET /api/v1/admin/gis/sources` | Admin: list `gis_parcel_sources` + health |
| `POST/PUT/DELETE /api/v1/admin/gis/sources` | Admin: CRUD catalog rows |
| `POST /api/v1/admin/gis/sources/:source_key/upload` | Admin: shapefile/zip upload → `gis.shapefile_import` job |

Refresh: manual **Refresh** button + 30s interval (pauses when tab hidden). Sessions persist in PostgreSQL (`dashboard_sessions`); re-login after migration or cookie invalidation.

### Monitoring tabs

- **Overview**: system rollup, queue pressure, cache efficiency, activation counters.
- **Ingest & Sync**: listing freshness + lag by dataset and `replica_pages` pipeline state.
- **Queue & Jobs**: ready / reserved / scheduled roll-up, per-queue counts, **in-flight jobs** table (job id, type, state, age, stale flag), **active batches** (`job_batches`), ready vs reserved job-type breakdown, failed job hotspots.
- **Data Quality**: GIS layer freshness (parcels, cities, counties, zips) and source status.
- **Infrastructure**: scheduler advisory-lock leadership probe and infra health.
- **Integrations**: crypto/dependency freshness.
- **Incidents**: active warning/critical incidents generated from snapshot health checks.

### Metrics glossary

| Section | Fields | Notes |
|---------|--------|-------|
| **Listings** | total, active/pending, lag, freshness mode | Per `dataset_slug`; dashboard drill-down → **Ingest & Sync** tab (not `/api/v1/bridge/stats`, which requires API domain token auth) |
| **GIS** | parcels, cities, counties, zips, source states, layer freshness | Stale if parcel/zip sync &gt;35d or generation mismatch; `api_status` from `last_probe_ok` |
| **GIS ops (admin)** | Data Quality tab: Probe / Sync / Probe all | Requires `is_admin`; see [gis-sources.md](gis-sources.md) |
| **Crypto** | BTC/ETH/SOL USD + age | Stale if snapshot &gt;1h |
| **Cache** | 15m hit rate from `mls_proxy_audit_logs` | `cache_hit` stored as `HIT`/`MISS`; status `no_data` when no audits in window |
| **Queues** | `pending` (ready now), `scheduled` (delayed), `reserved`, `stale_reserved`, failed | Merges configured `WORKER_QUEUES` with zero rows when empty; reads primary pool |
| **In-flight jobs** | up to 25 rows from `jobs` | States: `ready`, `scheduled`, `reserved`; **stale** when reserved longer than half of `DB_QUEUE_RESERVATION_TIMEOUT` (min 10m) |
| **Active batches** | open `job_batches` with `pending_jobs` or `failed_jobs` | Shows persist batch drain (e.g. `spark-replica-persist:beaches`) |
| **Queues (empty)** | empty-state copy on in-flight / batches | Completed jobs are **deleted** from `jobs` on success — an empty table means idle workers, not “broken monitoring” |
| **Queue failures** | top failed job types + latest exception preview | Grouped from `failed_jobs` via `payload::jsonb->>'type'` |
| **Sync pipeline** | `replica_pages` counts by dataset/status | Empty table returns `by_status: []` and UI “sync idle”; stale when failed or pending &gt; 500 |
| **Infrastructure** | scheduler advisory lock probe (`SCHEDULER_LEADER_LOCK_ID`) | `leader_active: false` = no session holds the lock; infra status `critical` |
| **Activation** | domains, keys, verified, 30d audit traffic | Traffic proxies “first API call” setup step |

### Monitoring thresholds

| Signal | Threshold | Notes |
|--------|-----------|-------|
| Queue pending stale | &gt; 500 total pending | Matches dashboard.js `QUEUE_PENDING_STALE` |
| Replica pending stale | &gt; 500 per dataset/status row | Pending rows below threshold stay **healthy** |
| Stale reserved | `max(600s, DB_QUEUE_RESERVATION_TIMEOUT / 2)` | Aligns with worker reservation, not a fixed 10m |
| Failed jobs | any `failed_jobs` row | Opens warning incident; queue rollup status `stale` |
| Cache | 15m audit window | `UPPER(cache_hit)` for hit/miss counts |

### Scheduler leadership verification (ops)

The monitoring API observes `pg_locks` on the Patroni **primary** RW pool (it does **not** call `pg_try_advisory_lock` on the pool — that pattern leaked session locks and blocked real schedulers). `leader_active: true` means a granted advisory lock exists for `SCHEDULER_LEADER_LOCK_ID` (default `913374211`).

1. **Coolify:** `idx-api-scheduler` NYC + ATL apps **Running**; one log stream shows `scheduler leader acquired`, the other `scheduler standby`.
2. **Env:** Same `DB_RW_DSN` and `SCHEDULER_LEADER_LOCK_ID` on web and scheduler (see repo-root `temp/` paste templates — do not commit secrets).
3. **SQL (read-only on primary):**

```sql
SELECT pid, granted, objid
FROM pg_locks
WHERE locktype = 'advisory' AND objid = 913374211;
```

Expect one granted row while the leader process is up. If empty and monitoring shows critical, restart scheduler apps or fix DB connectivity before changing application code.

**Beaches `STALE` while replica pipeline is active:** Listing tiles use `catching_up` when `replica_pages` has `pending`/`processing` rows or `replication_in_progress` is true, even if `last_sync_finished_at` is older than `MLS_REPLICATION_FRESHNESS_MINUTES` (default 15). Lag seconds still reflect time since last finished sync. True `stale` means no active pipeline work and the mirror is outside the freshness window.

**`enqueue never` on the Infrastructure tile:** `last_enqueue_at` is `MAX(created_at)` from `jobs` for scheduler-owned types (`mls.replication_kickoff`, `mls.replication_resume`, `mls.proxy_cache_purge`, `crypto.refresh_pricing`). Successful jobs are **removed from `jobs` immediately** after workers finish them, so an empty or fast-draining queue shows no timestamp even while the scheduler is healthy and listings continue to update. Prefer `leader_active` + scheduler logs (`enqueued scheduled job`) over this field; the UI labels an empty queue as “none pending” when a leader is active.

### Optional failed_jobs hygiene

Historical rows (e.g. `mls.replication_resume` before workers registered the handler, Spark HTTP 400) remain in `failed_jobs` until manually removed. After confirming workers consume `sync-kickoff` and handlers are deployed, operators may delete or archive stale rows **only with explicit approval** — the dashboard will continue to report `total_failed` &gt; 0 until then.

### GIS freshness {#gis-freshness}

Parcels and ZIP tiles show a subtitle from `parcels_last_synced_at` / `zips_last_synced_at` (e.g. “Synced 2d ago” or “Never synced”) and an amber **stale** badge when the timestamp is older than 35 days. Parcel source rows also mark stale when stored generation does not match `gis_source_states.generation`.

County parcel sources, FDOR/FDOT upstream issues, and MLS coverage: [GIS sources](gis-sources.md).

## UI state matrix

| State | Monitoring | Domains |
|-------|------------|---------|
| Loading | Skeleton tiles while preserving previous tab content on refresh | — |
| Empty | Per-tab empty card when no records match available metrics | Empty state + add-domain form |
| Error | Alert + Retry; stale cached snapshot remains visible | `.form-error`, `verify_error`, `error` query |
| Success | Timestamp flash + tab-local render | Verify panel after provision; `verified=1` / `deleted=1` banners |
| Stale | Amber status chips + critical strip for cross-tab incidents | Pending badge on domain rows |

## Deprecated dashboard token endpoints

Manual token creation is disabled; use per-domain **Regenerate** on Domains:

- `POST /dashboard/api-tokens` → redirect with error
- `POST /dashboard/api-tokens/staging` → redirect with error
- `POST /dashboard/api-tokens/:id/revoke` → still supported (redirects to Domains)

## Schema (domain-scoped tokens)

Migration `00002_domain_token_scoping.sql` adds:

- `domains.parent_domain_id` (staging → production, `ON DELETE CASCADE`)
- `personal_access_tokens.domain_id` (`UNIQUE`, `ON DELETE CASCADE`)
