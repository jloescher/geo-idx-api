# Product Analytics Reference

How Quantyra IDX tracks product usage and how to scope analytics for new features.

## Contents
- Existing Analytics Infrastructure
- Scoping Analytics for New Features
- Anti-Patterns

## Existing Analytics Infrastructure

Analytics is audit-log driven. There is no analytics SDK, no event pipeline, no Mixpanel/Amplitude.

| Component | Location | Captures |
|-----------|----------|----------|
| Audit logger | `internal/service/audit/logger.go` | Domain slug, token name, request type, listing count, IP, user ID, cache hit |
| Cache headers | Bridge handler | `X-IDX-Cache: HIT/MISS` on every response |
| Replication stats | `bridgeH.Stats` endpoint | Per-dataset sync state, timestamps |
| Queue depth | `jobs` table | Pending/processing job counts per queue |
| Scheduler logs | `cmd/scheduler` | Leader status, cron fire times |

### Audit log schema

```go
// internal/service/audit/logger.go:20
func (l *Logger) Log(c *fiber.Ctx, requestType string, listingCount *int, cacheHit *string)
```

Parameters: `requestType` distinguishes proxy/listing/search/agents/offices/openhouses. `cacheHit` records `"HIT"` or `"MISS"`. `listingCount` captures response size.

## Scoping Analytics for New Features

### Template: Analytics for a new endpoint

For every new `/api/v1/*` endpoint, the audit logger must be called:

```go
// new code to add — call existing audit logger
auditor.Log(c, "comps_run", &count, nil)
```

Acceptance criteria:
```
Given an authenticated request to [new endpoint]
When the handler completes
Then an mls_proxy_audit_logs row exists with request_type='[type]'
```

### Template: New metrics from existing data

Before adding new instrumentation, check if the query can be derived from existing audit logs:

```sql
-- Adoption rate: % of verified domains that made API calls in last 7 days
SELECT
  COUNT(DISTINCT d.domain_slug) AS verified_domains,
  COUNT(DISTINCT a.domain_slug) AS active_domains,
  ROUND(COUNT(DISTINCT a.domain_slug)::numeric /
        NULLIF(COUNT(DISTINCT d.domain_slug), 0) * 100, 1) AS adoption_pct
FROM domains d
LEFT JOIN mls_proxy_audit_logs a ON a.domain_slug = d.domain_slug
  AND a.created_at > NOW() - INTERVAL '7 days'
WHERE d.verification_status IN ('verified', 'verified_ghl');
```

### Scoping decision: audit log vs new table

| Need | Use | Example |
|------|-----|---------|
| Request-level tracking | `mls_proxy_audit_logs` | Every API call |
| Aggregate counters | New materialized view or table | Daily domain summary |
| Feature-specific events | Extend audit `request_type` values | `comps_run`, `gis_parcel` |
| Long-term trends | SQL on audit logs with date range | Monthly active domains |

## Anti-Patterns

### WARNING: Adding an analytics SDK or event pipeline

Do not introduce Segment, Mixpanel, or any client-side analytics. This is a server-side API. All analytics come from audit logs and database queries.

### WARNING: Logging full request/response bodies

`mls_proxy_audit_logs` captures metadata (request type, counts, cache status). NEVER log full MLS response payloads — they contain PII (agent phone numbers, property addresses) and are voluminous.

## See Also

- See the **queue-postgresql** skill for job queue metrics
- See the **cache-postgres** skill for cache analytics