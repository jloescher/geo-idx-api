---
name: geospatial
description: |
  Manages PostGIS spatial queries and geographic data processing for MLS listing search,
  GIS parcel proxy, and comps engine. Covers coordinate storage, spatial indexing,
  bounding box handling, distance filtering, and ArcGIS parcel proxy with teaser tiers.
  Use when: working with PostGIS functions, coordinates column, BBox parsing, GIS proxy,
  spatial search filters, ST_DWithin/ST_MakePoint queries, or geographic data pipelines.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash, mcp__4_5v_mcp__analyze_image, mcp__web_reader__webReader
---

# Geospatial Skill

PostGIS spatial queries for listing search (`ST_DWithin` radius), GIS parcel proxy (ArcGIS with layered cache), and comps engine — all using SRID 4326 `geography(Point)` with a partial GiST index on active/pending rows.

## Before You Code (REQUIRED)

This skill's content was captured at generation time and MAY be stale. For ANY non-trivial change involving geospatial, verify against current docs FIRST:



Then:

1. **Match the installed version.** Cross-reference against the version installed in this repo. APIs change across minor versions; do not assume.
2. **Discover provider best practices.** If the task touches a production-sensitive capability, inspect the provider service catalog, official docs, and project docs before choosing an implementation.
3. **Respect explicit direction.** If the user explicitly asks for a specific mechanism, follow it. If project docs clearly mandate a mechanism, follow the project. In both cases, mention the provider-recommended alternative and make the chosen path safe.
4. **Prefer provider-native primitives by default.** If no explicit user/project override exists and the change involves caching, rate limiting, background work, scheduled jobs, shared state, queues, or secrets, use the provider-recommended binding/API. Do not hand-roll an in-memory or polyfill solution that "works" locally but breaks under the provider's execution model — derive the need→native-primitive mapping yourself from this provider's docs.

## Capability Contract

Use this section when the user prompt touches production risk, even if the prompt does not name this technology explicitly.




Required wiring surfaces:
- runtime/infrastructure config: Dockerfile
- nearest typed request/context boundary
- handler/procedure boundary before external side effects

Side-effect barrier:
- Place guards before external APIs, auth mutations, email sends, analytics events, storage writes, and database mutations.


Fallback policy:
- Prefer provider-native/platform-managed primitives by default when no explicit override exists.
- Follow clear user/project overrides, but mention the native alternative and tradeoff.
- Fallbacks must be durable, multi-instance safe, and atomic under concurrency.

Verification rules:
- [error] native-or-explicit-override: Use the provider-native primitive first unless the user/project explicitly overrides it.
- [error] atomic-fallback: Fallback counters must be atomic under concurrency.

## Quick Start

### Verified Existing Pattern — Radius Search

```go
// internal/service/search/postgis.go:141-150
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

### Verified Existing Pattern — Batch Coordinate Flush

```go
// internal/service/sync/listing_mirror.go:263-292
// Batch of 250: UPDATE listings SET coordinates = v.geom FROM (VALUES ...) AS v(...)
parts = append(parts, fmt.Sprintf(
    "($%d::varchar, $%d::varchar, ST_SetSRID(ST_MakePoint($%d::float8, $%d::float8), 4326)::geography)",
    n, n+1, n+2, n+3))
```

### New Code Pattern — Adding a Spatial Filter

```go
// new code to add — follow the parameterized pattern, never interpolate coordinates
meters := radiusMiles * 1609.34
q += fmt.Sprintf(` AND coordinates IS NOT NULL AND ST_DWithin(
        coordinates::geography,
        ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography,
        $%d
    )`, n, n+1, n+2)
args = append(args, lng, lat, meters) // lng first (X), lat second (Y) for ST_MakePoint
n += 3
```

## Key Concepts

| Concept | Usage | Reference |
|---------|-------|-----------|
| SRID 4326 | All coordinates — WGS84 lat/lng | `ST_SetSRID(..., 4326)` everywhere |
| `geography` type | Accurate spherical distance in meters | `::geography` cast on Point and search point |
| Miles to meters | `miles * 1609.34` | Search, comps radius filters |
| Partial GiST index | Active/pending only — keeps index small | `migrations/00001_initial.sql:272` |
| BBox span guard | Max `0.35` deg default — rejects world queries | `GIS_MAX_BBOX_SPAN_DEG` |
| Coordinate order | `ST_MakePoint(lng, lat)` — X then Y | All spatial queries |
| Batch size 250 | `flushCoordinates` chunk limit | `internal/service/sync/listing_mirror.go:267` |

## Common Patterns

### GIS Parcel Proxy with Caching

Three cache layers: edge (TTL-based `gis_cache`), origin (source-generation-aware), filesystem backup. ArcGIS source metadata fingerprinted weekly; generation change invalidates stale rows.

### Teaser Tier for Non-Full Access

`idx:access`-only tokens get capped features (default 40) and rounded coordinates (default 4 decimal places). Full access passes through unchanged. See `internal/service/gis/teaser.go`.

## See Also

- [patterns](references/patterns.md)
- [workflows](references/workflows.md)

## Related Skills

- See the **go** skill for Go-specific patterns and error handling
- See the **postgresql** skill for general PostgreSQL query and migration patterns
- See the **fiber** skill for HTTP handler and middleware patterns
- See the **cache-postgres** skill for PostgreSQL-backed caching strategies
- See the **queue-postgresql** skill for background job processing