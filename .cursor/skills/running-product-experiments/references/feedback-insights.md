# Feedback & Insights Reference

## Contents
- Feedback Surfaces
- Audit-Driven Insights
- Structured Feedback Collection
- Insight Extraction Patterns
- Anti-Patterns

---

## Feedback Surfaces

idx-api has no built-in feedback mechanism. Available signal sources:

| Source | Location | Signal type |
|--------|----------|-------------|
| Audit logs | `mls_proxy_audit_logs` | Usage patterns, errors |
| Error responses | Handler return values | Friction points |
| Support contact | External (email/Slack) | Qualitative |
| Sync stats | `GET /api/v1/bridge/stats` | Data quality issues |
| DNS verification failures | `domains` table | Onboarding friction |

## Audit-Driven Insights

### Extract pain points from error rates

```sql
-- Domains with highest error rates (proxy failures logged by request_type)
SELECT domain_slug,
       COUNT(*) AS total_requests,
       COUNT(*) FILTER (WHERE cache_hit = 'miss') AS cache_misses,
       ROUND(100.0 * COUNT(*) FILTER (WHERE cache_hit = 'miss') / COUNT(*), 1) AS miss_pct
FROM mls_proxy_audit_logs
WHERE created_at > now() - INTERVAL '7 days'
GROUP BY 1
HAVING COUNT(*) FILTER (WHERE cache_hit = 'miss') > 0
ORDER BY miss_pct DESC;
```

### Identify under-engaged domains

```sql
-- Domains that verified but never made an API call
SELECT d.slug
FROM domains d
WHERE d.verified_at IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM mls_proxy_audit_logs l
    WHERE l.domain_slug = d.slug
  );
```

These domains likely hit a friction point between verification and first API call.

## Structured Feedback Collection

### DO: Add feedback endpoint with minimal friction

```go
// new code to add
// POST /api/v1/feedback — token-authenticated
type FeedbackRequest struct {
    Category string `json:"category"` // "bug", "feature", "confusion"
    Message  string `json:"message"`
    URL      string `json:"url"`      // page or endpoint context
}

// Store in feedback table, no email required
```

### DO: Tag audit logs with error categories

Extend `Logger.Log()` with an `error_category` field for classifying proxy failures:

```go
// new code to add
type AuditOpts struct {
    ErrorCategory *string // "timeout", "auth", "upstream_5xx", "invalid_query"
}
```

## Insight Extraction Patterns

1. **Weekly digest query** — Aggregate top errors, under-engaged domains, new activations
2. **Support correlation** — When support contacts arrive, cross-reference audit logs for that domain's recent activity
3. **Feature request clustering** — Group feedback by category, rank by domain count

```sql
-- new code to add — feedback summary view
CREATE VIEW v_feedback_summary AS
SELECT category,
       COUNT(*) AS mentions,
       COUNT(DISTINCT domain_slug) AS unique_domains,
       array_agg(DISTINCT message ORDER BY created_at DESC) FILTER (WHERE created_at > now() - INTERVAL '30 days') AS recent_messages
FROM feedback
GROUP BY category
ORDER BY unique_domains DESC;
```

## Anti-Patterns

### WARNING: Treating all feedback equally

A single loud user is not product insight. Weight feedback by domain engagement — a power user's bug report matters more than a never-activated domain's feature request.

**Fix:** Always join feedback against usage data before prioritizing.

### WARNING: Collecting feedback without acting on it

A `/feedback` endpoint that writes to an unmonitored table is theater. Either commit to a review cadence (weekly) or don't build it.

See the **queue-postgresql** skill for scheduling feedback aggregation jobs and the **auth-api-token** skill for securing feedback endpoints.