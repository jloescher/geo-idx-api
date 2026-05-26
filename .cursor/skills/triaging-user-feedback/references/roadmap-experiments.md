# Roadmap & Experiments Feedback

## Contents
- Prioritization Framework
- Experiment Surfaces
- Validation Patterns
- Impact Measurement

## Prioritization Framework

Roadmap decisions for Quantyra IDX should be data-driven using signals from audit logs and operational metrics, not gut feeling.

### Signal-Based Prioritization

| Signal Source | Weight | How to Measure |
|---------------|--------|----------------|
| Multiple users requesting same feature | High | Support tickets + forum mentions |
| High error rate on specific endpoint | High | `mls_proxy_audit_logs` error frequency |
| Low adoption of existing feature | Medium | Endpoint usage distribution |
| Competitor parity gap | Medium | Market research |
| Internal operational pain | Low-Medium | Worker/scheduler error logs |

### Priority Score Formula

```
Priority = (Users Affected × Frequency) / (Implementation Effort × Risk)
```

Where:
- **Users Affected**: Count from `domains` table or audit log distinct domains
- **Frequency**: Occurrences per week from audit logs
- **Implementation Effort**: Hours estimated (1 = hours, 2 = days, 3 = weeks)
- **Risk**: Breaking change risk (1 = safe, 2 = moderate, 3 = high)

## Experiment Surfaces

### API Surface Experiments

The API proxy layer (`internal/handler/`) supports feature experimentation through:

| Surface | What Can Vary | How to Segment |
|---------|--------------|----------------|
| Search response format | Property JSON shape, included fields | By `domain_slug` or token scope |
| Cache TTL | `mls_search_cache` expiration | By dataset or endpoint |
| GIS teaser limits | Fields included in teaser response | By `idx:access` vs `idx:full` scope |
| Comps output | BPO vs home value vs investor mode | By request payload |

### WARNING: A/B testing API responses

API consumers are developers who write code against specific response shapes. Varying API responses by domain is dangerous — it creates hard-to-debug integration issues. Prefer:
1. **Feature flags on new endpoints** (not varying existing ones)
2. **Opt-in parameters** (`?include=gis` instead of automatic inclusion)
3. **Versioned endpoints** (`/api/v2/search`) for breaking changes

### Dashboard Surface Experiments

The server-rendered dashboard (`internal/handler/dashboard/handler.go`) supports simpler experimentation:

| Surface | What Can Vary | Risk |
|---------|--------------|------|
| DNS verification instructions | Copy, layout, auto-refresh | Low |
| Token creation flow | Show example, scope descriptions | Low |
| Domain listing order | Sort by status, recent activity | Low |
| Empty states | Guidance for new users | Low |

## Validation Patterns

### Pattern: Validate Feature Request with Audit Data

Before building a requested feature:

```sql
-- Check if requesters are actually active users
SELECT a.domain_slug, COUNT(*) AS requests_last_30d
FROM mls_proxy_audit_logs a
WHERE a.domain_slug IN ('requester-domain-1', 'requester-domain-2')
  AND a.created_at > NOW() - INTERVAL '30 days'
GROUP BY a.domain_slug;
```

If requesting domains have zero API activity, the feature request is theoretical. Prioritize active users' feedback.

### Pattern: Measure Impact After Ship

After shipping a change:

1. **Before/after comparison**: Same metric, two time periods
2. **Affected vs unaffected**: Domains that got the change vs those that didn't
3. **Support ticket volume**: Count of related tickets before and after

```sql
-- Before/after: compare cache hit rate after TTL change
SELECT
  CASE WHEN created_at < '2026-05-15' THEN 'before' ELSE 'after' END AS period,
  ROUND(COUNT(CASE WHEN cache_hit = 'hit' THEN 1 END)::numeric / COUNT(*) * 100, 1) AS hit_rate
FROM mls_proxy_audit_logs
WHERE created_at BETWEEN '2026-05-01' AND '2026-05-30'
GROUP BY period;
```

## Impact Measurement

### DO: Ship small, measure fast

Each roadmap item should have a measurable outcome defined before implementation:
- "Reduce DNS verification retries by 50%" (measurable from domain status history)
- "Increase search endpoint adoption to 60% of active domains" (measurable from audit logs)
- "Reduce replication lag to < 5 minutes" (measurable from sync cursors)

### DON'T: Ship without a measurement plan

If you can't measure whether a change improved things, it shouldn't be on the roadmap. Every item needs a before-state baseline.

## Related Skills

- See the **product-analytics** skill for measurement queries
- See the **engagement-adoption** skill for adoption tracking
- See the **proxy-web** skill for cache TTL experimentation