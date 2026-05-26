# Feedback & Insights Reference

## Contents
- Signal Sources
- Audit Log as Feedback Channel
- Bridge Stats as Health Signal
- Error Signal Extraction
- Anti-Patterns

## Signal Sources

The platform has limited explicit feedback mechanisms. Product insights must be derived from operational signals.

### Available Signals

| Signal | Source | Granularity | Location |
|--------|--------|-------------|----------|
| API usage volume | `mls_proxy_audit_logs` | Per request | `audit/logger.go` |
| Cache performance | `mls_proxy_audit_logs.cache_hit` | Per request | `audit/logger.go` |
| Replication health | `listing_sync_cursors` | Per sync cycle | `sync/` services |
| Token churn | `tokens.revoked_at` | Per token | `domains` table |
| Domain activity | `domains.active` | Per domain | `domains` table |
| Job failure rate | `jobs` table (transient) | Per job | `queue/` package |
| GIS teaser triggers | Request type in audit log | Per request | `teaser.go` |

## Audit Log as Feedback Channel

The `mls_proxy_audit_logs` table is the richest signal source for product insights.

### Querying for Insights

```sql
-- Domains with declining usage (potential churn)
SELECT domain_slug,
       COUNT(*) AS requests_this_month,
       LAG(COUNT(*)) OVER (PARTITION BY domain_slug ORDER BY DATE_TRUNC('month', created_at)) AS prev_month
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '3 months'
GROUP BY domain_slug, DATE_TRUNC('month', created_at);

-- Feature adoption by domain
SELECT domain_slug,
       COUNT(DISTINCT request_type) AS features_used,
       array_agg(DISTINCT request_type) AS feature_list
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY domain_slug;

-- Token utilization (unused tokens = setup friction)
SELECT t.name, t.domain_slug, COUNT(a.id) AS api_calls
FROM tokens t
LEFT JOIN mls_proxy_audit_logs a ON a.token_name = t.name AND a.created_at > NOW() - INTERVAL '30 days'
WHERE t.revoked_at IS NULL
GROUP BY t.name, t.domain_slug
HAVING COUNT(a.id) = 0;
```

## Bridge Stats as Health Signal

`GET /api/v1/bridge/stats` provides replication state — a proxy for data freshness and system health:

| Metric | Meaning | Action |
|--------|---------|--------|
| `replication_in_progress` | Sync running | Wait, don't force kickoff |
| `last_sync_finished_at` | Data freshness | Alert if stale beyond threshold |
| `total_listings` | Mirror completeness | Compare to upstream counts |

## Error Signal Extraction

### HTTP Error Rates

```sql
-- High error domains (potential integration issues)
SELECT domain_slug, request_type,
       COUNT(*) FILTER (WHERE /* error indicator */) AS errors,
       COUNT(*) AS total
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY domain_slug, request_type;
```

### Job Failure Patterns

Workers log failures to `slog`. Failed jobs remain in `jobs` table briefly before retry. Monitor with:

```sql
SELECT type, queue, COUNT(*) AS failed_count
FROM jobs
WHERE status = 'failed'
GROUP BY type, queue;
```

## Anti-Patterns

### WARNING: Building a Feedback UI

**The Problem:** Adding a feedback form or rating widget to the dashboard.

**Why This Breaks:** The platform is invite-only B2B with few users. A feedback UI adds complexity for minimal signal. Direct communication (email, Slack) is more effective for this user count.

**The Fix:** Use audit log queries to identify friction points. Reach out directly to domains with declining usage or unused tokens.

### WARNING: Survey Emails for API Consumers

**The Problem:** Sending NPS or satisfaction surveys to API consumers.

**Why This Breaks:** API consumers are developers integrating an API, not end users clicking buttons. Surveys add noise to their workflow.

**The Fix:** Monitor API usage patterns. Declining request counts, unused tokens, and failed integrations are stronger signals than survey responses. If qualitative feedback is needed, do it in person or via the existing admin relationship.

### WARNING: Interpreting Teaser Hits as Dissatisfaction

**The Problem:** Counting GIS teaser responses as negative feedback.

**Why This Breaks:** Teaser responses indicate the endpoint is being evaluated — not that users are unhappy. It's the top of the adoption funnel.

**The Fix:** Track the conversion from teaser to `idx:full` token creation. That's the real signal: users who tried GIS and upgraded.

## Workflow Checklist

Copy this checklist for feedback analysis tasks:
- [ ] Step 1: Identify the signal source (audit logs, tokens, domains, jobs)
- [ ] Step 2: Write SQL query against `mls_proxy_audit_logs`
- [ ] Step 3: Segment by `domain_slug` for per-customer insight
- [ ] Step 4: Cross-reference with `tokens` table for token-level attribution
- [ ] Step 5: Check `listing_sync_cursors` for data freshness correlation
- [ ] Step 6: Document findings — do not automate actions on correlations

See the **postgres** skill for SQL query patterns and migration tools.
See the **queue-postgresql** skill for job monitoring and failure handling.