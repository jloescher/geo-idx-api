# Listings mirror — payload layout and sync

How idx-api stores replicated **Active + Pending** listings in PostgreSQL (`listings`) and reassembles RESO Property JSON for mirror-backed API responses.

**Related:** [Database migrations](database-migrations.md), [IDX-API Bridge proxy](idx-api-bridge-proxy.md), [Spark integration](spark/idx-api-integration.md), [Bridge RESO Web API](bridge_interactive/reso_web_api.md).

---

## Scope

| In mirror | Not in mirror |
|-----------|----------------|
| **Active**, **Pending** (bulk replication + incremental) | **Closed** (live Bridge/Spark OData on demand) |
| Indexed scalars + JSONB payload columns | Full upstream row shape at rest (split across columns) |

Replication flow: scheduler kickoff → `bridge.fetch_page` / `spark.fetch_page` → `replica_pages` (gzip staging) → `bridge.persist_chunk` / `spark.persist_chunk` → `listings`.

---

## Storage layout (`listings`)

| Column | Contents |
|--------|----------|
| Typed columns | `list_price` (required on Active/Pending upsert), `bedrooms_total`, `living_area` (sq ft, `NUMERIC(12,2)`), geo (`coordinates`), `flood_zone_code`, `estimated_total_monthly_fees`, `standard_status`, etc. — populated at persist for search indexes |
| `mirror_persisted_at` | When idx-api last wrote the row to the mirror (`NOW()` on each upsert); used for rolling-window purge — not the MLS modification clock |
| `raw_data` | Slim RESO Property JSON: scalars and fields that map to typed columns; **no** expanded collections; **no** `@odata.*` keys |
| `media` | RESO `Media[]` when present |
| `unit` | RESO `Unit[]` (Spark) or `UnitTypes[]` (Bridge), normalized at persist |
| `room` | RESO `Room[]` (Spark) or `Rooms[]` (Bridge), normalized at persist |
| `open_house` | RESO `OpenHouse[]` (Spark) or `OpenHouses[]` (Bridge), normalized at persist |
| `custom_fields` | **All other** upstream keys not stored elsewhere (provider extensions, unmapped RESO fields). **Not** returned as a nested object on API responses — see [API responses](#api-responses-mirror-backed) |
| `modification_timestamp` | **Single** canonical modification time per row (search, purge, rolling window) |

There is **no** `bridge_modification_timestamp` column. Bridge vs Spark upstream field choice is resolved at sync time by `dataset_slug` (see [Modification timestamps](#modification-timestamps)).

---

## OData `$expand` by provider

Expand lists control **upstream fetch** and which keys are stripped into JSONB at persist.

| Variable | Default | Used for |
|----------|---------|----------|
| `MLS_SYNC_EXPAND` | `Media,Unit,Room,OpenHouse` | **Spark** replication/incremental (`replication.sparkapi.com`) |
| `MLS_SYNC_REPLICATION_EXPAND` | (optional) | **Spark replication only** — smaller `$expand` when set (incremental still uses `MLS_SYNC_EXPAND`) |
| `SPARK_SYNC_EXPAND` | (alias) | Falls back to `MLS_SYNC_EXPAND` when unset |
| `BRIDGE_SYNC_EXPAND` | `Media,OpenHouses,Rooms,UnitTypes` | **Bridge** when `$select` mode (`BRIDGE_SYNC_FULL_PROPERTY=false`) |
| `BRIDGE_SYNC_FULL_PROPERTY` | `true` | When **true**, `Media` is inline on Property; **`$expand=OpenHouses,Rooms,UnitTypes`** is still sent on **`/Property`** (incremental + nav hydrate). `/Property/replication` does not return expanded nav collections even with `$expand`. |
| `BRIDGE_SYNC_NAV_HYDRATE_AFTER_REPLICATION` | `true` | After replication completes, paginate **`/Property`** with nav `$expand` to backfill `unit` / `room` / `open_house` JSONB. |

**Stellar navigation names** (from `docs/bridge_interactive/stellar_metadata.xml`): `Media`, `OpenHouses`, `Rooms`, `UnitTypes` — not Spark’s `Unit` / `Room` / `OpenHouse`.

At persist, Bridge upstream keys are mapped to canonical JSONB columns (`Rooms` → `room`, etc.) in `internal/service/mls/listing_payload.go`.

---

## Modification timestamps

One stored value per listing; sync code picks the upstream RESO field by **`dataset_slug`**:

| `dataset_slug` | Stored `modification_timestamp` from | Incremental OData `$filter` / `$orderby` |
|----------------|--------------------------------------|------------------------------------------|
| `stellar` (Bridge) | `BridgeModificationTimestamp` if present, else `ModificationTimestamp` | `BridgeModificationTimestamp` (fallback to `ModificationTimestamp` on HTTP 400/501) |
| `beaches` (Spark) | `ModificationTimestamp` | `ModificationTimestamp` |

**Cursor:** `listing_sync_cursors.last_modification_timestamp` holds the high-water mark per dataset for incremental sync.

### After initial replication (Bridge / Stellar)

Per [Bridge RESO Web API](https://bridgedataoutput.com/docs/platform/API/zg-data) and `docs/bridge_interactive/reso_web_api.md`:

| Phase | Endpoint | Purpose |
|-------|----------|---------|
| **Seed** | `GET …/Property/replication` | Bulk Active/Pending; follow `Link: rel="next"` until complete. No `$orderby` / `$skip`. Status-only `$filter` on Stellar (no timestamp on `/replication`). |
| **Updates** | `GET …/Property` | Incremental: **Active/Pending** and **`BridgeModificationTimestamp gt {cursor}`**, **`$orderby=BridgeModificationTimestamp asc`**, `$top` ≤ 200, `$skip` for pages. Bridge documents this field as the correct change signal (not MLS `ModificationTimestamp`). |
| **Ongoing** | Same as updates | Scheduler enqueues `mls.replication_kickoff` on **`MLS_SYNC_KICKOFF_QUEUE`** (default `sync-kickoff`) every minute. Kickoff **does not** enqueue replication or incremental while `replication_in_progress`, `replication_next_url`, or a `pending`/`processing` `replica_pages` row exists — replication pages chain from **persist finalize** only. In **catch-up** (`Freshness` mode), kickoff skips incremental; in **steady** state, incremental runs when `last_sync_finished_at` is older than **`MLS_REPLICATION_FRESHNESS_MINUTES`** (default 15). After replication completes, finalize chains one incremental fetch immediately. |

**OData datetime literal:** Bridge expects a **bare ISO-8601** instant (`BridgeModificationTimestamp gt 2025-05-20T04:51:45Z`). The `datetime'…'` wrapper returns **HTTP 400** on Stellar.

**Alternative (Bridge docs):** poll `/Property/replication` via the `next` link on a schedule. idx-api uses **`/Property` + `BridgeModificationTimestamp`** instead so incremental shares the same persist path and respects the 200-row `$skip` cap on the standard collection.

**Rolling mirror window:** when `MLS_LOCAL_MIRROR_ROLLING_MONTHS` > 0, **Spark** replication adds `ModificationTimestamp gt …` to the Active/Pending filter. **Bridge (Stellar)** `/replication` rejects timestamp predicates (HTTP 400) — only `(StandardStatus eq 'Active' or StandardStatus eq 'Pending')` is sent; older rows are removed by the daily purge job using `listings.modification_timestamp`. Aligning Spark replication to Bridge-style status-only OData (purge-only rolling window) is a **product decision** — not enabled by default because page counts and upstream behavior differ.

---

## API responses (mirror-backed)

Clients expect one **flat RESO Property object** (same shape as upstream), not internal column names.

**`POST /api/v1/search`** (PostGIS leg) and future mirror-backed reads use `MergeMirrorListing` (`internal/service/mls/listing_payload.go`):

1. Start from `raw_data`
2. Reattach `Media`, `Unit`, `Room`, `OpenHouse` from JSONB columns when present
3. **Flat-merge** keys from `custom_fields` onto the root (`raw_data` wins on key collision)
4. Do **not** emit a top-level `"custom_fields"` property

**Live proxy** (`GET /api/v1/properties`, Bridge/Spark upstream) passes JSON through unchanged — no `custom_fields` column involved.

Example response fragment after merge:

```json
{
  "ListingKey": "stellar:abc",
  "ListPrice": 450000,
  "STELLAR_SomeExtension": "value",
  "Media": [ { "MediaKey": "...", "MediaURL": "..." } ]
}
```

---

## Purge and rolling window

Queue job **`mls.purge_closed_listings`** (scheduler daily cron):

- Always deletes **Closed** rows from `listings`
- When `MLS_LOCAL_MIRROR_ROLLING_MONTHS` > 0, also deletes Active/Pending rows whose effective age is older than the window: **`COALESCE(mirror_persisted_at, modification_timestamp) < cutoff`** (and stale `close_date`). Bulk Stellar replication can carry old `BridgeModificationTimestamp` on the row while `mirror_persisted_at` reflects when we persisted it — purge retention follows **mirror persist time**, not upstream modification alone.

Default rolling months: **12** (local/dev), **3** (staging `APP_ENV`), **0** = all-time (production default).

### RESO numeric normalization (persist)

Indexed numerics are clamped to PostgreSQL column bounds in `internal/service/mls/normalize.go` (full payloads remain in `raw_data` / `custom_fields`):

| Mirror column | Rule |
|---------------|------|
| `list_price` | `ListPrice` → `PreviousListPrice` → `OriginalListPrice`; clamp `NUMERIC(14,2)`; skip upsert if Active/Pending and all absent |
| `bathrooms_total_decimal` | Prefer `BathroomsTotalDecimal`; integer fallback only if **0–30**; clamp ≤ **99.99**; NULL for implausible values (e.g. `BathroomsTotalInteger=6602`) |
| `living_area` | `LivingArea` → `BuildingAreaTotal`; decimal sq ft, clamp `NUMERIC(12,2)` |
| Optional prices / lot | Clamp to column width; NULL when API omits field |

**Spark incremental OData:** `FetchIncrementalPage` uses **status-only** `(Active|Pending)` plus cursor `ModificationTimestamp gt/lt`. It does **not** wrap `activePendingReplicationBaseFilter` (which adds a second rolling `ModificationTimestamp gt` and caused HTTP 400). Rolling `ModificationTimestamp gt` applies on **replication seed** via `SparkReplicationFilter` only.

---

## Fresh database verification

After changing `migrations/00001_initial.sql`, reset to an **empty** database and apply schema once:

```bash
export GOOSE_DBSTRING="postgres://postgres:postgres@127.0.0.1:5432/idx_api?sslmode=disable"
make migrate
```

Start worker queues: `sync-kickoff,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist` (scheduler enqueues kickoff). Then confirm both feeds populated and replication cleared:

```sql
SELECT dataset_slug, COUNT(*), MIN(mirror_persisted_at), MAX(mirror_persisted_at)
FROM listings GROUP BY 1;

SELECT dataset_slug, replication_in_progress, last_sync_finished_at
FROM listing_sync_cursors;
```

Expect row counts to grow for `stellar` and `beaches`, `replication_in_progress` false after initial replication, and no new `failed_jobs` with numeric overflow (`22003`) or Spark incremental HTTP 400.

---

## Code map (Go)

| Concern | Package / file |
|---------|----------------|
| Payload split, merge, custom_fields | `internal/service/mls/listing_payload.go` |
| Modification timestamp resolve | `internal/service/mls/modification_timestamp.go` |
| Build row for upsert | `internal/service/mls/listing_row.go` → `BuildListingRecord` |
| RESO numeric clamp / list price | `internal/service/mls/normalize.go` |
| Mirror upsert | `internal/service/sync/listing_mirror.go` |
| Bridge fetch | `internal/service/sync/bridge_sync.go` |
| Spark fetch | `internal/service/sync/spark_sync.go` |
| Replication filters | `internal/service/sync/mirror_window.go` |
| PostGIS search read path | `internal/service/search/postgis.go` |
| Config | `internal/config/config.go` (`MLS.SyncExpand`, `Bridge.SyncExpand`) |

---

## Schema

Single migration: [`migrations/00001_initial.sql`](../migrations/00001_initial.sql). Fresh databases: `make migrate` then full re-replication to populate JSONB columns and canonical timestamps.
