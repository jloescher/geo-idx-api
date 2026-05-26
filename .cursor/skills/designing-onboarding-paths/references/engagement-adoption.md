# Engagement & Adoption Reference

## Contents
- Engagement Signals in the Codebase
- Feature Discovery Patterns
- Adoption Metrics from Audit Logs
- DO/DON'T Patterns
- Common Errors and Solutions

## Engagement Signals in the Codebase

The audit log (`mls_proxy_audit_logs`) is the primary engagement signal. It tracks every authenticated API request.

**Key fields for engagement analysis:**

| Field | Purpose |
|-------|---------|
| `domain_slug` | Which domain is making requests |
| `token_name` | Which API key was used |
| `request_type` | Endpoint accessed (proxy, search, GIS, etc.) |
| `listing_count` | Volume of data returned |
| `cache_hit` | Whether the cache served the request |
| `created_at` | Timestamp for frequency analysis |

**Weekly active domains query:**

```sql
-- new code to add — engagement measurement
SELECT domain_slug,
       COUNT(*) AS requests,
       COUNT(DISTINCT DATE(created_at)) AS active_days,
       SUM(listing_count) AS total_listings_served
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY domain_slug
ORDER BY requests DESC;
```

## Feature Discovery Patterns

The dashboard currently has no feature discovery mechanism. Adding it means extending the server-rendered HTML.

### Pattern: Conditional Feature Cards

Show feature hints based on what the user has NOT yet used:

```go
// new code to add — in Dashboard handler
type FeatureUsage struct {
    UsedSearch bool
    UsedGIS    bool
    UsedComps  bool
}

func detectFeatureUsage(auditRepo repository.AuditRepo, domainSlug string) FeatureUsage {
    // Query audit logs for distinct request_types in last 30 days
    // Map to feature flags
}
```

### WARNING: Frontend-Only Feature Discovery

**The Problem:**

```javascript
// BAD — feature hints in JS with no server awareness
if (!localStorage.getItem('seen_gis_intro')) {
    showTooltip('#gis-button', 'Try our GIS parcel lookup!')
}
```

**Why This Breaks:**
1. Shows hints for features the user already uses via API (not dashboard)
2. No visibility into whether the feature is actually useful
3. Cannot correlate hint display with adoption in analytics

**The Fix:** Base feature discovery on actual usage data from audit logs.

## Adoption Metrics from Audit Logs

Track feature adoption by correlating audit events with onboarding milestones:

| Metric | Query | Purpose |
|--------|-------|---------|
| Time to first API call | `MIN(created_at) - user.created_at` per domain | Activation speed |
| Search adoption rate | Domains with `request_type LIKE '%search%'` | Feature penetration |
| GIS usage | Domains with GIS audit entries | Premium feature adoption |
| Cache hit ratio | `SUM(cache_hit)::float / COUNT(*)` | Infrastructure efficiency |
| Token rotation | Tokens created vs revoked per domain | Security hygiene |

**Activation velocity** — how fast users go from signup to first call:

```sql
-- new code to add
SELECT u.id,
       u.created_at AS signed_up,
       MIN(a.created_at) AS first_call,
       EXTRACT(EPOCH FROM (MIN(a.created_at) - u.created_at)) / 3600 AS hours_to_first_call
FROM users u
JOIN domains d ON d.user_id = u.id
JOIN mls_proxy_audit_logs a ON a.domain_slug = d.slug
GROUP BY u.id
ORDER BY hours_to_first_call;
```

## DO/DON'T Patterns

### DO: Use audit logs for engagement measurement

```sql
-- GOOD — accurate, tamper-proof, server-side
SELECT COUNT(DISTINCT domain_slug) FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '30 days';
```

### DON'T: Add client-side analytics events

```javascript
// BAD — easily blocked, no server correlation
fetch('/analytics', { method: 'POST', body: JSON.stringify({ event: 'dashboard_view' }) })
```

### DO: Surface usage data in the dashboard

```go
// GOOD — show users their own usage to encourage deeper adoption
// In Dashboard handler, pass usage stats to template
type DashboardData struct {
    Domains        []domain.Domain
    Tokens         []domain.APIToken
    WeeklyRequests int
    TopEndpoints   []string
}
```

### DON'T: Create separate analytics tables for dashboard metrics

```sql
-- BAD — duplicates data already in mls_proxy_audit_logs
CREATE TABLE dashboard_analytics (domain_slug TEXT, event_type TEXT, created_at TIMESTAMPTZ);
```

## Common Errors and Solutions

| Error | Cause | Fix |
|-------|-------|-----|
| No engagement data for new users | Audit logs only track API requests, not dashboard views | Add a lightweight dashboard visit audit event |
| Stale feature hints | Hints based on signup date, not actual usage | Query audit logs for real usage data |
| Missing correlation between signup and usage | Users table and audit logs not joined by user | Join through `domains.user_id` |

## Integration Points

- **Audit logger**: `internal/service/audit/logger.go` — the canonical event recording mechanism.
- **Dashboard handler**: `internal/handler/dashboard/handler.go` — where usage data can be surfaced.
- See the **auth-api-token** skill for token-based request tracking.
- See the **cache-postgres** skill for cache hit/miss patterns in audit data.