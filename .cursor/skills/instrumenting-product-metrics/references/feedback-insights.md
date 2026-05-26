# Feedback & Insights Reference

## Contents
- Existing Feedback Surfaces
- Audit Log as Implicit Feedback
- Error Rate as Feedback Signal
- Operational Insights from Structured Logs
- Anti-Patterns

## Existing Feedback Surfaces

idx-api has **no explicit feedback mechanism** — no feedback form, NPS survey, or support ticket integration. Feedback must be inferred from:

1. **API usage patterns** in `mls_proxy_audit_logs`
2. **Error responses** logged via `slog` in handlers
3. **Failed jobs** in `failed_jobs` table
4. **Domain verification failures** (DNS TXT check fails)

### Implicit Feedback from Audit Data

| Signal | Interpretation | Data Source |
|--------|---------------|-------------|
| Domain created but never verified | DNS setup is confusing | `domains` table: `verified = false` |
| Token created but zero API calls | Documentation or integration friction | `personal_access_tokens` + `mls_proxy_audit_logs` |
| High cache MISS rate | Upstream latency, stale data | `cache_hit` column aggregation |
| Only `properties.collection` used | Integrators may not know about search/comps | `request_type` distribution |
| Sudden drop in requests per domain | Integration broken or customer churned | `COUNT(*)` per `domain_slug` per week |
| Repeated failed_jobs for same type | Systemic sync or processing issue | `failed_jobs` table |

## Audit Log as Implicit Feedback

### Pattern: Detect Under-Utilized Features

```sql
-- Domains using only basic proxy (not search/comps/GIS)
SELECT d.slug AS domain,
       COUNT(a.id) AS total_requests,
       BOOL_OR(a.request_type LIKE 'search%') AS uses_search,
       BOOL_OR(a.request_type LIKE 'comps%')  AS uses_comps,
       BOOL_OR(a.request_type LIKE 'pub.%')    AS uses_gis
FROM domains d
LEFT JOIN mls_proxy_audit_logs a ON a.domain_slug = d.slug
  AND a.logged_at > NOW() - INTERVAL '30 days'
WHERE d.verified = true
GROUP BY d.slug
HAVING COUNT(a.id) > 0
   AND NOT BOOL_OR(a.request_type LIKE 'search%')
ORDER BY total_requests DESC;
```

### Pattern: Detect At-Risk Domains (Churn Signal)

```sql
WITH weekly AS (
  SELECT domain_slug,
         date_trunc('week', logged_at) AS week,
         COUNT(*) AS requests
  FROM mls_proxy_audit_logs
  WHERE logged_at > NOW() - INTERVAL '8 weeks'
  GROUP BY domain_slug, week
)
SELECT curr.domain_slug,
       curr.requests AS this_week,
       prev.requests AS last_week,
       ROUND((curr.requests - prev.requests)::numeric
             / NULLIF(prev.requests, 0) * 100, 1) AS pct_change
FROM weekly curr
JOIN weekly prev ON curr.domain_slug = prev.domain_slug
  AND curr.week = prev.week + INTERVAL '1 week'
WHERE prev.requests > 10
ORDER BY pct_change ASC LIMIT 20;
```

## Error Rate as Feedback Signal

### Proxy Errors

`internal/handler/bridge/handler.go` logs upstream failures via `slog`:

```go
slog.Error("upstream proxy error", "url", targetURL, "error", err)
```

To make errors queryable, record them in the audit log:

```go
// new code to add — record upstream errors in audit log
if resp != nil && resp.StatusCode() >= 400 {
    cacheHit := "ERROR"
    l.audit.Log(c, requestType, nil, &cacheHit)
}
```

### Worker Job Failures

`failed_jobs` table captures permanently failed queue jobs:

```sql
SELECT
  payload::json->>'type' AS job_type,
  LEFT(exception, 200) AS error_preview,
  COUNT(*) AS failures,
  MAX(failed_at) AS latest_failure
FROM failed_jobs
WHERE failed_at > NOW() - INTERVAL '7 days'
GROUP BY job_type, error_preview
ORDER BY failures DESC LIMIT 20;
```

## Operational Insights from Structured Logs

### Key `slog` Call Sites

| Location | What's Logged | Insight |
|----------|---------------|---------|
| `internal/queue/worker.go` | `"job failed"` with id, type, error | Processing health |
| `internal/queue/worker.go` | `"discarded legacy Laravel queue job"` | Migration completeness |
| `internal/service/sync/bridge_sync.go` | Fetch/persist cycle progress | Replication health |
| `internal/scheduler/scheduler.go` | `"scheduler leader acquired"` / `"standby"` | Multi-DC leadership |
| `internal/handler/bridge/handler.go` | `"upstream proxy error"` | MLS provider issues |

## Anti-Patterns

### WARNING: Adding a Feedback Form to an API

idx-api is an API service consumed programmatically. Adding a feedback form or survey endpoint adds integration burden. Instead:
1. Infer feedback from usage patterns (audit logs, error rates)
2. Use the dashboard for human-facing feedback (the only HTML surface)
3. Let account managers gather qualitative feedback externally

### WARNING: Ignoring `failed_jobs` as Feedback

The `failed_jobs` table is a rich signal source. If `bridge.fetch_page` failures spike, the upstream provider may be degraded. If `spark.persist_chunk` fails, the schema may have changed. Monitor this table proactively.

See the **queue-postgresql** skill for job failure handling and retry patterns.
See the **postgresql** skill for querying JSONB payloads in `failed_jobs`.