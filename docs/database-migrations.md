# Database migrations (PostgreSQL)

Schema is managed with **[goose](https://github.com/pressly/goose)** SQL migrations in [`migrations/`](../migrations/).

---

## Migration inventory

| File | Tables / purpose |
|------|------------------|
| `00001_initial.sql` | **Single consolidated schema** for fresh databases (staging cutover and greenfield installs). Includes `users`, `dashboard_sessions`, `jobs`, `job_batches`, `failed_jobs`, `personal_access_tokens`, `user_invitations`, `domains`, **`listings_cache`** (30-day per-listing Closed comps cache for `POST /api/v1/comps/run`), `mls_search_cache` (on-demand live proxy cache), `mls_proxy_audit_logs` (**`cache_hit`** `VARCHAR(8)` for `HIT`/`MISS`), `gis_cache`, `gis_source_states`, **`gis_parcels`**, **`gis_counties`** (`county_slug`, `mls_stellar`, `mls_beaches`), **`gis_cities`**, **`gis_zips`**, **`gis_parcel_sources`** (22-county MLS catalog mirror), `crypto_price_snapshots`, PostGIS `listings` (`raw_data` + `media`/`unit`/`room`/`open_house` JSONB + `custom_fields`; single `modification_timestamp`), `listing_sync_cursors` (`last_modification_timestamp`), `replica_pages` (`fetch_url`, `upstream_url`, `odata_query`, `batch_id`). **Removed (Go-only):** Laravel `sessions`, `password_reset_tokens`, `cache`, `cache_locks`. |

There is **no** `00002_*.sql` or `00003_*.sql` — `cache_hit` on `mls_proxy_audit_logs` and `dashboard_sessions` are part of `00001_initial.sql`.

**Note:** Laravel Telescope/Pulse tables are **not** included.

---

## Commands

```bash
# From project root — set DSN (match DB_* in .env)
export GOOSE_DBSTRING="postgres://user:pass@host:5432/dbname?sslmode=require"
make migrate

# Bootstrap admin (Argon2id password from ADMIN_SEED_*)
make seed-admin
```

Install goose on PATH (optional): `make migrate-install`

---

## Fresh staging / greenfield (GIS multi-county cutover)

**Do not** incrementally migrate old GIS rows. Drop and recreate the database, then apply only `00001`:

```bash
# Local / Patroni primary — adjust names and credentials
dropdb geoidxapi_staging && createdb geoidxapi_staging

export GOOSE_DBSTRING="postgres://..."   # match .env
make migrate
make seed-admin
```

Start processes (same DSN via `DB_*`):

```bash
export WORKER_QUEUES=default,sync-kickoff,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist
make run-worker    # include GIS_SYNC_QUEUE (default: default)
make run-scheduler # enqueues gis.initial_sync + MLS replication kickoff
make run-api
```

Verify GIS bootstrap (boundaries ~15 min; 22-county parcels 24–48h):

```bash
python3 scripts/probe-county-parcels.py
```

```sql
SELECT county, COUNT(*) FROM gis_parcels GROUP BY county ORDER BY county;
SELECT COUNT(*) FILTER (WHERE county IS NOT NULL) FROM gis_cities;
SELECT county_slug, enabled FROM gis_parcel_sources ORDER BY county_slug;
```

MLS listings mirror repopulates via normal replication after worker + scheduler start.

---

## Fresh staging / greenfield (general)

Use a **new database** (or drop all objects) and apply only `00001`:

```bash
export GOOSE_DBSTRING="postgres://..."
make migrate
make seed-admin
```

Then run **API**, **worker**, and **scheduler** against the same DSN (`DB_*` in `.env`).

---

## Existing databases (pre-merge)

If a database was created from an **older** `00001` **without** `mls_proxy_audit_logs.cache_hit`, either:

- **Rebuild** (recommended for staging): new DB + `make migrate`, or  
- **Patch once:**

```sql
ALTER TABLE mls_proxy_audit_logs ADD COLUMN IF NOT EXISTS cache_hit VARCHAR(8) NULL;
```

---

## PostGIS

`00001_initial.sql` runs `CREATE EXTENSION IF NOT EXISTS postgis`. On managed Postgres without superuser, create the extension once before migrating.

**Mirror scope:** replication stores **Active + Pending** in `listings`; **Closed** is fetched on-demand via live Bridge/Spark RESO.

**Extension columns:** `flood_zone_code` (MLS at persist), `fema_flood_zone_code` / `flood_zone_*` (FEMA NFHL jobs — [fema-flood-enrichment.md](fema-flood-enrichment.md)), `low_risk_flood_zone_yn` (FEMA-derived only), `estimated_total_monthly_fees` on `listings`.

**Go services:** `internal/service/sync` (mirror persist), `internal/service/search/postgis.go` (hybrid search).

**Payload layout:** expanded collections (`media`, `unit`, `room`, `open_house`), overflow in `custom_fields`, canonical `modification_timestamp`, cursor `last_modification_timestamp`. See [Listings mirror](listings-mirror.md).

---

## Queue table

Go workers expect job payloads:

```json
{"type":"bridge.fetch_page","args":{...}}
```

Purge legacy Laravel rows after cutover:

```sql
DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
DELETE FROM jobs WHERE payload LIKE '%mls.listings_cache_refresh%';
```

See [go-cutover.md](go-cutover.md).

### Existing DB cleanup (pre–fresh migrate)

Drop unused Laravel tables once (Go does not reference them):

```sql
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS cache_locks;
DROP TABLE IF EXISTS cache;
```

If `listings_cache` exists with the old shape, prefer **`dropdb` + `make migrate`** per cutover runbook, or `ALTER TABLE listings_cache` to add `close_date`, `latitude`, `longitude`, `close_price`, and convert timestamp columns to `TIMESTAMPTZ`.

**Multi-DC schedulers:** use the same primary DSN; leadership via `pg_try_advisory_lock` on a dedicated **`pgx.Connect`** session (`SCHEDULER_LEADER_LOCK_ID`) — see [Coolify §7](coolify-deployment.md#7-scheduler-cluster-leadership-required-for-2-schedulers). Not a migration concern, but both scheduler containers must reach the primary.

---

## Incremental migrations (00006–00008) + backfill order

When upgrading an **existing** production database (not greenfield `00001` only):

| Migration | Purpose | Backfill before next step |
|-----------|---------|---------------------------|
| `00006_listings_search_columns.sql` | Typed IDX/facet columns, `unparsed_address`, `public_remarks`, geocode audit cols, search indexes | → [listings field promote](production-data-backfill.md) |
| — | Deploy Go (persist + API use new columns) | |
| — | [GIS city/county expand](production-data-backfill.md) | `COUNT(*) … WHERE county IS NULL` → 0 |
| `00008_gis_cities_county_not_null.sql` | `gis_cities.county NOT NULL` | After expand verify |
| `00007_gis_trgm_autocomplete.sql` | `pg_trgm` indexes for GIS autocomplete | After 00008 |

Scripts: `docs/scripts/run_listings_field_promote_backfill.sh`, `docs/scripts/run_gis_cities_county_expand.sh`. DSN: `docs/scripts/.env.backfill.local` (gitignored; see `.env.backfill.local.example`).

---

## Related docs

- [Production data backfill](production-data-backfill.md)
- [Listings mirror](listings-mirror.md)
- [Deployment & operations](deployment-operations.md)
- [Coolify deployment](coolify-deployment.md)
- [IDX-API Bridge proxy](idx-api-bridge-proxy.md)
