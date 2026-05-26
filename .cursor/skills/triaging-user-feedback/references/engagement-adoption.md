# Engagement & Adoption Feedback

## Contents
- Engagement Signal Sources
- Adoption Metrics from Audit Logs
- Feature-Specific Adoption Patterns
- Declining Engagement Detection

## Engagement Signal Sources

Engagement for a B2B developer platform is measured through API usage patterns, not page views. The primary signal source is `mls_proxy_audit_logs` (see `internal/service/audit/logger.go`).

### Key Metrics

| Metric | Source | Healthy Range |
|--------|--------|---------------|
| Daily active domains | `audit_logs` distinct `domain_slug` per day | Depends on customer count |
| Cache hit rate | `audit_logs.cache_hit` field | > 70% for repeated queries |
| Request type diversity | `audit_logs.request_type` distribution | Using multiple endpoints |
| Search usage | `POST /api/v1/search` frequency | Growing week-over-week |
| Image proxy usage | `/images/*` in audit logs | Correlates with listing views |

### Engagement Query Patterns

```sql
-- Weekly engagement trend by domain
SELECT domain_slug,
       DATE_TRUNC('week', created_at) AS week,
       COUNT(*) AS requests,
       COUNT(DISTINCT request_type) AS endpoint_variety
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '8 weeks'
GROUP BY domain_slug, DATE_TRUNC('week', created_at)
ORDER BY domain_slug, week;
```

## Adoption Metrics from Audit Logs

### Feature Adoption Tracking

The platform exposes these feature surfaces. Track adoption by monitoring which endpoints each domain uses:

| Feature | Endpoint | Adoption Signal |
|---------|----------|-----------------|
| MLS proxy | `GET /api/v1/properties` | Basic usage |
| Search | `POST /api/v1/search` | Advanced usage |
| GIS parcels | `GET /api/v1/gis` | Premium feature |
| Comps/BPO | `POST /api/v1/comps/run` | High-value feature |
| Image proxy | `/images/*` | Essential feature |
| Bridge stats | `GET /api/v1/bridge/stats` | Operational monitoring |

```sql
-- Feature adoption matrix per domain
SELECT domain_slug,
       COUNT(CASE WHEN request_type = 'properties' THEN 1 END) > 0 AS uses_proxy,
       COUNT(CASE WHEN request_type = 'search' THEN 1 END) > 0 AS uses_search,
       COUNT(CASE WHEN request_type = 'gis' THEN 1 END) > 0 AS uses_gis,
       COUNT(CASE WHEN request_type = 'comps' THEN 1 END) > 0 AS uses_comps
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY domain_slug;
```

## Feature-Specific Adoption Patterns

### DO: Correlate feature adoption with account age

New accounts that start with search (not just proxy) are power users. Domains that only hit the proxy after 30 days may need guidance.

### DON'T: Count dashboard logins as engagement

Dashboard logins are setup events, not ongoing engagement. Real engagement is API calls. A domain with zero API calls but weekly dashboard logins is likely stuck on configuration.

### WARNING: Treating all domains equally

Domains on the `stellar` dataset (Bridge) and `beaches` dataset (Spark) may have different usage patterns. Segment analysis by `dataset_slug` to avoid misleading averages.

```sql
-- Adoption by dataset
SELECT d.dataset_slugs,
       COUNT(DISTINCT a.domain_slug) AS active_domains,
       AVG(req_count) AS avg_requests
FROM domains d
LEFT JOIN LATERAL (
    SELECT domain_slug, COUNT(*) AS req_count
    FROM mls_proxy_audit_logs a
    WHERE a.domain_slug = d.slug
      AND a.created_at > NOW() - INTERVAL '30 days'
    GROUP BY domain_slug
) a ON true
GROUP BY d.dataset_slugs;
```

## Declining Engagement Detection

```sql
-- Domains with declining usage (compare last 2 weeks to prior 2 weeks)
SELECT recent.domain_slug,
       recent.requests AS last_2_weeks,
       prior.requests AS prior_2_weeks,
       ROUND((recent.requests - prior.requests)::numeric / NULLIF(prior.requests, 0) * 100, 1) AS pct_change
FROM (
    SELECT domain_slug, COUNT(*) AS requests
    FROM mls_proxy_audit_logs
    WHERE created_at > NOW() - INTERVAL '14 days'
    GROUP BY domain_slug
) recent
JOIN (
    SELECT domain_slug, COUNT(*) AS requests
    FROM mls_proxy_audit_logs
    WHERE created_at BETWEEN NOW() - INTERVAL '28 days' AND NOW() - INTERVAL '14 days'
    GROUP BY domain_slug
) prior ON prior.domain_slug = recent.domain_slug
WHERE recent.requests < prior.requests;
```

## Related Skills

- See the **product-analytics** skill for deeper metric patterns
- See the **cache-postgres** skill for cache hit rate optimization
- See the **auth-api-token** skill for token usage patterns