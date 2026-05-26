# Feedback Insights

## Contents
- Feedback Collection Surfaces
- Categorization Framework
- Insight Extraction Patterns
- Quick Win Identification

## Feedback Collection Surfaces

This B2B developer platform collects feedback through operational signals rather than traditional survey mechanisms. There is no in-app feedback widget.

### Active Feedback Channels

| Channel | Type | Where Captured |
|---------|------|---------------|
| API error responses | Implicit | `mls_proxy_audit_logs`, slog |
| Dashboard error states | Implicit | `internal/handler/dashboard/handler.go` responses |
| Support email/tickets | Explicit | External (not in repo) |
| Worker/scheduler failures | Implicit | slog structured logs |
| Domain verification failures | Implicit | `domains` table `verification_status` |
| Token revocation patterns | Implicit | `tokens` table |

### WARNING: No structured feedback collection

The platform has no feedback form, NPS survey, or in-app rating system. All feedback is implicit (behavioral) or external (support emails). This means:
1. Frustrated users may silently stop using the API
2. Only highly motivated users submit support tickets
3. Behavioral data (audit logs) is the primary signal source

Adding a lightweight feedback mechanism to the dashboard is a high-value quick win.

## Categorization Framework

### Feedback Categories

| Category | Example | Typical Priority | Code Surface |
|----------|---------|-----------------|--------------|
| Auth friction | "Can't get my API key to work" | P1 | `internal/api/middleware/domain_token.go` |
| Data quality | "Listings are stale/missing" | P1 | `internal/service/sync/`, `listings` table |
| Performance | "Search is slow" | P2 | `internal/service/search/postgis.go` |
| Feature gap | "Need GIS data in my area" | P2 | `internal/handler/gis/` |
| Documentation | "How do I use the search API?" | P3 | `docs/` |
| Dashboard UX | "Can't find where to copy my token" | Quick win | `internal/handler/dashboard/` |

### Pattern: Auto-Categorize from Error Logs

```sql
-- Categorize recent errors by likely feedback type
SELECT
  CASE
    WHEN request_type IN ('properties', 'search') AND cache_hit = 'miss' THEN 'performance'
    WHEN request_type IS NULL THEN 'auth_friction'
    WHEN listing_count = 0 THEN 'data_quality'
    ELSE 'general'
  END AS category,
  COUNT(*) AS occurrences,
  COUNT(DISTINCT domain_slug) AS affected_domains
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY category
ORDER BY occurrences DESC;
```

## Insight Extraction Patterns

### Pattern: Identify Silent Churn Signals

```sql
-- Domains that were active but went silent (potential churn)
SELECT domain_slug,
       MAX(created_at) AS last_activity,
       COUNT(*) AS total_requests
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '60 days'
GROUP BY domain_slug
HAVING MAX(created_at) < NOW() - INTERVAL '14 days'
   AND COUNT(*) > 10  -- meaningful usage before silence
ORDER BY last_activity;
```

### Pattern: Extract Common Error Themes

```sql
-- Error frequency by response status (proxy errors indicate upstream issues)
SELECT
  request_type,
  COUNT(*) AS total,
  COUNT(CASE WHEN cache_hit = 'error' THEN 1 END) AS errors,
  COUNT(DISTINCT domain_slug) AS affected_domains
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY request_type
HAVING COUNT(CASE WHEN cache_hit = 'error' THEN 1 END) > 0
ORDER BY affected_domains DESC, errors DESC;
```

### Pattern: Correlate Feedback with Operational Events

When a support ticket arrives about "slow search", check:
1. PostGIS query performance (not directly in audit logs — needs slog analysis)
2. Cache miss rate for that domain during the reported time window
3. Worker queue depth at the time (replication may have been blocking)
4. Upstream MLS response times (check slog for Bridge/Spark fetch durations)

## Quick Win Identification

### Quick Win Criteria

A quick win meets ALL of these:
1. **Effort**: < 1 hour implementation
2. **Visibility**: Affects multiple users or a high-value user
3. **Measurable**: Can verify improvement with existing data

### Current Quick Win Candidates

| Candidate | Effort | Visibility | Measurement |
|-----------|--------|------------|-------------|
| Add "token shown once" warning on creation | 15 min | All new users | Support ticket reduction |
| Improve DNS verification copy with exact TXT value | 30 min | All onboarding users | Verification success rate |
| Add dataset filter to stats endpoint | 30 min | All API consumers | Endpoint usage |
| Show last sync time in dashboard | 1 hour | All users with domains | Support ticket reduction |

### DO: Ship quick wins immediately

Quick wins don't need backlog grooming. If it takes less than an hour and helps users, ship it. Track the before-state first so you can measure impact.

### DON'T: Bundle quick wins into larger releases

Quick wins lose their value if they wait for a sprint boundary. Ship them as individual commits.

## Related Skills

- See the **product-analytics** skill for measurement queries
- See the **ux** skill for error message improvements
- See the **frontend-design** skill for dashboard changes
- See the **engagement-adoption** skill for churn detection