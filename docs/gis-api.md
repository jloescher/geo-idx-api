# GIS Data Proxy API (Florida public government layers)

Quantyra GeoIDX **idx-api** exposes a GIS proxy that returns **public Florida government parcel polygons** as **GeoJSON** (with a `meta` foreign member) optimized for **Leaflet**. **No MLS or Bridge RESO data** is read or merged into this endpoint—only external open-government ArcGIS services.

## Revenue & compliance notes

- **Lead capture / time-on-site:** Parcel overlays increase map dwell time before and after OTP, improving registration completion rates without expanding MLS data processing.
- **Infra margin:** 15-minute PostgreSQL + Laravel cache mirrors `listings_cache` economics so government ArcGIS instability does not scale linearly with traffic.
- **Stellar MLS PDA / IDX:** This layer uses **public** cadastral and county GIS only, consistent with enhancing IDX display with non-MLS context.

## Authentication

Same as other `/api/v1/*` Bridge proxy routes:

- **Domain mode:** `X-Domain-Slug` header (or `domain` query / Referer host) for an **active** `domains` row.
- **Token mode:** Bearer Sanctum token with `idx:access` or `idx:full`.

**Teaser vs full access**

- Domain / `idx:access` → **teaser:** simplified coordinates, limited properties, `meta.teaser=true`.
- `idx:full` → **full:** richer attributes, `meta.context_layers` hints (URLs only; client-side fetch).

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/gis` | Florida GIS parcels for the authenticated domain/token. |
| GET | `/api/v1/mls/{mlsCode}/gis` | Same payload, MLS-scoped for analytics / future routing (`mlsCode` must be listed in `GIS_FLORIDA_MLS_CODES`). |

## Query parameters

Aligned with typical geo-web listing map queries:

| Parameter | Required | Description |
|-----------|----------|-------------|
| `bbox` | one-of | Comma-separated **west,south,east,north** (WGS84), e.g. Clearwater pilot: `-82.83,27.95,-82.79,27.98`. |
| `north`, `south`, `east`, `west` | one-of | Alternative bounding box. |
| `lat` / `latitude` + `lng` / `lon` / `longitude` + `radius` | one-of | Radius in **meters** (50–500000). Builds a square envelope. |
| `limit` | no | Max ArcGIS features (capped by `GIS_MAX_FEATURES`, default 500). |
| `layers` | no | Reserved comma list (default `parcels`). Full access may expose future multi-layer orchestration. |

**BBox span guard:** `GIS_MAX_BBOX_SPAN_DEG` (default `0.35`) rejects abusive world queries with `422`.

## Example requests

**Clearwater / Pinellas (bbox)**

```http
GET /api/v1/gis?bbox=-82.83,27.95,-82.79,27.98&limit=120
X-Domain-Slug: your-registered-domain.com
```

**Tampa / Hillsborough overlap (lat/lng + radius)**

```http
GET /api/v1/gis?lat=27.9506&lng=-82.4572&radius=800&limit=80
X-Domain-Slug: your-registered-domain.com
```

**MLS-scoped (Stellar)**

```http
GET /api/v1/mls/stellar/gis?bbox=-82.48,27.92,-82.44,27.96&limit=50
X-Domain-Slug: your-registered-domain.com
```

## Example response (truncated)

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "geometry": { "type": "Polygon", "coordinates": [] },
      "properties": { "PARCELID": "…" }
    }
  ],
  "meta": {
    "source_used": "pinellas_enterprise_parcels",
    "source_tier": "pinellas",
    "county_hint": "pinellas",
    "teaser": true,
    "full_access": false,
    "mls_code": null,
    "layers": ["parcels"],
    "cached": false,
    "cache_hit": null,
    "cache_generation": 0,
    "blob_valid_until": "2026-05-24T00:00:00+00:00",
    "warnings": ["Served from Pinellas County Enterprise GIS parcels."],
    "degraded": false,
    "bbox": { "min_lon": -82.83, "min_lat": 27.95, "max_lon": -82.79, "max_lat": 27.98 },
    "expires_at": "2026-05-24T00:00:00+00:00",
    "context_layers": [],
    "leaflet_fallback": null
  }
}
```

When **degraded** (`meta.degraded=true`), `features` is empty and `meta.leaflet_fallback` contains a public OSM raster tile template for Leaflet `L.tileLayer`.

## Failover behavior (server-side)

1. **Primary:** Florida Department of Revenue / FGIO statewide cadastral (`FeatureServer/0`). When `county_hint` is `pinellas` or `hillsborough`, a `CO_NO=` filter is applied to shrink the query.
2. **Failover 1 — Pinellas:** Only if the bbox intersects the Pinellas envelope in `config/gis.php`.
3. **Failover 2 — Hillsborough:** Only if the bbox intersects the Hillsborough envelope.
4. **Graceful degrade:** Empty `FeatureCollection` + `warnings` + OSM tile fallback metadata.

## Caching & durability (layered)

| Layer | Policy | Notes |
|-------|--------|-------|
| **Laravel `Cache` (edge)** | `GIS_EDGE_CACHE_TTL` (seconds, default 900; falls back to legacy `GIS_CACHE_TTL`) | Full JSON payload keyed by `query_hash`. First read path for hot repeat requests. |
| **PostgreSQL `gis_cache` (origin)** | Per-source **max age in days** (`GIS_ORIGIN_MAX_DAYS_PRIMARY` default 90 for statewide, `GIS_ORIGIN_MAX_DAYS_COUNTY` default 30 for county layers, degraded 1 day) | `meta.cache_generation` + column `source_generation` must match `gis_source_states.generation` for that `source_used`. |
| **`gis_source_states`** | Weekly scheduled probe | `php artisan gis:probe-sources` (or `--queued`) fetches each layer `?f=json`, fingerprints `currentVersion` + `editingInfo` + `serviceItemId`; fingerprint change **increments `generation`**, invalidating edge + origin rows for that source. |
| **Filesystem `gis_backup`** | Snapshot per `query_hash` | `GIS_BACKUP_PATH`; optional `GIS_QUEUE_BACKUP_WRITES` on `GIS_QUEUE`. |

**Operations**

- `php artisan gis:probe-sources` — run metadata fingerprints now (sync).
- `php artisan gis:probe-sources --queued` — dispatch `RefreshGisSourceMetadataJob` on the GIS queue.
- `php artisan gis:clear-cache --source=pinellas_enterprise_parcels` — delete origin rows for one source and bump its generation.
- `php artisan gis:clear-cache --all` — truncate `gis_cache` and bump **all** source generations.

**Scheduler:** `routes/console.php` dispatches `RefreshGisSourceMetadataJob` **weekly** (Monday 06:30 app timezone). Requires `schedule:run` / `schedule:work` and a **queue worker** when using `--queued` probes or queued backup writes.

**HTTP client:** `GIS_HTTP_TIMEOUT` / `GIS_HTTP_CONNECT_TIMEOUT` for parcel queries; `GIS_METADATA_TIMEOUT` for cheap layer metadata probes.

## Chaining with `/listings`

Geo-web can call `/api/v1/listings` then `/api/v1/gis` with the **same map bbox** (or derived from `lat`/`lng`/`radius`) so markers and parcels stay aligned—no MLS data crosses into GIS responses.

## Configuration reference

See `config/gis.php` for source URLs, county bounding boxes, teaser limits, and Florida MLS allow-list.
