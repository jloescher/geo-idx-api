# Product Analytics Reference

## Contents
- Audit Log Architecture
- Adoption Metrics
- Usage Tracking Patterns
- Data Availability
- Anti-Patterns

## Audit Log Architecture

The platform uses `mls_proxy_audit_logs` as the primary analytics store. The audit logger (`internal/service/audit/logger.go`) records every MLS API request.

### Existing Audit Schema

The `Log()` function captures:

| Field | Source | Purpose |
|-------|--------|---------|
| `domain_slug` | `ctxkeys.MLSDomainSlug` | Per-domain usage |
| `token_name` | `ctxkeys.MLSTokenName` | Per-token attribution |
| `user_id` | `ctxkeys.MLSUserID` | User-level tracking |
| `ip_address` | `c.IP()` | Client location |
| `request_type` | Handler argument | API endpoint used |
| `listing_count` | Handler argument | Volume metric |
| `cache_hit` | Handler argument | Performance signal |
| `created_at` | `now()` | Timestamp |

### Context Key Flow

Middleware sets context keys before the audit logger reads them:

```go
// existing pattern — domain_token.go sets context
c.Locals(ctxkeys.MLSDomainSlug, domain.Slug)
c.Locals(ctxkeys.MLSTokenName, token.Name)
c.Locals(ctxkeys.MLSUserID, domain.UserID)
```

```go
// existing pattern — audit/logger.go reads context
func (l *Logger) Log(c *fiber.Ctx, requestType string, listingCount *int, cacheHit *string) {
    domainSlug, _ := c.Locals(ctxkeys.MLSDomainSlug).(string)
    tokenName, _ := c.Locals(ctxkeys.MLSTokenName).(string)
    userID, _ := c.Locals(ctxkeys.MLSUserID).(string)
    // INSERT INTO mls_proxy_audit_logs
}
```

## Adoption Metrics

### Key Metrics Derivable from Audit Logs

| Metric | Query Pattern | Business Meaning |
|--------|--------------|------------------|
| First API call | `MIN(created_at) WHERE domain_slug = ?` | Activation timestamp |
| DAU/MAU | `COUNT(DISTINCT domain_slug) WHERE created_at BETWEEN` | Engagement frequency |
| Feature penetration | `COUNT(DISTINCT request_type) WHERE domain_slug = ?` | Feature adoption breadth |
| Search volume | `SUM(listing_count) WHERE request_type = 'search'` | Core usage intensity |
| Cache hit rate | `COUNT(*) WHERE cache_hit = 'hit'` / total | Performance + freshness |
| GIS teaser exposure | `COUNT(*) WHERE request_type LIKE '%gis%'` | Upsell opportunity |

### Activation Funnel

```
Invitation accepted → Account created → First login → Domain added →
Domain verified → Token created → First API call (activated)
```

Each step is queryable from existing tables: `users`, `domains`, `tokens`, `mls_proxy_audit_logs`.

## Usage Tracking Patterns

### Adding a New Trackable Event

```go
// new code to add — extend audit logger
func (l *Logger) LogFeatureUse(c *fiber.Ctx, feature string, meta map[string]any) {
    domainSlug, _ := c.Locals(ctxkeys.MLSDomainSlug).(string)
    userID, _ := c.Locals(ctxkeys.MLSUserID).(string)
    // INSERT INTO feature_usage (domain_slug, user_id, feature, meta, created_at)
    // Fire-and-forget: use background INSERT, do not block response
}
```

### Bridge Stats as Usage Dashboard

`GET /api/v1/bridge/stats` provides replication and mirror statistics — a built-in health/usage endpoint for MLS data freshness.

## Data Availability

| Data Source | Granularity | Retention |
|------------|-------------|-----------|
| `mls_proxy_audit_logs` | Per request | Indefinite (purge policy TBD) |
| `domains` | Per domain | Indefinite |
| `tokens` | Per token | Until revoked |
| `listings` | Per listing | Rolling window (configurable) |
| `jobs` | Per job | Deleted after completion |

### WARNING: No Product Analytics Service

**Detected:** No dedicated analytics service (Amplitude, Mixpanel, PostHog) in dependencies.

**Impact:** Adoption metrics require SQL queries against audit logs. No real-time dashboards, no funnel visualization, no cohort analysis out of the box.

**Current state:** Acceptable for B2B API platform. The audit log table provides raw data for ad-hoc analysis.

**If analytics needs grow:** Consider adding PostHog (self-hosted option available) or building a materialized view on `mls_proxy_audit_logs` with `pg_cron` aggregation.

## Anti-Patterns

### WARNING: Logging to Stdout for Analytics

**The Problem:**

```go
// BAD — structured log as analytics
slog.Info("feature used", "feature", "gis", "domain", slug)
```

**Why This Breaks:** Structured logs are for operational debugging. They lack queryability, are rotated away, and cannot be aggregated for product metrics.

**The Fix:** Use the audit log pattern — INSERT to a dedicated table. Structured logging with `slog` is for operational concerns (see `internal/` logging patterns).

### WARNING: Counting API Tokens as Adoption

**The Problem:** Treating token creation as the activation metric.

**Why This Breaks:** Tokens can be created and never used. Activation requires a successful API call.

**The Fix:** Define activation as `MIN(mls_proxy_audit_logs.created_at) WHERE domain_slug = ?` — the first actual API usage.

See the **queue-postgresql** skill for background job patterns that could support analytics aggregation.
See the **postgres** skill for query patterns on audit data.