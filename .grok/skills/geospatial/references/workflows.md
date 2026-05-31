# Geospatial Workflows Reference

## Contents
- Adding a New Spatial Query
- Populating Coordinates During Replication
- Adding a New GIS Source
- Testing Spatial Queries
- Troubleshooting

## Adding a New Spatial Query

Copy this checklist when building a new PostGIS filter:

```
- [ ] 1. Start WHERE clause with AND coordinates IS NOT NULL
- [ ] 2. Use ST_DWithin(coordinates::geography, ST_SetSRID(ST_MakePoint($N, $N+1), 4326)::geography, $N+2)
- [ ] 3. Parameter order: lng (X), lat (Y), meters
- [ ] 4. Convert user-facing miles: meters := miles * 1609.34
- [ ] 5. Increment parameter counter by 3 (n += 3)
- [ ] 6. Test with EXPLAIN ANALYZE — verify GiST index hit
```

### Verified Template

```go
// new code to add
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

### Validation

```bash
# After adding, verify index usage:
psql "$GOOSE_DBSTRING" -c "
EXPLAIN ANALYZE SELECT listing_key FROM listings
WHERE dataset_slug = 'stellar'
  AND coordinates IS NOT NULL
  AND ST_DWithin(
        coordinates::geography,
        ST_SetSRID(ST_MakePoint(-82.8, 27.95), 4326)::geography,
        16093
      )
LIMIT 50;
"
# Look for "Index Scan using listings_ap_geom_gix" in the output.
```

1. Make changes
2. Validate: `psql` EXPLAIN ANALYZE as above
3. If plan shows `Seq Scan`, verify the partial index WHERE clause matches your query conditions
4. Only proceed when `Index Scan using listings_ap_geom_gix` appears

## Populating Coordinates During Replication

Coordinates are written at persist time, not during the initial listing upsert. The replication pipeline:

1. **Fetch** (`bridge.fetch_page` / `spark.fetch_page`) — stores raw MLS JSON in `replica_pages`
2. **Persist** (`bridge.persist_chunk` / `spark.persist_chunk`) — parses JSON, upserts `listings` row
3. **Coordinate flush** — separate pass extracts `Latitude`/`Longitude`, batches into `UPDATE FROM VALUES`

### How flushCoordinates Works

```go
// internal/service/sync/listing_mirror.go:263-292
// 1. Collect (datasetSlug, listingKey, lng, lat) pairs during persist
// 2. Flush in batches of 250 using UPDATE FROM VALUES
// 3. Null coordinates flushed separately with SET coordinates = NULL
```

### Adding a New Coordinate Source

If adding a new dataset that needs spatial queries:

1. Ensure upstream data has `Latitude` and `Longitude` fields
2. Verify `BuildListingRecord` extracts them into `coordPair` structs
3. The existing `flushCoordinates`/`flushNullCoordinates` handles them automatically
4. Verify after replication:

```sql
SELECT COUNT(*) AS total,
       COUNT(coordinates) AS with_geom
FROM listings WHERE dataset_slug = 'your_dataset';
```

5. If `with_geom` is 0, check that upstream field names match the extraction logic in `listing_row.go`.

## Adding a New GIS Source

Sources are defined in `internal/service/gis/sources.go`. To add a new Florida county ArcGIS layer:

```
- [ ] 1. Add source definition with URL, county bounding box, and CO_NO filter if applicable
- [ ] 2. Add source to fallback hierarchy in internal/service/gis/proxy.go
- [ ] 3. Set GIS_MAX_FEATURES and source-specific max age env vars
- [ ] 4. Test: GET /api/v1/gis?bbox=... with bbox intersecting the new county
- [ ] 5. Verify cache: check gis_cache rows with source_used matching new source key
- [ ] 6. Verify probe: gis.probe_sources job fingerprints new source metadata
```

See the **cache-postgres** skill for PostgreSQL caching patterns.

## Testing Spatial Queries

### Verify Coordinate Population

```sql
SELECT COUNT(*) AS total,
       COUNT(list_price) AS with_price,
       COUNT(coordinates) AS with_geom
FROM listings WHERE dataset_slug = 'stellar';
```

`with_geom` should approach `total` for listings with upstream coordinates.

### Verify Index Usage

```sql
EXPLAIN ANALYZE SELECT listing_key FROM listings
WHERE coordinates IS NOT NULL
  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
  AND ST_DWithin(
        coordinates::geography,
        ST_SetSRID(ST_MakePoint(-82.8, 27.95), 4326)::geography,
        16093
      );
```

Expect: `Index Scan using listings_ap_geom_gix` — if you see `Seq Scan`, the partial index condition is not met or coordinates are null.

### Verify BBox Span Guard

```bash
# Should return 422 (span > 0.35 deg)
curl -s -o /dev/null -w '%{http_code}' \
  'http://localhost:8000/api/v1/gis?bbox=-90,20,-80,30'
```

## Troubleshooting

### Spatial Queries Return No Results

1. Check `coordinates IS NOT NULL` — `SELECT COUNT(coordinates) FROM listings`
2. Check `standard_status` — partial index only covers `active`/`pending`
3. Check SRID: `SELECT ST_SRID(coordinates) FROM listings LIMIT 1` — must be 4326
4. Check parameter order: `ST_MakePoint(lng, lat)`, not `(lat, lng)`

### Replication Not Populating Coordinates

1. Verify upstream `Latitude`/`Longitude` fields in raw data:
   ```sql
   SELECT raw_data->>'Latitude', raw_data->>'Longitude'
   FROM listings WHERE dataset_slug = 'stellar' LIMIT 5;
   ```
2. Check `flushCoordinates` is called during persist — look for `coordinates = v.geom` in worker logs
3. If fields exist but coordinates are null, check field name mapping in `listing_row.go`

### GIS Cache Stale After Source Change

The `gis.probe_sources` job (Monday 06:30) detects source changes. To force invalidation:

```sql
-- Bump generation to invalidate all cache for that source
UPDATE gis_source_states SET generation = generation + 1 WHERE source_key = 'your_source';
```

Or enqueue the probe job manually via the scheduler.

See the **postgresql** skill for general query debugging and the **queue-postgresql** skill for job troubleshooting.