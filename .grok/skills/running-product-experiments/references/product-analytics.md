# Product Analytics Reference

## Contents
- Current Analytics Infrastructure
- Extending Audit Logging
- Aggregation Strategy
- Experiment Metric Queries
- Anti-Patterns

---

## Current Analytics Infrastructure

idx-api has one analytics mechanism: **audit logging** in `internal/service/audit/logger.go`.

```go
// Existing audit fields
INSERT INTO mls_proxy_audit_logs
  (domain_slug, token_name, request_type, listing_count, ip_address, user_id, cache_hit)
```

**What exists:** Request-level proxy audit.
**What's missing:** Funnel tracking, event taxonomy, cohort analysis, retention metrics, experiment assignment.

## Extending Audit Logging

### DO: Add experiment columns via migration

```sql
-- new code to add — migrations/YYYYMMDDHHMMSS_add_experiment_columns.sql
ALTER TABLE mls_proxy_audit_logs
  ADD COLUMN experiment_id TEXT,
  ADD COLUMN variant      TEXT;
CREATE INDEX idx_audit_experiment ON mls_proxy_audit_logs (experiment_id);
```

Then extend `Logger.Log()` to accept optional experiment metadata:

```go
// new code to add
type AuditOpts struct {
    ExperimentID *string
    Variant      *string
}

func (l *Logger) LogWithOpts(c *fiber.Ctx, requestType string, listingCount *int, cacheHit *string, opts AuditOpts) {
    // ... existing insert with experiment_id, variant
}
```

### DON'T: Create a parallel event system alongside audit logs

Two event pipelines means two sources of truth. Extend the existing table rather than building a separate `events` table unless the schema is fundamentally different.

## Aggregation Strategy

### WARNING: Querying raw audit logs at scale

`mls_proxy_audit_logs` grows unbounded. Analytical queries on raw data degrade after ~1M rows.

**Fix:** Add a scheduled aggregation job (see the **queue-postgresql** skill):

```sql
-- new code to add — daily rollup table
CREATE TABLE daily_domain_metrics (
    date         DATE NOT NULL,
    domain_slug  TEXT NOT NULL,
    request_type TEXT NOT NULL,
    request_count INT NOT NULL DEFAULT 0,
    avg_listing_count NUMERIC,
    cache_hit_rate   NUMERIC,
    PRIMARY KEY (date, domain_slug, request_type)
);
```

Scheduler cron: `40 1 * * *` → `analytics.aggregate_daily` job type.

## Experiment Metric Queries

```sql
-- Variant conversion: % of domains making first search within 7 days
WITH assigned AS (
    SELECT domain_slug, variant,
           MIN(created_at) AS assigned_at
    FROM mls_proxy_audit_logs
    WHERE experiment_id = 'new_search_ui'
    GROUP BY 1, 2
),
converted AS (
    SELECT a.domain_slug, a.variant,
           MIN(l.created_at) AS first_search
    FROM assigned a
    JOIN mls_proxy_audit_logs l ON l.domain_slug = a.domain_slug
        AND l.request_type = 'search'
        AND l.created_at <= a.assigned_at + INTERVAL '7 days'
    GROUP BY 1, 2
)
SELECT a.variant,
       COUNT(*) AS assigned,
       COUNT(c.first_search) AS converted,
       ROUND(100.0 * COUNT(c.first_search) / COUNT(*), 1) AS conversion_pct
FROM assigned a
LEFT JOIN converted c USING (domain_slug, variant)
GROUP BY 1;
```

## Anti-Patterns

### WARNING: Blocking request processing on analytics writes

Never make analytics inserts part of a transaction that also does business logic. If analytics DB is slow, the API slows.

**Fix:** Fire-and-forget goroutine or enqueue a job for analytics writes. Audit logs already use this pattern — the insert happens after the response is sent.

See the **queue-postgresql** skill for async job patterns and the **postgresql** skill for migration patterns.