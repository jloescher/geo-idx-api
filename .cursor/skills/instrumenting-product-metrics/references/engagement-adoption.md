# Engagement & Adoption Reference

## Contents
- Engagement Metrics from Existing Data
- Product Surfaces for Adoption Tracking
- Feature Usage Instrumentation
- Anti-Patterns

## Engagement Metrics from Existing Data

`mls_proxy_audit_logs` already captures enough for basic engagement metrics — no new instrumentation needed:

| Metric | Query Surface | Aggregation |
|--------|---------------|-------------|
| Daily Active Domains | `COUNT(DISTINCT domain_slug)` | Per `logged_at::date` |
| Requests per Domain | `COUNT(*)` grouped by `domain_slug` | Per day/week |
| Endpoint Diversity | `COUNT(DISTINCT request_type)` per domain | Per week |
| Cache Efficiency | `COUNT(*) FILTER (WHERE cache_hit = 'HIT')` / `COUNT(*)` | Global or per domain |
| Search Adoption | Rows with `request_type = 'search.listings'` | Per domain |
| Comps Adoption | Rows with `request_type = 'comps.run'` | Per domain |

### Weekly Active Domains

```sql
SELECT COUNT(DISTINCT domain_slug) AS weekly_active_domains
FROM mls_proxy_audit_logs
WHERE logged_at > NOW() - INTERVAL '7 days';
```

### Feature Adoption by Endpoint

```sql
SELECT domain_slug,
  BOOL_OR(request_type LIKE 'search%') AS uses_search,
  BOOL_OR(request_type LIKE 'comps%')  AS uses_comps,
  BOOL_OR(request_type LIKE 'pub.parcel%') AS uses_gis,
  BOOL_OR(request_type = 'properties.collection') AS uses_proxy
FROM mls_proxy_audit_logs
WHERE logged_at > NOW() - INTERVAL '30 days'
GROUP BY domain_slug;
```

## Product Surfaces for Adoption Tracking

### Currently Instrumented (via `mls_proxy_audit_logs`)

| Surface | `request_type` Values | File |
|---------|----------------------|------|
| MLS Proxy (Bridge/Spark) | `listings.*`, `properties.*`, `agents.*`, `offices.*`, `members.*`, `openhouses.*` | `internal/handler/bridge/handler.go` |
| Search | `search.listings` | `internal/service/search/service.go` |
| Comps (BPO) | `comps.run` | `internal/service/comps/service.go` |
| GIS Teaser | `pub.parcel*`, `pub.assessments`, `pub.transactions` | `internal/handler/bridge/handler.go` |

### NOT Instrumented (gaps)

| Surface | Handler | Why It Matters |
|---------|---------|----------------|
| GIS authenticated parcels | `internal/handler/gis/` | High-value feature, no usage signal |
| Image proxy | `internal/handler/images/` | Bandwidth cost driver, no usage data |
| Dashboard views | `internal/handler/dashboard/handler.go` | Cannot measure engagement with settings |
| Token create/revoke | Dashboard handler | Cannot track API key rotation behavior |
| Auth login | `internal/handler/auth/handler.go` | Cannot measure login frequency |

## Feature Usage Instrumentation

### Pattern: Extending the Audit Logger

Reuse the existing `audit.Logger.Log` pattern for non-proxy surfaces:

```go
// new code to add — reuse the audit insert pattern for GIS
func (l *Logger) LogGIS(c *fiber.Ctx, parcelID string, cacheHit bool) {
    slug, _ := c.Locals(ctxkeys.MLSDomainSlug).(string)
    hit := "MISS"
    if cacheHit { hit = "HIT" }
    _, _ = l.db.Pool.Exec(context.Background(), `
        INSERT INTO mls_proxy_audit_logs
            (domain_slug, token_name, request_type, cache_hit, ip_address, user_id)
        VALUES ($1, $2, 'gis.parcel', $3, $4, $5)
    `, slug, tokenName(c), hit, c.IP(), userID(c))
}
```

### DO: Reuse `request_type` as a Namespace

```go
// GOOD — consistent naming with existing proxy audit types
requestType: "gis.parcel.detail"
requestType: "image.proxy"
```

### DON'T: Create Separate Audit Tables Per Feature

```go
// BAD — fragments data, makes cross-feature queries harder
// Don't create: gis_audit_logs, image_audit_logs, dashboard_audit_logs
```

One table with discriminated `request_type` is the established pattern. See `internal/handler/bridge/handler.go` for the 20+ existing type values.

## Engagement Event Definitions

### Namespace: `engagement.*`

| Event | When | Properties |
|-------|------|------------|
| `engagement.feature.first_use` | First `request_type` occurrence per domain | `feature`, `domain_slug` |
| `engagement.api.volume_milestone` | 100th, 1000th request per domain | `milestone`, `domain_slug` |
| `engagement.cache_efficiency_low` | Cache HIT rate drops below 50% for a domain | `hit_rate`, `domain_slug` |

## Anti-Patterns

### WARNING: Per-Request Event Emission

Do NOT emit a `product_events` row for every API request. `mls_proxy_audit_logs` already handles per-request tracking. New product events should capture **state transitions** (first use, milestones, adoption changes), not duplicate per-request data.

### WARNING: Client-Side Analytics for API Usage

idx-api is a backend service. API usage metrics belong in the server audit log, not in a client-side analytics pixel. The `mls_proxy_audit_logs` table is the authoritative source.

See the **cache-postgres** skill for cache HIT/MISS patterns and TTL configuration.
See the **geospatial** skill for GIS handler surfaces and parcel telemetry.