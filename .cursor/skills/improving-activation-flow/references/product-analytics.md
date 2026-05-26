# Product Analytics Reference

## Contents
- Analytics architecture
- Audit log as event source
- Funnel queries
- Replication metrics
- Anti-patterns

## Analytics Architecture

This project uses PostgreSQL-native analytics. No external analytics services (Mixpanel, Amplitude, PostHog). The `audit_logs` table is the event source of truth.

### DO: Extend `audit_logs` for new event types

```sql
-- Existing audit_logs pattern (from migration)
INSERT INTO audit_logs (action, subject_type, subject_id, metadata, created_at)
VALUES ('api.request', 'domain', $1, '{"endpoint": "/api/v1/search", "status": 200}', NOW());
```

Add new actions as needed: `token.created`, `domain.activated`, `search.first_use`, `gis.first_use`, `comps.first_use`.

### DON'T: Create a separate events table

A second table duplicates the audit infrastructure. Add new `action` values to the existing table. The `metadata` JSONB column handles arbitrary event properties.

## Audit Log as Event Source

### Activation funnel query

```sql
-- Time from domain creation to first successful API call
WITH first_token AS (
    SELECT domain_id, MIN(created_at) AS token_at
    FROM tokens
    GROUP BY domain_id
),
first_call AS (
    SELECT subject_id AS domain_id, MIN(created_at) AS call_at
    FROM audit_logs
    WHERE action = 'api.request'
    GROUP BY subject_id
)
SELECT d.hostname,
       d.created_at AS domain_created,
       ft.token_at,
       fc.call_at,
       EXTRACT(EPOCH FROM (fc.call_at - d.created_at))/60 AS minutes_to_first_call
FROM domains d
LEFT JOIN first_token ft ON ft.domain_id = d.id
LEFT JOIN first_call fc ON fc.domain_id = d.id
ORDER BY d.created_at DESC
LIMIT 20;
```

### DO: Use the funnel to identify drop-off

If most domains have a token but `call_at` is NULL, the problem is between token creation and first API use — improve the post-token-creation guidance.

## Funnel Queries

### Weekly activation rate

```sql
SELECT
    DATE_TRUNC('week', d.created_at) AS week,
    COUNT(DISTINCT d.id) AS domains_created,
    COUNT(DISTINCT ft.domain_id) AS tokens_created,
    COUNT(DISTINCT fc.domain_id) AS first_call_made
FROM domains d
LEFT JOIN (SELECT domain_id, MIN(created_at) AS at FROM tokens GROUP BY domain_id) ft ON ft.domain_id = d.id
LEFT JOIN (SELECT subject_id AS domain_id, MIN(created_at) AS at FROM audit_logs WHERE action = 'api.request' GROUP BY subject_id) fc ON fc.domain_id = d.id
GROUP BY 1
ORDER BY 1 DESC;
```

### Feature adoption by domain

```sql
SELECT d.hostname,
       BOOL_OR(a.action = 'search.request') AS uses_search,
       BOOL_OR(a.action = 'gis.request') AS uses_gis,
       BOOL_OR(a.action = 'comps.request') AS uses_comps
FROM domains d
LEFT JOIN audit_logs a ON a.subject_id = d.id AND a.subject_type = 'domain'
GROUP BY d.hostname;
```

## Replication Metrics

Replication health directly affects product experience. Track via `listing_sync_cursors`:

```sql
SELECT dataset_slug,
       replication_in_progress,
       last_sync_finished_at,
       EXTRACT(EPOCH FROM (NOW() - last_sync_finished_at))/60 AS minutes_since_sync
FROM listing_sync_cursors;
```

See the **queue-postgresql** skill for worker/scheduler health.

## Anti-patterns

### WARNING: Adding analytics in the API hot path

```go
// BAD — synchronous external call blocks the response
func SearchHandler(c *fiber.Ctx) error {
    results, _ := searchService.Search(ctx, params)
    http.Post("https://analytics.example.com/track", ...) // DO NOT DO THIS
    return c.JSON(results)
}

// GOOD — audit log write is co-located in same DB transaction
func SearchHandler(c *fiber.Ctx) error {
    results, _ := searchService.Search(ctx, params)
    auditRepo.Record(ctx, "search.request", domainID, metadata) // same DB, fast
    return c.JSON(results)
}
```

### WARNING: Using `SELECT COUNT(*)` on audit_logs for real-time counters

`audit_logs` grows unbounded. For real-time counters (e.g., "requests today"), use a summary table or materialized view — not a full table scan.