# Geospatial Patterns Reference

## Contents
- Spatial Query Construction
- Coordinate Storage and Indexing
- Bounding Box Handling
- GIS Parcel Proxy
- Teaser Tier
- Anti-Patterns

## Spatial Query Construction

All spatial queries follow one pattern: `ST_DWithin` on `geography(Point, 4326)` with parameterized coordinates.

### Radius Filter (Search, Comps)

```go
// internal/service/search/postgis.go:141-150
// internal/service/comps/mirror.go:47-55
if req.Lat != nil && req.Lng != nil && req.RadiusMiles != nil {
    meters := *req.RadiusMiles * 1609.34
    q += fmt.Sprintf(` AND coordinates IS NOT NULL AND ST_DWithin(
            coordinates::geography,
            ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography,
            $%d
        )`, n, n+1, n+2)
    args = append(args, *req.Lng, *req.Lat, meters)
    n += 3
}
```

Key rules:
- Always `AND coordinates IS NOT NULL` before `ST_DWithin` — nulls skip the spatial check
- `ST_MakePoint(lng, lat)` — longitude is X, latitude is Y
- `::geography` cast on both sides for spherical distance in meters
- Miles to meters: `* 1609.34`
- Parameterized (`$N`), never interpolated

### Batch Coordinate Population

```go
// internal/service/sync/listing_mirror.go:278-282
// During replication persist, coordinates are flushed in batches of 250:
parts = append(parts, fmt.Sprintf(
    "($%d::varchar, $%d::varchar, ST_SetSRID(ST_MakePoint($%d::float8, $%d::float8), 4326)::geography)",
    n, n+1, n+2, n+3))
args = append(args, p.datasetSlug, p.listingKey, p.lng, p.lat)
```

Then bulk UPDATE:

```sql
UPDATE listings AS l SET coordinates = v.geom, updated_at = $1
FROM (VALUES ...) AS v(ds, k, geom)
WHERE l.dataset_slug = v.ds AND l.listing_key = v.k
```

Null coordinates are flushed separately with `SET coordinates = NULL`.

## Coordinate Storage and Indexing

### Schema

```sql
-- migrations/00001_initial.sql:262
coordinates geography(Point, 4326) NULL,
latitude DOUBLE PRECISION NULL,
longitude DOUBLE PRECISION NULL,
```

- `coordinates` is the authoritative spatial column (PostGIS `geography`)
- `latitude`/`longitude` are scalar copies for non-spatial reads
- Both populated at persist time from upstream `Latitude`/`Longitude` fields

### Partial GiST Index

```sql
-- migrations/00001_initial.sql:272-273
CREATE INDEX listings_ap_geom_gix ON listings USING GIST (coordinates)
    WHERE coordinates IS NOT NULL
      AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending');
```

Only active/pending rows with coordinates are indexed. This keeps the index proportional to the searchable set, not the full mirror history.

### WARNING: Do Not Use Geometry Type for Distance

**The Problem:**

```sql
-- BAD — uses planar math, wrong results at Florida latitudes
ST_DWithin(coordinates, ST_MakePoint(lng, lat), 16093)
```

**Why This Breaks:** `geometry` type treats coordinates as Cartesian. At ~28N (Florida), one degree of longitude is ~98 km, not ~111 km. A 10-mile radius query returns incorrect results.

**The Fix:**

```sql
-- GOOD — geography type uses spherical math
ST_DWithin(
    coordinates::geography,
    ST_SetSRID(ST_MakePoint(lng, lat), 4326)::geography,
    16093  -- meters
)
```

### WARNING: Never Interpolate Coordinates into SQL

**The Problem:**

```go
// BAD — SQL injection and precision loss
q += fmt.Sprintf(" AND ST_DWithin(coordinates, ST_MakePoint(%f, %f), %f)", lng, lat, meters)
```

**Why This Breaks:** Float formatting can truncate precision. More critically, any user-supplied coordinate is an injection vector if not parameterized.

**The Fix:** Always use `$N` placeholders (verified pattern above).

## Bounding Box Handling

`internal/service/gis/query.go` parses three input formats:

```go
// 1. bbox=west,south,east,north
BBox{West: vals[0], South: vals[1], East: vals[2], North: vals[3]}

// 2. north/south/east/west params
BBox{West: west, South: south, East: east, North: north}

// 3. lat/lng + radius (meters) -> square envelope
deg := radius / 111320.0
BBox{West: lng - deg, East: lng + deg, South: lat - deg, North: lat + deg}
```

### Span Guard

```go
// internal/service/gis/query.go:60-73
func (b BBox) SpanDeg() float64 { /* max of width, height */ }
```

`GIS_MAX_BBOX_SPAN_DEG` (default `0.35`) rejects requests spanning more than ~0.35 degrees. At Florida latitudes this is roughly 35-39 km — enough for a neighborhood view, too small for "show me the whole state."

### ArcGIS Envelope Format

```go
func (b BBox) EsriEnvelope() string {
    return fmt.Sprintf("%f,%f,%f,%f", b.West, b.South, b.East, b.North)
}
```

Used as the `geometry` parameter for ArcGIS FeatureServer queries.

## GIS Parcel Proxy

### Layered Caching

| Layer | Table/Mechanism | Invalidation |
|-------|----------------|--------------|
| Edge | `gis_cache` (TTL) | `GIS_EDGE_CACHE_TTL` seconds (default 900) |
| Origin | `gis_cache` (source-generation) | `gis_source_states.generation` bump |
| Filesystem | `GIS_BACKUP_PATH` | Optional snapshot per `query_hash` |

Source metadata is fingerprinted weekly by `gis.probe_sources` (scheduler Monday 06:30). Fingerprint change triggers `generation` increment which invalidates stale cache rows.

### ArcGIS Source Fallback

1. Primary: statewide cadastral with optional `CO_NO=` county filter
2. Failover: county-specific ArcGIS (Pinellas, Hillsborough)
3. Graceful degrade: empty `FeatureCollection` + OSM tile fallback in `meta.leaflet_fallback`

### Teaser Tier

`internal/service/gis/teaser.go` applies to `idx:access`-only tokens:

- Feature cap: `GIS_TEASER_MAX_FEATURES` (default 40)
- Coordinate rounding: `GIS_TEASER_COORD_DECIMALS` (default 4 — ~11m precision)
- Full payloads cached; teaser applied on response path only

## Anti-Patterns

### WARNING: ST_Distance in WHERE Clause

**The Problem:**

```sql
-- BAD — computes distance for every row, no index use
WHERE ST_Distance(coordinates::geography, point::geography) < 16093
```

**Why This Breaks:** `ST_Distance` is not indexable. PostgreSQL evaluates it for every row with non-null coordinates. On 100k listings this is a full sequential scan.

**The Fix:** `ST_DWithin` uses the GiST index for a bounding-box pre-filter, then refines.

### WARNING: Omitting coordinates IS NOT NULL Guard

**The Problem:**

```sql
-- BAD — NULL coordinates cause ST_DWithin to return NULL (not false)
WHERE ST_DWithin(coordinates::geography, ...)
```

**Why This Breaks:** `ST_DWithin(NULL, ...)` returns NULL, which is falsy but confuses the planner and can cause unexpected full scans.

**The Fix:** Always guard with `AND coordinates IS NOT NULL` before spatial predicates (verified pattern above).