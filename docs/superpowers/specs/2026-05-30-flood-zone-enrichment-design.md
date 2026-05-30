# Flood zone accuracy and API enrichment

Design spec for coordinate normalization at MLS sync, FEMA outcome taxonomy, MLS fallback for `low_risk_flood_zone_yn`, and nested `flood_zone` API object.

See implementation plan and `docs/fema-flood-enrichment.md` for operational details.

## Goals

1. Fix swapped FL lat/lng at MLS sync ingest (not only FEMA recovery).
2. Set `low_risk_flood_zone_yn` from MLS `flood_zone_code` when FEMA cannot return a zone.
3. Classify FEMA outcomes with clearer reasons (`out_of_coverage`, etc.).
4. Expose nested `flood_zone` object with `status` and `reason` on listing and search API responses.

## API shape

```json
"flood_zone": {
  "mls_code": "X",
  "fema_code": null,
  "effective_code": "X",
  "sfha": null,
  "low_risk": true,
  "source": "mls",
  "status": "mls_fallback",
  "reason": "nfhl_no_polygon_at_point",
  "updated_at": "2026-05-25T12:00:00Z"
}
```

Top-level `FloodZoneCode` (MLS) remains for backward compatibility.

## Status mapping

| DB `fema_failure_reason` | Client `status` | Client `reason` |
|--------------------------|-----------------|-----------------|
| NULL + FEMA code | `enriched` | `nfhl_success` |
| `no_nfhl_feature` | `mls_fallback` or `no_data` | `nfhl_no_polygon_at_point` |
| `out_of_coverage` | `out_of_coverage` | `outside_nfhl_coverage` |
| `insufficient_coords` | `coords_recovery` | `suspicious_coordinates` |
| `request_error` | `error` | `upstream_error` |
| NULL, no watermark | `pending` | `pending_enrichment` |
