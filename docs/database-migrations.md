# Database migrations (PostgreSQL)

Schema is managed with **[goose](https://github.com/pressly/goose)** SQL migrations in [`migrations/`](../migrations/).

---

## Migration inventory

| File | Tables / purpose |
|------|------------------|
| `00001_initial.sql` | Single consolidated schema: `users`, `sessions`, `cache`, `jobs`, `job_batches`, `failed_jobs`, `personal_access_tokens`, `user_invitations`, `domains`, `listings_cache`, `mls_search_cache`, `mls_proxy_audit_logs`, `gis_*`, `crypto_price_snapshots`, PostGIS `listings` (`raw_data` + `media`/`unit`/`room`/`open_house` JSONB + `custom_fields`; single `modification_timestamp`), `listing_sync_cursors` (`last_modification_timestamp`), `replica_pages` (`fetch_url`, `upstream_url`, `odata_query`, `batch_id`) |

**Note:** Laravel Telescope/Pulse tables are **not** included in the Go cutover migration.

---

## Commands

```bash
# From project root (.env DB_* or explicit DSN)
export GOOSE_DBSTRING="postgres://user:pass@host:5432/dbname?sslmode=require"
make migrate

# Bootstrap admin (Argon2id password from ADMIN_SEED_*)
make seed-admin
```

Install goose on PATH (optional): `make migrate-install`

---

## PostGIS

`00001_initial.sql` runs `CREATE EXTENSION IF NOT EXISTS postgis`. On managed Postgres without superuser, create the extension once before migrating.

**Mirror scope:** replication stores **Active + Pending** in `listings`; **Closed** is on-demand via live Bridge/Spark API.

**Extension columns:** `flood_zone_code`, `low_risk_flood_zone_yn`, `estimated_total_monthly_fees` on `listings` (set at mirror persist from RESO; `low_risk_flood_zone_yn` derived from `flood_zone_code`).

**Go services:** `internal/service/sync` (mirror persist), `internal/service/search/postgis.go` (hybrid search).

**Payload layout:** expanded collections (`media`, `unit`, `room`, `open_house`), overflow in `custom_fields`, canonical `modification_timestamp`, cursor `last_modification_timestamp`. See [Listings mirror](listings-mirror.md).

---

## Fresh / disposable databases

```bash
# Drop all tables manually or use a new database, then:
make migrate
make seed-admin
```

Use dedicated DB names for tests (`TEST_DATABASE_URL`).

---

## Queue table

Go workers expect job payloads:

```json
{"type":"bridge.fetch_page","args":{...}}
```

Purge legacy Laravel rows after cutover:

```sql
DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%';
```

See [go-cutover.md](go-cutover.md).

---

## Related docs

- [Listings mirror](listings-mirror.md)
- [Deployment & operations](deployment-operations.md)
- [Coolify deployment](coolify-deployment.md)
- [IDX-API Bridge proxy](idx-api-bridge-proxy.md)
