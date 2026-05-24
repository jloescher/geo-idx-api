# GIS Data Proxy API (Florida public government layers)

Quantyra GeoIDX **idx-api** exposes a GIS proxy that returns **public Florida government parcel polygons and administrative boundaries** as **GeoJSON** (with a `meta` foreign member) optimized for **Leaflet**. **No MLS or Bridge RESO data** is read or merged into this endpoint—only external open-government ArcGIS services (synced into PostGIS for fast reads).

## Product and compliance notes

- **Map UX:** Parcel and boundary overlays pair with listing markers so consumers get cadastral context without expanding MLS processing beyond Bridge-backed listing calls.
- **Infra:** PostGIS persistent tables + edge TTL (`GIS_EDGE_CACHE_TTL`) so upstream ArcGIS instability does not scale linearly with traffic.
- **Stellar MLS PDA / IDX:** This layer uses **public** cadastral, county GIS, and FDOT admin boundaries only.

## Authentication

Same as other `/api/v1/*` Bridge proxy routes:

- **Domain mode:** `X-Domain-Slug` header (or `domain` query / Referer host) for an **active** `domains` row.
- **Token mode:** Bearer PAT with `idx:access` or `idx:full`, plus **`X-Domain-Slug`** / **`?domain=`** for a verified domain on the token owner's account.

**Access shape**

- **Domain** identification and PATs with **`idx:full`**: full GeoJSON (`meta.teaser=false`, `meta.full_access=true`).
- PATs with **`idx:access` only** (no `idx:full`): teaser tier — feature cap `GIS_TEASER_MAX_FEATURES` (default 40), coordinate precision `GIS_TEASER_COORD_DECIMALS` (default 4). Full payloads are cached in `gis_cache`; teaser limits apply on the response path.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/gis` | Florida GIS features for the authenticated domain/token. |
| GET | `/api/v1/mls/{mlsCode}/gis` | Same payload, MLS-scoped for analytics / future routing (`mlsCode` must be listed in `GIS_FLORIDA_MLS_CODES`). |

## Query parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `type` | no | Feature layer: `parcel` (default), `city`, `county`, or `zip`. |
| `bbox` | one-of | Comma-separated **west,south,east,north** (WGS84), e.g. Clearwater pilot: `-82.83,27.95,-82.79,27.98`. |
| `north`, `south`, `east`, `west` | one-of | Alternative bounding box. |
| `lat` / `latitude` + `lng` / `lon` / `longitude` + `radius` | one-of | Radius in **meters** (50–500000). Builds a square envelope. |
| `limit` | no | Max features returned (capped by `GIS_MAX_FEATURES`, default 500). |

**BBox span guard:** `GIS_MAX_BBOX_SPAN_DEG` (default `0.35`) rejects abusive world queries with `422`.

## Example requests

**Parcels (default)**

```http
GET /api/v1/gis?bbox=-82.83,27.95,-82.79,27.98&limit=120
X-Domain-Slug: your-registered-domain.com
```

**City boundaries**

```http
GET /api/v1/gis?type=city&bbox=-82.83,27.95,-82.79,27.98
X-Domain-Slug: your-registered-domain.com
```

**County boundaries**

```http
GET /api/v1/gis?type=county&bbox=-82.5,27.5,-82.0,28.0
X-Domain-Slug: your-registered-domain.com
```

**ZIP code boundaries**

```http
GET /api/v1/gis?type=zip&bbox=-82.83,27.95,-82.79,27.98
X-Domain-Slug: your-registered-domain.com
```

## Example response (truncated)

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "geometry": { "type": "MultiPolygon", "coordinates": [] },
      "properties": { "PARCELID": "…" }
    }
  ],
  "meta": {
    "source_used": "gis_parcels",
    "source_tier": "persistent",
    "county_hint": "pinellas",
    "query_type": "parcel",
    "teaser": false,
    "full_access": true,
    "cache_generation": 3,
    "cached": false
  }
}
```

When **degraded** (`meta.degraded=true`), `features` may be empty and `meta.leaflet_fallback` contains a public OSM raster tile template for Leaflet `L.tileLayer`.

## Persistent PostGIS tables

