# GIS data sources (Florida parcels and boundaries)

Operator reference for **where idx-api gets GIS data**, known upstream failures, MLS county coverage, and verification probes. Implementation lives in `internal/service/gis/` (`parcel_sources.go`, `sync_parcels.go`, `sync_boundaries.go`, `field_extract.go`).

**Fresh DB deploy:** Multi-county GIS schema ships in consolidated `migrations/00001_initial.sql` only — drop/recreate DB, `make migrate`, then bootstrap sync. See [Database migrations § GIS multi-county cutover](database-migrations.md#fresh-staging--greenfield-gis-multi-county-cutover).

**Catalog mirror:** Go source of truth is `ParcelSourceCatalog()` in `parcel_sources.go`; ops table `gis_parcel_sources` is upserted on parcel kickoff for monitoring and `gis-enqueue -county` workflows.

**Related:** [GIS API](gis-api.md) (HTTP surface), [Deployment & operations](deployment-operations.md) (queues/workers).

---

## MLS county coverage

idx-api serves listings from two Florida MLS partitions. Parcel sync should eventually cover all counties in these markets.

### Stellar (`?dataset=stellar` / Bridge)

Central and west Florida (~17 core counties):

| County | Shareholder / region (Stellar MLS) |
|--------|-----------------------------------|
| Alachua | Gainesville-Alachua (GACAR) |
| Charlotte | Punta Gorda / Port Charlotte / DeSoto (PGPCNP), Englewood (EABOR) |
| DeSoto | PGPCNP |
| Flagler | Flagler County Association of REALTORS |
| Hillsborough | Tampa Bay / Suncoast (STAR, Greater Tampa) |
| Lake | Lake & Sumter (RALSC) |
| Manatee | Sarasota & Manatee (RASM) |
| Marion | Ocala-Marion (OMCAR) |
| Okeechobee | Okeechobee County Board of REALTORS |
| Orange | Orlando Regional (ORRA) |
| Osceola | Osceola County Association of REALTORS |
| Pasco | West Pasco + Central Pasco (Tampa Bay) |
| Pinellas | Pinellas REALTOR Organization (Tampa Bay) |
| Polk | Bartow, East Polk, Lakeland |
| Sarasota | RASM, Venice (VABR) |
| Sumter | RALSC |
| Volusia | West Volusia + New Smyrna Beach |

Reference: [Stellar MLS shareholders](https://www.stellarmls.com/about/shareholders).

### Beaches (`?dataset=beaches` / Spark)

Southeast Florida and Treasure Coast:

| County | Notes |
|--------|--------|
| Broward | Core BeachesMLS |
| Palm Beach | Core |
| Martin | Core |
| St. Lucie | Core |
| Miami-Dade | Included after MIAMI MLS + BeachesMLS merger (May 2026); Spark partition remains `beaches` |

---

## Boundary sources (FDOT)

Administrative boundaries sync from **FDOT Admin Boundaries** (`gis.fdot.gov`):

| Layer | FDOT FeatureServer | PostGIS table | Refresh |
|-------|-------------------|---------------|---------|
| 6 | Counties | `gis_counties` | Annual (`gis.annual_boundaries_refresh`) |
| 7 | Cities (Census places) | `gis_cities` | Annual |
| 8 | ZIP polygons | `gis_zips` | Annual + gap-fill (`gis.zip_sync`) |

Constants: `FDOTCountiesURL`, `FDOTCitiesURL`, `FDOTZipsURL` in `internal/service/gis/arcgis_client.go`.

### City–county pairs (`gis_cities.county`)

FDOT layer 7 (Census places) has **no county attribute** in ArcGIS properties (`ExtractCityRow` in `internal/service/gis/field_extract.go`). County is derived after sync:

1. **`ExpandCityCountyPairs`** (`internal/repository/gis/boundaries.go`) — one row per `(city_name, county_slug)` for each county polygon that intersects the city geometry; **nearest county** fallback when no intersection (Florida Keys / island gaps).
2. Wired from **`sync_boundaries.go`** after each FDOT city import.
3. One-time prod backfill: [`docs/scripts/run_gis_cities_county_expand.sh`](../docs/scripts/run_gis_cities_county_expand.sh) (SQL: [`gis_cities_county_expand.sql`](../docs/scripts/gis_cities_county_expand.sql)).
4. Migration **`00008_gis_cities_county_not_null.sql`** sets `gis_cities.county NOT NULL` after expand validation passes.

Full deploy order and DSN setup: [production-data-backfill.md](production-data-backfill.md).

**Verify before NOT NULL migration:**

```sql
SELECT COUNT(*) FROM gis_cities WHERE county IS NULL;  -- expect 0
```

**Spot-check multi-county cities:**

```sql
SELECT city_name, county FROM gis_cities
WHERE lower(city_name) IN ('jacksonville', 'midway', 'four corners')
ORDER BY city_name, county;
```

---

## Parcel sources — production status

### Implemented in code today

| Source key | County | Query URL | Background sync | Live proxy fallback |
|------------|--------|-----------|-----------------|---------------------|
| `hillsborough_hc_parcels` | Hillsborough | `https://maps.hillsboroughcounty.org/arcgis/rest/services/InfoLayers/HC_ParcelsPublic/FeatureServer/0/query` | **Yes** (default kickoff) | Yes (bbox intersect) |
| `pinellas_enterprise_parcels` | Pinellas | `https://egis.pinellascounty.org/arcgis/rest/services/PARCEL/MapServer/0/query` | Opt-in (`GIS_SYNC_PINELLAS_ENTERPRISE=true`) | Yes (bbox intersect) |
| `florida_statewide_cadastral` | Statewide | `https://services9.arcgis.com/Gh9awoU677aKree0/.../Florida_Statewide_Cadastral/FeatureServer/0/query` | **Removed from kickoff** | Still first in `sourcesForBBox()` — **do not rely on** |

### FDOR statewide 2025 — not viable

Florida Department of Revenue **FDOR Cadastral 2025** on `services9.arcgis.com` is documented as statewide coverage but **does not accept production sync queries** (verified 2026-05-24).

| Query pattern | HTTP | Body | Worker impact (pre-fix) |
|---------------|------|------|-------------------------|
| Bbox only (small envelope) | 200 | `{"count":0}` | Parsed as empty success |
| Bbox + `CO_NO={nn}` | 200 | `{"error":{"code":400,...}}` | Same after ~55s |
| `CO_NO` only | 200 | 400 error object | Timeout or error |

**False-success chain (fixed in working tree, verify merged):**

1. ArcGIS returns HTTP 200 with `error` JSON or zero-count bbox.
2. Old `ArcGISClient.get()` ignored ArcGIS `error` objects.
3. `ParseFeatureCollection` returned 0 features with no error.
4. `SyncPage` finalized sync (`DeleteStaleParcels`) on empty first page.

**Fixes:**

- `arcGISResponseError()` — fail when response body contains `error` (`internal/service/gis/arcgis_client.go`).
- FDOR removed from `parcelSyncTargets()` kickoff.
- Empty first page → hard error in `SyncPage`.
- Recommended: remove FDOR from live `sourcesForBBox()` chain when implementing multi-county sources.

**Verify FDOR still broken:**

```bash
curl -sS --max-time 90 \
  'https://services9.arcgis.com/Gh9awoU677aKree0/ArcGIS/rest/services/Florida_Statewide_Cadastral/FeatureServer/0/query?f=json&geometry=-82.85,27.95,-82.84,27.96&geometryType=esriGeometryEnvelope&inSR=4326&spatialRel=esriSpatialRelIntersects&returnCountOnly=true'
# Expect: {"count":0}
```

---

## Parcel sources — verified county REST endpoints (MLS coverage)

Probed with `scripts/probe-county-parcels.py` (bbox + `returnCountOnly`, May 2026). Use these when extending `sources.go` and `parcelSyncTargets()`.

### Stellar counties

| County | Primary REST `/query` URL | Status | Notes |
|--------|---------------------------|--------|-------|
| Alachua | `https://gis.floridahealth.gov/server/rest/services/EHWATER/Parcels/MapServer/0/query` | OK | Bbox works |
| Charlotte | `https://agis3.charlottecountyfl.gov/arcgis/rest/services/Essentials/CCGISLayers/MapServer/27/query` | OK | |
| DeSoto | SWFWMD parcel_search **layer 3** (see regional table) | OK | No county-hosted REST |
| Flagler | `https://gis.palmcoast.gov/hosting/rest/services/External/FlaglerCountyParcels/MapServer/1/query` | OK | Use `f=json` |
| Hillsborough | See implemented source above | OK | In production sync |
| Lake | `https://gis.lakecountyfl.gov/lakegis/rest/services/OpenData/OpenData1/FeatureServer/12/query` | OK | |
| Manatee | `https://www.mymanatee.org/gisits/rest/services/commonoperational/parcellines/MapServer/0/query` | OK | |
| Marion | `https://gis.marionfl.org/public/rest/services/General/Parcels/MapServer/0/query` | OK | |
| Okeechobee | SFWMD NormalizedParcels + `where=CNTYNAME='Okeechobee'` | OK | |
| Orange | `https://services2.arcgis.com/N4cKzJ9dzXmsPNRs/.../orange_county_parcels/FeatureServer/0/query` | Caveat | Pagination (`where=1=1`) works; bbox tiles often return 0 — tile by county grid |
| Osceola | — | **Gap** | Hosted layer has ~12 features; use PA shapefile bulk ingest |
| Pasco | `https://maps.pascopa.com/arcgis/rest/services/Parcels/MapServer/3/query` | OK | |
| Pinellas | SWFWMD parcel_search **layer 13** | OK | Prefer over enterprise host (timeouts). Alt: `https://egis.pinellas.gov/pcpagis/rest/services/PcpaBaseMap/BaseMapParcelAerials/MapServer/167/query` |
| Polk | `https://gis.polk-county.net/hosting/rest/services/TPO/TPO_Parcel_and_Permit_Map/MapServer/1/query` | OK | |
| Sarasota | `https://services3.arcgis.com/icrWMv7eBkctFu1f/.../ParcelHosted/FeatureServer/0/query` | OK | |
| Sumter | `https://gis.ecfrpc.org/arcgis/rest/services/Basemap/MapServer/4/query` | OK | |
| Volusia | `https://maps5.vcgov.org/arcgis/rest/services/Open_Data/Open_Data_3/FeatureServer/36/query` | OK | Parcel id field: `PID` |

### Beaches counties

| County | Primary REST `/query` URL | Status | Notes |
|--------|---------------------------|--------|-------|
| Broward | `https://services5.arcgis.com/wI5GZmCtnUU8ueya/.../Broward_County_Parcel_Boundary/FeatureServer/1/query` | OK | Slow — use `GIS_HTTP_TIMEOUT=120s` or higher |
| Palm Beach | `https://maps.co.palm-beach.fl.us/arcgis/rest/services/OpenData/open_data_v2/MapServer/0/query` | OK | Parcel id: `PCN` |
| Martin | SFWMD NormalizedParcels + `where=CNTYNAME='Martin'` | OK | No open county polygon REST |
| St. Lucie | SFWMD NormalizedParcels + `where=CNTYNAME='St Lucie'` | OK | Exact string (no period) |
| Miami-Dade | `https://gisweb.miamidade.gov/arcgis/rest/services/MD_LandInformation/MapServer/26/query` | OK | Layer `PaParcel`. SFWMD filter uses `CNTYNAME='Dade'` |

---

## Regional fallback layers

Use when a county primary is slow, missing fields, or unreachable.

### SWFWMD parcel_search (west-central Stellar)

Base: `https://www45.swfwmd.state.fl.us/arcgis12/rest/services/BaseVector/parcel_search/MapServer/{id}/query`

| Layer ID | County |
|----------|--------|
| 1 | Charlotte |
| 3 | DeSoto |
| 7 | Hillsborough |
| 8 | Lake |
| 10 | Manatee |
| 11 | Marion |
| 12 | Pasco |
| 13 | Pinellas |
| 14 | Polk |
| 15 | Sarasota |
| 16 | Sumter |

### SFWMD NormalizedParcels (southeast + Okeechobee)

`https://geoweb.sfwmd.gov/agsext2/rest/services/LandOwnershipAndInterests/NormalizedParcels/FeatureServer/0/query`

Filter with `where=CNTYNAME='…'` (`Broward`, `Martin`, `St Lucie`, `Okeechobee`, **`Dade`** for Miami-Dade).

### TIGERweb (boundaries only — not parcels)

[Census TIGERweb](https://tigerweb.geo.census.gov/tigerwebmain/TIGERweb_main.html) provides counties, places, ZCTAs, legislative districts, and census geographies via ArcGIS REST. **No property parcels.** Useful as a Census-standard boundary fallback; idx-api uses **FDOT** for cities/counties/zips today. ZCTAs ≠ USPS delivery ZIP polygons.

---

## Recommended sync rollout

| Phase | Counties | Rationale |
|-------|----------|-----------|
| 1 (current) | Hillsborough; Pinellas optional | Stellar core; code wired |
| 2 | Manatee, Pasco, Polk, Sarasota, Volusia, Pinellas (SWFWMD L13) | High Stellar listing density |
| 3 | Orange (paginated), Lake, Marion, Charlotte, Flagler, Sumter, DeSoto, Alachua | Central/west fill |
| 4 | Miami-Dade, Palm Beach, Broward, St. Lucie, Martin | Beaches MLS |
| 5 | Osceola | Shapefile/GDB ingest until REST gap closed |

Each new county needs: `source_key`, query URL, `syncBBoxForCounty` envelope, and entry in `parcelSyncTargets()`.

---

## Configuration (parcel sync)

| Variable | Default | Description |
|----------|---------|-------------|
| `GIS_SYNC_PAGE_SIZE` | 2000 | ArcGIS page size; **use 500** for FDOT zips and large counties |
| `GIS_HTTP_TIMEOUT` | 12s | ArcGIS HTTP timeout; **use 120s** for Broward / Pinellas enterprise |
| `GIS_SYNC_PINELLAS_ENTERPRISE` | false | Enqueue Pinellas county enterprise host (often timeout-prone) |
| `GIS_SYNC_QUEUE` | `default` | Queue for `gis.parcel_sync_page` jobs |
| `GIS_SYNC_UPSERT_CHUNK` | 500 | Bulk upsert batch size |

See `.env.example` and `internal/config/config.go`.

---

## Verification

### Re-run county probe script

```bash
python3 scripts/probe-county-parcels.py
```

### PostGIS counts (staging example)

```sql
-- Connection: GOOSE_DBSTRING from .env
SELECT COUNT(*) FROM gis_parcels;
SELECT county, COUNT(*) FROM gis_parcels GROUP BY county ORDER BY county;
SELECT COUNT(*) AS cities, COUNT(county) AS cities_with_county FROM gis_cities;
SELECT COUNT(*) FROM gis_counties;
SELECT COUNT(*) FROM gis_zips;
```

### Worker logs

Successful parcel page:

```text
gis parcel page synced source=hillsborough_hc_parcels county=hillsborough offset=… features=500
```

FDOR / ArcGIS error (after fix):

```text
arcgis error: Unable to perform query. Please check your parameters.
```

### Monitoring API

Dashboard GIS tile uses `parcels_last_synced_at`, `zips_last_synced_at`, per-source generation, and `api_status` from stored probes — see [Admin dashboard](admin-dashboard.md).

### Operator API (admin session)

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/v1/admin/gis/probe` | Probe one source (`source_key`) or all; response includes `{ failed: [{source_key, error}] }` on partial errors |
| `POST` | `/api/v1/admin/gis/sync` | Enqueue `gis.parcel_sync_page` chain (`source_key`, `force`) |
| `GET` | `/api/v1/admin/gis/sources` | List catalog + `gis_source_states` health |
| `POST` | `/api/v1/admin/gis/sources` | Upsert catalog row |
| `PUT` | `/api/v1/admin/gis/sources/:source_key` | Update row |
| `DELETE` | `/api/v1/admin/gis/sources/:source_key` | Soft-disable (`enabled=false`) or `?hard=true` |
| `POST` | `/api/v1/admin/gis/sources/:source_key/upload` | Multipart `.zip`/`.shp` → `gis.shapefile_import` worker job |

**Dashboard:** Monitoring → **Data Quality** → GIS Sources table when logged in as admin:

- **Probe** / **Probe all** — updates `last_probe_at`, `last_probe_ok`, `last_probe_http_status`, `last_probe_error` (shown in table)
- **Sync** — disabled when `sync_mode=shapefile` or source is a boundary-only row
- **Add / Edit / Disable** — CRUD modals wired to `/api/v1/admin/gis/sources`
- **Upload** — per-row file input (`.zip` or `.shp` + sidecars); status from `gis_import_uploads` (`pending` → `processing` → `done` / `failed`)

**Probe all behavior:** Probes the static county catalog **and** all `enabled` rows in `gis_parcel_sources`. Shapefile sources are skipped (no live ArcGIS query). `API: UNKNOWN` in the UI means `last_probe_at` is NULL — run Probe or Probe all after migration `00010`.

**Shapefile ingest:** Set `sync_mode=shapefile` on the catalog row (empty `query_url` allowed), upload via admin API or dashboard. Worker image includes `gdal-tools` (`ogr2ogr`); verify with `make docker-gis-smoke` after image build. Uploads enqueue to **`GIS_IMPORT_QUEUE`** (default `gis-import`); only worker 1 should consume that queue.

**Shared volume (required in production):** Mount the same host path or named volume at `GIS_IMPORT_PATH` (default `/var/cache/geoidx/gis-imports`) on **idx-api-web** and **idx-api-worker 1** in each DC. The API writes uploads; worker 1 runs `gis.shapefile_import` (GeoJSONSeq streaming ingest) and reads via `/vsizip/` or direct `.shp`. Without a shared mount, uploads succeed on API but the worker cannot read the file.

**Env:** `GIS_SYNC_QUEUE`, `GIS_IMPORT_PATH`, `GIS_IMPORT_MAX_BYTES` (default 512MB).

---

## File reference

| Path | Purpose |
|------|---------|
| `internal/service/gis/parcel_sources.go` | 22-county MLS catalog (21 enabled + Osceola stub), sync bboxes, sync modes |
| `internal/service/gis/sources.go` | Live proxy source selection from catalog |
| `internal/service/gis/sync_parcels.go` | Kickoff targets, paginated sync |
| `internal/service/gis/sync_boundaries.go` | FDOT boundary sync + city county backfill |
| `internal/service/gis/arcgis_client.go` | ArcGIS client, FDOT URLs, error detection, sync modes |
| `internal/service/gis/field_extract.go` | Feature → row mapping |
| `scripts/probe-county-parcels.py` | County endpoint smoke probe |
| `migrations/00001_initial.sql` | Consolidated `gis_*` schema (parcels, boundaries, `gis_parcel_sources`) |
