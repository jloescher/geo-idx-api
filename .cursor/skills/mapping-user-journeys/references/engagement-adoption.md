# Engagement & Adoption Reference

## Contents
- Core API Journeys
- Search Engagement
- Comps Engagement
- GIS Engagement
- Proxy Cache Interaction
- Adoption Signals

---

## Core API Journeys

After onboarding, users interact with three main API surfaces:

| Surface | Entry Point | Key Feature |
|---------|-------------|-------------|
| MLS Proxy | `GET /api/v1/properties` | Cached Bridge/Spark passthrough |
| Search | `POST /api/v1/search` | Hybrid PostGIS + upstream |
| Comps | `POST /api/v1/comps/run` | BPO, home value, investor modes |
| GIS | `GET /api/v1/gis` | Parcel data with teaser tiers |
| Images | `GET /images/:listingKey/:photoId` | Cached MLS photo delivery |

---

## Search Engagement

`POST /api/v1/search` is the primary engagement surface. The hybrid router (`internal/service/search/service.go`) decides execution path:

| Request Criteria | Route | Latency Profile |
|-----------------|-------|-----------------|
| Active/Pending only | `RoutePostgresOnly` | Fast (PostGIS local) |
| Closed only | `RouteUpstreamOnly` | Slow (Bridge/Spark HTTP) |
| Mixed statuses | `RouteSplit` | Medium (parallel) |
| Price reduced | `RouteUpstreamOnly` | Slow |
| Empty/no status | `RoutePostgresOnly` | Fast (default) |

### Search Request Fields (engagement drivers)

Key fields in `SearchRequest` that drive repeat usage:

- `statuses`, `min_price`, `max_price` — basic filtering
- `lat`, `lng`, `radius_miles` — geographic search (PostGIS `ST_DWithin`)
- `property_type`, `city`, `postal_code` — categorical filtering
- `low_risk_flood_zone`, `pool`, `waterfront` — premium filters
- `price_reduced_within_days` — triggers upstream-only route

### Response Envelope

```json
{
  "results": [...],
  "total": 42,
  "has_more": true,
  "next_skip": 25
}
```

Pagination via `skip`/`limit`. Hard cap at 200 results per request.

---

## Comps Engagement

`POST /api/v1/comps/run` supports multiple analysis modes (`internal/service/comps/types.go`):

| Mode | Purpose | Key Input |
|------|---------|-----------|
| `a_e_sales` | Standard comparable sales | Subject + scope + filters |
| `bpo` | Broker price opinion | Subject + adjustments |
| `home_value` | Automated valuation | Subject only |
| `rent_hold_cashflow` | Rental investment analysis | Subject + rent assumptions |
| `flip_vs_hold` | Flip/hold comparison | Subject + rehab costs |
| `appraiser_simulation` | Appraiser workflow | Subject + adjustments |

Limits: max 12 sold comps, 20 competition comps per run.

Engagement pattern: users start with `home_value` (simplest), graduate to `bpo` or `a_e_sales` for detailed analysis.

---

## GIS Engagement

`GET /api/v1/gis` provides parcel data with tiered access:

- **Unauthenticated**: teaser subset (`GIS.TelemetryMaxFeatures` = 40 features)
- **Authenticated (`idx:access`)**: full parcel data

Bounding box limit: `GIS.MaxBboxSpanDeg` = 0.35 degrees per request.

Cache layers: `GIS.EdgeCacheTTL` (900s), origin max age primary (90 days), county (30 days).

---

## Proxy Cache Interaction

Every MLS proxy request passes through `internal/service/cache/proxy.go`:

```
Request → FingerprintRequest() → cache.Get()
  → HIT: return cached JSON + X-IDX-Cache: HIT
  → MISS: proxy to upstream → cache.Put() → X-IDX-Cache: MISS
```

Cache partitions isolate data per domain:

- `cache.WebPartition(domainSlug, feedCode, auditType)` — listings/agents/offices
- `cache.ResoPartition(domainSlug, feedCode, entity)` — properties/members
- `cache.LookupPartition(domainSlug, feedCode)` — RESO lookup

TTL: `Bridge.ListingsCacheTTL` = 900s (15 min), `Bridge.LookupCacheTTL` = 720h (30 days).

### WARNING: Cache Stampede on Miss

**The Problem:** No request coalescing. Two identical requests during a cache miss trigger two upstream calls.

**Why This Breaks:** During high traffic, cache expiry causes upstream thundering herd.

**When You Might Be Tempted:** Adding singleflight would help, but verify it doesn't block the Fiber event loop.

---

## Adoption Signals

Track these signals to measure engagement depth:

| Signal | Source | Query |
|--------|--------|-------|
| API call volume | `mls_proxy_audit_logs` | `COUNT(*) GROUP BY domain_slug` |
| Cache hit rate | `mls_proxy_audit_logs.cache_hit` | `COUNT(*) FILTER (WHERE cache_hit = 'HIT') / COUNT(*)` |
| Search usage | audit `request_type = 'search'` | Per-domain search count |
| Comps usage | audit `request_type` containing 'comps' | Mode distribution |
| GIS usage | audit `request_type` containing 'gis' | Authenticated vs teaser ratio |
| Token usage | `personal_access_tokens.last_used_at` | Dormant token detection |

See the **cache-postgres** skill for cache internals.
See the **proxy-web** skill for proxy fingerprinting details.