| Table | Refresh | Source |
|-------|---------|--------|
| `gis_parcels` | Monthly (`gis.monthly_parcel_refresh`, 1st @ 02:00) | Florida statewide cadastral + Pinellas + Hillsborough county layers |
| `gis_cities` | Annual (`gis.annual_boundaries_refresh`, Jan 1 @ 03:00) | FDOT Admin_Boundaries layer 7 |
| `gis_counties` | Annual | FDOT Admin_Boundaries layer 6 |
| `gis_zips` | Annual | FDOT Admin_Boundaries layer 8 |

All tables use **GIST** spatial indexes and `ST_Intersects` + envelope prefilter for bbox queries.

**Initial sync:** On scheduler startup, if all persistent GIS tables are empty, `gis.initial_sync` is enqueued once (parcels + boundaries).

## Read path (layered)

| Layer | Policy | Notes |
|-------|--------|-------|
| **Edge (`gis_cache`)** | `GIS_EDGE_CACHE_TTL` (seconds, default 900) | Keyed by full query string (includes `type`). Header `X-IDX-Cache`: `edge`, `persistent`, or `miss`. |
| **PostGIS persistent** | Monthly / annual sync jobs | Primary origin for `type=parcel|city|county|zip`. `meta.source_tier=persistent`. |
| **ArcGIS live fallback** | Parcels only | When persistent parcel table has no rows for the bbox, existing county failover runs (statewide → Pinellas → Hillsborough). |

## Background jobs

| Job type | Schedule | Queue | Purpose |
|----------|----------|-------|---------|
| `gis.probe_sources` | Weekly Mon 06:30 | `GIS_QUEUE` | ArcGIS metadata fingerprint; bumps `gis_source_states.generation` on change. |
| `gis.monthly_parcel_refresh` | 1st of month 02:00 | `GIS_SYNC_QUEUE` | Full parcel sync with paginated sub-jobs (`gis.parcel_sync_page`). |
| `gis.annual_boundaries_refresh` | Jan 1 03:00 | `GIS_SYNC_QUEUE` | FDOT cities/counties/zips sync. |
| `gis.initial_sync` | Scheduler startup (if empty) | `GIS_SYNC_QUEUE` | First-run parcel + boundary load. |

**Worker queues:** Include `GIS_SYNC_QUEUE` (default `default`) in `WORKER_QUEUES` alongside `GIS_QUEUE`.

## Failover behavior (parcel ArcGIS fallback)

When persistent `gis_parcels` has no data for the requested bbox:

1. **Primary:** Florida statewide cadastral with optional `CO_NO=` county filter.
2. **Failover — Pinellas:** When bbox intersects Pinellas envelope.
3. **Failover — Hillsborough:** When bbox intersects Hillsborough envelope.
4. **Graceful degrade:** Empty `FeatureCollection` + OSM tile fallback metadata.

Boundary types (`city`, `county`, `zip`) do not live-fallback to ArcGIS; run `gis.annual_boundaries_refresh` if empty.

## Chaining with `/listings`

Geo-web can call `/api/v1/listings` then `/api/v1/gis` with the **same map bbox** so markers and parcels stay aligned.

## Configuration reference

| Variable | Default | Description |
|----------|---------|-------------|
| `GIS_MAX_FEATURES` | 500 | API `limit` cap |
| `GIS_SYNC_PAGE_SIZE` | 2000 | ArcGIS pagination page size |
| `GIS_SYNC_UPSERT_CHUNK` | 500 | Bulk upsert batch size |
| `GIS_HTTP_TIMEOUT` | 12s | ArcGIS HTTP timeout |
| `GIS_SYNC_QUEUE` | `default` | Queue for sync jobs |
| `GIS_QUEUE` | `default` | Queue for `gis.probe_sources` |
| `GIS_EDGE_CACHE_TTL` | 900 | Edge cache TTL (seconds) |
| `GIS_TEASER_MAX_FEATURES` | 40 | Teaser feature cap |
| `GIS_TEASER_COORD_DECIMALS` | 4 | Teaser coordinate rounding |

See `internal/service/gis/sources.go` and `GIS_*` env vars in `.env.example` for source URLs and Florida MLS allow-list.
