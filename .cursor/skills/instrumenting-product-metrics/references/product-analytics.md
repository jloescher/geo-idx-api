# Product Analytics Reference

## Contents
- Data Sources
- Key Queries
- Schema Design for Product Events
- Anti-Patterns

## Data Sources

### Existing: `mls_proxy_audit_logs`

The only analytics-capable table. Schema from `migrations/00001_initial.sql`:

```sql
CREATE TABLE mls_proxy_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    logged_at TIMESTAMP NOT NULL DEFAULT NOW(),
    domain_slug VARCHAR(255) NULL,
    token_name VARCHAR(255) NULL,
    request_type VARCHAR(255) NOT NULL,
    listing_count INT NULL,
    cache_hit VARCHAR(8) NULL,
    ip_address VARCHAR(45) NULL,
    user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL
);
```

Index: `mls_proxy_audit_logs_logged_at_index` on `logged_at`.

### Missing: Product Events Table

No `product_events` table exists. Add one for non-proxy events (dashboard actions, auth, lifecycle):

```sql
-- new code to add — product events migration
CREATE TABLE product_events (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    event_name VARCHAR(255) NOT NULL,
    properties JSONB NOT NULL DEFAULT '{}',
    domain_slug VARCHAR(255) NULL,
    session_id VARCHAR(255) NULL
);
CREATE INDEX product_events_created_at ON product_events(created_at);
CREATE INDEX product_events_event_name ON product_events(event_name);
CREATE INDEX product_events_user_id ON product_events(user_id);
```

## Key Queries

### Activation Rate (Existing Data Only)

```sql
SELECT
  COUNT(*) AS total_domains,
  COUNT(DISTINCT a.domain_slug) AS active_domains
FROM domains d
LEFT JOIN (
  SELECT DISTINCT domain_slug FROM mls_proxy_audit_logs
  WHERE logged_at > NOW() - INTERVAL '30 days'
) a ON a.domain_slug = d.slug
WHERE d.verified = true;
```

### Retention Cohort by Domain

```sql
WITH weekly AS (
  SELECT domain_slug, date_trunc('week', logged_at) AS week
  FROM mls_proxy_audit_logs
  WHERE logged_at > NOW() - INTERVAL '12 weeks'
)
SELECT week, COUNT(DISTINCT domain_slug) AS active_domains
FROM weekly GROUP BY week ORDER BY week;
```

### Request Volume with Cache Efficiency

```sql
SELECT logged_at::date AS day,
       COUNT(*) AS total_requests,
       COUNT(*) FILTER (WHERE cache_hit = 'HIT') AS cache_hits,
       ROUND(COUNT(*) FILTER (WHERE cache_hit = 'HIT')::numeric
             / NULLIF(COUNT(*), 0) * 100, 1) AS cache_hit_pct
FROM mls_proxy_audit_logs
WHERE logged_at > NOW() - INTERVAL '30 days'
GROUP BY day ORDER BY day;
```

### Top Domains by Usage

```sql
SELECT domain_slug, COUNT(*) AS requests,
       COUNT(DISTINCT request_type) AS endpoints_used,
       MIN(logged_at) AS first_seen, MAX(logged_at) AS last_seen
FROM mls_proxy_audit_logs
WHERE logged_at > NOW() - INTERVAL '30 days'
GROUP BY domain_slug ORDER BY requests DESC LIMIT 10;
```

## Dashboard Analytics Endpoints

### Pattern: Admin Analytics API

```go
// new code to add — admin-only analytics endpoint
func (h *Handler) GetAnalytics(c *fiber.Ctx) error {
    isAdmin, _ := c.Locals("is_admin").(bool)
    if !isAdmin { return c.SendStatus(403) }

    rows, err := h.db.Pool.Query(c.Context(), `
        SELECT logged_at::date AS day, COUNT(*)
        FROM mls_proxy_audit_logs
        WHERE logged_at > NOW() - INTERVAL '30 days'
        GROUP BY day ORDER BY day
    `)
    // ... scan and return JSON ...
}
```

## Anti-Patterns

### WARNING: Unbounded Audit Table Growth

`mls_proxy_audit_logs` has no retention policy. At high request volumes, this table grows unbounded. Add a scheduled purge:

```sql
-- new code to add — 90-day retention
DELETE FROM mls_proxy_audit_logs WHERE logged_at < NOW() - INTERVAL '90 days';
```

Schedule via the scheduler (see `internal/scheduler/scheduler.go` for the cron job pattern).

### WARNING: COUNT(*) on Large Audit Tables

AVOID `SELECT COUNT(*) FROM mls_proxy_audit_logs` without a time filter. Always use `WHERE logged_at > NOW() - INTERVAL '...'`. The `logged_at` index supports this efficiently.

### WARNING: Missing Index for Analytics Queries

The only index is on `logged_at`. Common analytics patterns also filter by `domain_slug`. Add composite indexes when query performance degrades:

```sql
-- new code to add — if analytics queries are slow
CREATE INDEX mls_proxy_audit_logs_domain_slug ON mls_proxy_audit_logs(domain_slug, logged_at);
```

See the **postgresql** skill for JSONB queries and index patterns.
See the **queue-postgresql** skill for scheduled job patterns to run retention purges.