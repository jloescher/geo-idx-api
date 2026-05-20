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
| Typed columns | `list_price`, `bedrooms_total`, geo (`coordinates`), `flood_zone_code`, `estimated_total_monthly_fees`, `standard_status`, etc. — populated at persist for search indexes |
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
| `SPARK_SYNC_EXPAND` | (alias) | Falls back to `MLS_SYNC_EXPAND` when unset |
| `BRIDGE_SYNC_EXPAND` | `Media,OpenHouses,Rooms,UnitTypes` | **Bridge** when `$select` mode (`BRIDGE_SYNC_FULL_PROPERTY=false`) |
| `BRIDGE_SYNC_FULL_PROPERTY` | `true` | When **true**, Bridge replication/incremental **omit `$expand`** — `Media` is already inline on the Property resource ([Bridge docs](https://bridgedataoutput.com/docs/platform/API/zg-data)). Invalid Spark-style `$expand=Unit,Room,OpenHouse` returns **HTTP 400** on Stellar. |

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
| **Ongoing** | Same as updates | `mls.replication_kickoff` (every minute) enqueues **incremental** when `last_sync_finished_at` is older than **`MLS_REPLICATION_FRESHNESS_MINUTES`** (default 15), even after the mirror is seeded. After replication completes, the worker chains one incremental job immediately. |

**OData datetime literal:** Bridge expects a **bare ISO-8601** instant (`BridgeModificationTimestamp gt 2025-05-20T04:51:45Z`). The `datetime'…'` wrapper returns **HTTP 400** on Stellar.

**Alternative (Bridge docs):** poll `/Property/replication` via the `next` link on a schedule. idx-api uses **`/Property` + `BridgeModificationTimestamp`** instead so incremental shares the same persist path and respects the 200-row `$skip` cap on the standard collection.

**Rolling mirror window:** when `MLS_LOCAL_MIRROR_ROLLING_MONTHS` > 0, **Spark** replication adds `ModificationTimestamp gt …` to the Active/Pending filter. **Bridge (Stellar)** `/replication` rejects timestamp predicates (HTTP 400) — only `(StandardStatus eq 'Active' or StandardStatus eq 'Pending')` is sent; older rows are removed by the daily purge job using `listings.modification_timestamp`.

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

`mls:purge_closed_listings` (daily):

- Always deletes **Closed** rows from `listings`
- When `MLS_LOCAL_MIRROR_ROLLING_MONTHS` > 0, also deletes Active/Pending rows with `modification_timestamp` older than the window (and stale `close_date`)

Default rolling months: **12** (local/dev), **3** (staging `APP_ENV`), **0** = all-time (production default).

---

## Code map (Go)

| Concern | Package / file |
|---------|----------------|
| Payload split, merge, custom_fields | `internal/service/mls/listing_payload.go` |
| Modification timestamp resolve | `internal/service/mls/modification_timestamp.go` |
| Build row for upsert | `internal/service/mls/listing_row.go` → `BuildListingRecord` |
| Mirror upsert | `internal/service/sync/listing_mirror.go` |
| Bridge fetch | `internal/service/sync/bridge_sync.go` |
| Spark fetch | `internal/service/sync/spark_sync.go` |
| Replication filters | `internal/service/sync/mirror_window.go` |
| PostGIS search read path | `internal/service/search/postgis.go` |
| Config | `internal/config/config.go` (`MLS.SyncExpand`, `Bridge.SyncExpand`) |

---

## Schema

Single migration: [`migrations/00001_initial.sql`](../migrations/00001_initial.sql). Fresh databases: `make migrate` then full re-replication to populate JSONB columns and canonical timestamps.
