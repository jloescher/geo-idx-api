# Production data backfill (Patroni primary)

One-time SQL backfills for **existing** production databases after deploying IDX typed columns and multi-county `gis_cities`. Run against the **Patroni leader** on port **5432** over Tailscale — not HAProxy **:5000** (idle timeouts drop long sessions).

Shell runners live in [`docs/scripts/`](../docs/scripts/) and share DSN config via [`docs/scripts/.env.backfill.local.example`](../docs/scripts/.env.backfill.local.example) (copy to `.env.backfill.local`, gitignored).

---

## Prerequisites

| Requirement | Notes |
|-------------|--------|
| Patroni leader IP | `curl -s http://<any-patroni-node>:8008/cluster \| jq -r '.members[] \| select(.role=="leader") \| .host'` |
| Credentials | Same DB user/database as production (`geoidxapi`, etc.) |
| `psql` | Client on your laptop or jump host with Tailscale |
| Goose migrations | See [deploy order](#deploy-order) below |

### DSN setup

```bash
cp docs/scripts/.env.backfill.local.example docs/scripts/.env.backfill.local
# Edit: set BACKFILL_DSN to leader :5432 (not HAProxy :5000)
export BACKFILL_DSN='postgres://USER:PASS@100.x.x.x:5432/geoidxapi?sslmode=require&keepalives=1&keepalives_idle=20&keepalives_interval=5&keepalives_count=5'
```

Runners prefer `BACKFILL_DSN`, then `GOOSE_DBSTRING_DIRECT`, then `GOOSE_DBSTRING`.

---

## Deploy order (existing production)

| Step | Action |
|------|--------|
| 1 | `make migrate` through **`00006_listings_search_columns.sql`** (typed IDX/facet columns, geocode cols, indexes) |
| 2 | Deploy **Go** API/worker/scheduler (persist + public visibility use new columns) |
| 3 | **Listings field promote** — [`run_listings_field_promote_backfill.sh`](../docs/scripts/run_listings_field_promote_backfill.sh) |
| 4 | **GIS city–county expand** — [`run_gis_cities_county_expand.sh`](../docs/scripts/run_gis_cities_county_expand.sh) |
| 5 | Verify `SELECT COUNT(*) FROM gis_cities WHERE county IS NULL` → **0** |
| 6 | `make migrate` through **`00008_gis_cities_county_not_null.sql`** |
| 7 | `make migrate` through **`00007_gis_trgm_autocomplete.sql`** (GIS autocomplete indexes) |
| 8 | Set `GOOGLE_MAPS_GEOCODING_API_KEY` + geocode worker/scheduler env (see [listings-mirror.md](listings-mirror.md)) |

Fresh greenfield DBs: `make migrate` + replication populate typed columns; **skip** promote backfill unless upgrading legacy JSONB-only rows.

---

## 1. Listings field promote

**SQL:** [`docs/scripts/listings_field_promote_backfill.sql`](../docs/scripts/listings_field_promote_backfill.sql)  
**Runner:** [`docs/scripts/run_listings_field_promote_backfill.sh`](../docs/scripts/run_listings_field_promote_backfill.sh)

Promotes 14 RESO scalar/IDX keys plus `UnparsedAddress` / `PublicRemarks` from `custom_fields` / `raw_data` into typed columns; strips promoted keys from `custom_fields` and IDX keys from `raw_data` (scalars such as `GarageSpaces` remain in `raw_data` by design).

### Commands

```bash
cd /path/to/idx-api

# Sanity check
docs/scripts/run_listings_field_promote_backfill.sh check

# Foreground (first run installs procedures)
docs/scripts/run_listings_field_promote_backfill.sh 500 reconnect /tmp/listings_field_promote_backfill.log

# Background
nohup docs/scripts/run_listings_field_promote_backfill.sh 500 reconnect /tmp/listings_field_promote_backfill.log \
  >>/tmp/listings_field_promote_backfill.log 2>&1 &
tail -f /tmp/listings_field_promote_backfill.log

# Resume after SQL reinstall
SKIP_INSTALL=1 docs/scripts/run_listings_field_promote_backfill.sh 500 reconnect /tmp/listings_field_promote_backfill.log
```

| Arg | Default | Meaning |
|-----|---------|---------|
| `batch_size` | `500` | Rows per connection (`primary` then `scalars` phases) |
| `mode` | `reconnect` | One `psql` per batch (recommended on Tailscale) |
| `log` | `/tmp/listings_field_promote_backfill.log` | Log file path |

### Verify (fast — do not full-table `COUNT` on `listings_row_needs_field_promote_row`)

```sql
SELECT
  EXISTS (SELECT 1 FROM listings WHERE custom_fields ? 'GarageSpaces' LIMIT 1) AS cf_garage_left,
  EXISTS (SELECT 1 FROM listings WHERE custom_fields ? 'InternetEntireListingDisplayYN' LIMIT 1) AS cf_ield_left,
  EXISTS (SELECT 1 FROM listings WHERE custom_fields ? 'IDXParticipationYN' LIMIT 1) AS cf_idx_left,
  EXISTS (SELECT 1 FROM listings WHERE raw_data ? 'InternetEntireListingDisplayYN' LIMIT 1) AS raw_ield_left,
  EXISTS (SELECT 1 FROM listings WHERE raw_data ? 'IDXParticipationYN' LIMIT 1) AS raw_idx_left;
```

All should be **`f`**. The runner prints the same check at the end.

### Tuning (optional env)

| Variable | Default | Purpose |
|----------|---------|---------|
| `BACKFILL_BATCH_TIMEOUT_SEC` | `120` | Kill hung `psql` per batch |
| `BACKFILL_BATCH_PAUSE_SEC` | `2` on `:5000`, else `0` | Pause between successful batches |
| `BACKFILL_MAX_CONSECUTIVE_FAILURES` | `25` | Stop after repeated SSL errors |

### Phases

1. **primary** — strip/promote IDX keys from `custom_fields` and strippable `raw_data` keys.  
2. **scalars** — fill NULL typed columns when a **promotable** value exists in JSON (unparseable keys are skipped to avoid infinite loops).

---

## 2. GIS cities county expand

**SQL:** [`docs/scripts/gis_cities_county_expand.sql`](../docs/scripts/gis_cities_county_expand.sql)  
**Runner:** [`docs/scripts/run_gis_cities_county_expand.sh`](../docs/scripts/run_gis_cities_county_expand.sh)

Expands `gis_cities` to one row per `(city_name, county_slug)` using PostGIS intersection + nearest-county fallback. Required **before** `00008` NOT NULL on `gis_cities.county`.

### Commands

```bash
docs/scripts/run_gis_cities_county_expand.sh check

docs/scripts/run_gis_cities_county_expand.sh 5 reconnect /tmp/gis_cities_county_expand.log

# Background
nohup docs/scripts/run_gis_cities_county_expand.sh 5 reconnect /tmp/gis_cities_county_expand.log \
  >>/tmp/gis_cities_county_expand.log 2>&1 &
tail -f /tmp/gis_cities_county_expand.log
```

| Arg | Default | Meaning |
|-----|---------|---------|
| `cities_per_batch` | `5` | City/generation pairs per connection |
| `mode` | `reconnect` | Recommended |

### Verify

```sql
SELECT COUNT(*) FROM gis_cities WHERE county IS NULL;  -- expect 0

SELECT city_name, COUNT(DISTINCT county) AS counties
FROM gis_cities
WHERE lower(city_name) IN ('jacksonville', 'midway', 'four corners')
GROUP BY city_name ORDER BY 1;
```

### Keys islands edge case

If a few rows remain NULL after expand, use the emergency `UPDATE` at the bottom of `gis_cities_county_expand.sql`, then re-check the count before `00008`.

### Tuning

| Variable | Default | Purpose |
|----------|---------|---------|
| `GIS_BATCH_TIMEOUT_SEC` | `300` | Per-batch timeout (spatial joins) |
| `GIS_COMMIT_EVERY` | `5` | **monolithic** mode only |

Ongoing sync: new cities get county pairs from **`ExpandCityCountyPairs`** in Go (`internal/service/gis/sync_boundaries.go`) — see [gis-sources.md](gis-sources.md).

---

## Operational notes

| Topic | Guidance |
|-------|----------|
| **DataGrip / JDBC** | Avoid long `CALL` in IDE; use shell runners or `psql` |
| **One job at a time** | Runners use lock dirs under `/tmp/*.lock.d` |
| **Progress** | `run total` in logs is **cumulative UPDATE count**, not unique rows |
| **Patroni failover** | If leader changes mid-run, update `BACKFILL_DSN` and resume (`SKIP_INSTALL=1`) |
| **Monolithic mode** | Single long `CALL` — only on stable `:5432`, not HAProxy |

---

## Related docs

- [Listings mirror](listings-mirror.md) — payload split, geocode job, schema
- [GIS sources](gis-sources.md) — city–county pair design
- [Database migrations](database-migrations.md) — Goose commands
- [Deployment & operations](deployment-operations.md) — migrations on primary
- [Coolify deployment](coolify-deployment.md) — multi-DC Patroni
