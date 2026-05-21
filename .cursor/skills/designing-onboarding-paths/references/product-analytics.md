# Product Analytics Reference

## Contents
- Audit Log as Analytics Backbone
- Key Metrics and Queries
- Dashboard Analytics Patterns
- DO/DON'T Patterns

## Audit Log as Analytics Backbone

This project has no dedicated analytics platform. The `mls_proxy_audit_logs` table is the only server-side event stream. Use it as the analytics backbone — it records every authenticated API request with domain, token, endpoint, listing count, and cache status.

**Schema** (from `internal/service/audit/logger.go`):

| Field | Type | Analytics use |
|-------|------|--------------|
| `domain_slug` | text | Segment by customer |
| `token_name` | text | Segment by API key |
| `request_type` | text | Feature usage (proxy, search, GIS, comps) |
| `listing_count` | int | Volume metric |
| `cache_hit` | bool | Performance metric |
| `ip_address` | text | Geographic analysis |
| `user_id` | int | Per-user tracking |
| `created_at` | timestamptz | Time-series analysis |

### WARNING: Missing Product Analytics Events

**Detected:** No frontend/session analytics; no event tracking for dashboard actions (domain add, token create, login).

**Impact:** Cannot measure activation funnel (signup → domain → verify → first call) or correlate dashboard behavior with API usage.

**Mitigation:** Add lightweight audit events for dashboard actions alongside the existing API audit log. Use the same table with a distinct `request_type` namespace:

```go
// new code to add — extend audit logger for dashboard events
func (l *AuditLogger) LogDashboardEvent(userID int64, action string, metadata map[string]any) {
    // Insert into mls_proxy_audit_logs with request_type = 'dashboard.' + action
    // Reuses existing schema without a new table
}
```

## Key Metrics and Queries

### Monthly Active Domains (MAD)

```sql
SELECT COUNT(DISTINCT domain_slug) AS monthly_active_domains
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '30 days';
```

### Feature Penetration

```sql
SELECT
    COUNT(DISTINCT CASE WHEN request_type LIKE '%search%' THEN domain_slug END) AS search_users,
    COUNT(DISTINCT CASE WHEN request_type LIKE '%gis%' THEN domain_slug END) AS gis_users,
    COUNT(DISTINCT CASE WHEN request_type LIKE '%comps%' THEN domain_slug END) AS comps_users,
    COUNT(DISTINCT domain_slug) AS total_active_domains
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '30 days';
```

### Activation Rate (signups who made an API call within 7 days)

```sql
SELECT
    COUNT(*) AS total_signups,
    COUNT(a.domain_slug) AS activated,
    ROUND(COUNT(a.domain_slug)::numeric / COUNT(*) * 100, 1) AS activation_pct
FROM users u
JOIN domains d ON d.user_id = u.id
LEFT JOIN LATERAL (
    SELECT 1 FROM mls_proxy_audit_logs a
    WHERE a.domain_slug = d.slug
    AND a.created_at < d.created_at + INTERVAL '7 days'
    LIMIT 1
) a ON true
WHERE u.created_at > NOW() - INTERVAL '30 days';
```

## Dashboard Analytics Patterns

### Pattern: Usage Summary in Dashboard

Surface per-domain usage to help customers understand their own consumption:

```go
// new code to add — in Dashboard handler
type UsageSummary struct {
    RequestsLast24h int
    TopEndpoint     string
    CacheHitRate    float64
}
```

### Pattern: Admin Analytics Page

Add a `/dashboard/analytics` route (admin-only) showing aggregate metrics. Query the audit log directly — no pre-aggregation needed at this scale.

```go
// new code to add — admin analytics handler
func (h *Handler) Analytics() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Query monthly active domains, feature penetration, activation rate
        // Render as dashboard HTML table/charts
    }
}
```

## DO/DON'T Patterns

### DO: Query audit logs directly for analytics

```sql
-- GOOD — single source of truth, no duplication
SELECT domain_slug, COUNT(*) FROM mls_proxy_audit_logs GROUP BY domain_slug;
```

### DON'T: Create parallel analytics tables

```sql
-- BAD — duplication, drift, extra maintenance
CREATE TABLE domain_analytics (
    domain_slug TEXT PRIMARY KEY,
    request_count INT,
    last_updated TIMESTAMPTZ
);
```

### DO: Use the existing audit logger for new event types

```go
// GOOD — extends existing infrastructure
l.Log(c, "dashboard.domain_added", domainSlug, 0, false)
```

### DON'T: Build a separate event pipeline

```go
// BAD — duplicate infrastructure for no benefit
type AnalyticsEvent struct { Name string; Props map[string]any }
func TrackEvent(e AnalyticsEvent) { /* new HTTP client, new endpoint */ }
```

## Integration Points

- **Audit logger**: `internal/service/audit/logger.go` — the single event recording mechanism.
- **Config**: `internal/config/config.go` — `AuthConfig` for session and invitation TTL.
- See the **queue-postgresql** skill for async job processing patterns.
- See the **postgres** skill for query optimization and indexing.