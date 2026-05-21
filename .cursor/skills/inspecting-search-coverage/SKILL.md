---
name: inspecting-search-coverage
description: Audits technical and on-page search coverage across the idx-api Go codebase, including Bridge MLS listing filters, GIS parcel queries, and hybrid PostGIS search.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Inspecting Search Coverage (Go)

Audits search and filter behavior in the **Go** idx-api service: live MLS proxy, PostGIS mirror search, and GIS parcel queries.

## Quick start

```bash
# Bridge / RESO proxy and cache
grep -rn "finishProxy\|FingerprintRequest\|filters" internal/handler/bridge/ internal/service/cache/

# Hybrid search routing (mirror vs live)
grep -rn "Route\|PostGIS\|live" internal/service/search/

# GIS bbox / failover
grep -rn "ParseBBox\|sourcesForBBox" internal/service/gis/

# Comps sold vs mirror legs
grep -rn "fetchSoldComps\|findMirrorComps" internal/service/comps/
```

## Key files

| Surface | Location |
|---------|----------|
| Live listings / RESO proxy | `internal/handler/bridge/handler.go` |
| On-demand proxy cache | `internal/service/cache/proxy_cache.go`, `canonical.go` |
| Hybrid search | `internal/service/search/service.go`, `route.go`, `postgis.go`, `live_search.go` |
| GIS parcels | `internal/service/gis/proxy.go`, `query.go`, `sources.go` |
| Comps | `internal/service/comps/engine.go`, `upstream.go`, `mirror.go` |
| Auth | `internal/api/middleware/domain_token.go` |

## Bridge filter forwarding

- Query parameters (including `filters`) are forwarded to Bridge/Spark upstream via `mlspoxy` clients.
- **Caching:** `FingerprintRequest` hashes method, upstream URL, and sorted query (excluding `domain`). Identical requests return `X-IDX-Cache: HIT` from `mls_search_cache`. Different `filters` values produce different fingerprints (separate cache entries).
- **No teaser truncation** in this deployment: authenticated `domain.token` traffic receives full JSON (see `docs/idx-api-bridge-proxy.md`).

## GIS spatial queries

- `GET /api/v1/gis` — `ParseBBox` from `west,south,east,north` or `lat`/`lng` + `radius` (meters).
- `GIS_MAX_BBOX_SPAN_DEG` rejects oversized envelopes.
- Source failover: statewide → county layers → degraded empty FC + OSM tile hint (`internal/service/gis/proxy.go`).

## Hybrid search routing

- `POST /api/v1/search` — Active/Pending from PostGIS `listings` when possible; Closed (and mixed status) from live RESO (`internal/service/search/route.go`).
- Mirror filters: `low_risk_floodzone`, monthly fee bounds on indexed columns.
- Result caching uses the same proxy-cache machinery as listings (15-minute TTL default).

## Access abilities

- PATs require `idx:access` or `idx:full` plus `X-Domain-Slug` / `?domain=` for a verified domain.
- **`idx:full` does not gate search or comps modes** in Go; both abilities receive full payloads.

## References

- `docs/idx-api-bridge-proxy.md` — proxy, cache, search endpoint
- `docs/gis-api.md` — GIS parameters and caching
- `docs/comps-api.md` — comps modes and data sources
- `docs/listings-mirror.md` — mirror payload and indexed columns
