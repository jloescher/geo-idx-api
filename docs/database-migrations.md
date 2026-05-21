# Database migrations (PostgreSQL)

Schema is managed with **[goose](https://github.com/pressly/goose)** SQL migrations in [`migrations/`](../migrations/).

---

## Migration inventory

| File | Tables / purpose |
|------|------------------|
| `00001_initial.sql` | **Single consolidated schema** for fresh databases (staging cutover and greenfield installs). Includes `users`, `sessions`, `cache`, `jobs`, `job_batches`, `failed_jobs`, `personal_access_tokens`, `user_invitations`, `domains`, `listings_cache` (legacy table; not used for Active/Pending pre-warm), `mls_search_cache` (on-demand live proxy cache), `mls_proxy_audit_logs` (**`cache_hit`** `VARCHAR(8)` for `HIT`/`MISS`), `gis_cache`, `gis_source_states`, `crypto_price_snapshots`, PostGIS `listings` (`raw_data` + `media`/`unit`/`room`/`open_house` JSONB + `custom_fields`; single `modification_timestamp`), `listing_sync_cursors` (`last_modification_timestamp`), `replica_pages` (`fetch_url`, `upstream_url`, `odata_query`, `batch_id`) |

There is **no** `00002_*.sql` — `cache_hit` on `mls_proxy_audit_logs` is part of `00001_initial.sql`.

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

## Fresh staging / greenfield

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

**Extension columns:** `flood_zone_code`, `low_risk_flood_zone_yn`, `estimated_total_monthly_fees` on `listings` (set at mirror persist from RESO; `low_risk_flood_zone_yn` derived from `flood_zone_code`).

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

**Multi-DC schedulers:** use the same primary DSN; leadership via `pg_try_advisory_lock` (`SCHEDULER_LEADER_LOCK_ID`) — not a migration concern, but both scheduler containers must reach the primary.

---

## Related docs

- [Listings mirror](listings-mirror.md)
- [Deployment & operations](deployment-operations.md)
- [Coolify deployment](coolify-deployment.md)
- [IDX-API Bridge proxy](idx-api-bridge-proxy.md)
