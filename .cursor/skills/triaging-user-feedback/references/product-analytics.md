# Product Analytics Feedback

## Contents
- Analytics Data Sources
- Metric Queries
- Funnel Analysis
- Data Quality Monitoring

## Analytics Data Sources

This platform does not use a dedicated analytics service. Product analytics are derived from PostgreSQL tables that already exist for operational purposes.

| Source | Table | Analytics Use |
|--------|-------|---------------|
| API audit logs | `mls_proxy_audit_logs` | Usage patterns, endpoint adoption, cache performance |
| Domain records | `domains` | Activation funnel, verification rates |
| Token records | `tokens` | Token lifecycle, scope usage |
| Listings mirror | `listings` | Data coverage, dataset health |
| Sync cursors | `listing_sync_cursors` | Replication freshness |
| Job queue | `jobs` | Processing health (ephemeral — completed jobs deleted) |

### WARNING: Job queue is ephemeral

Completed jobs are **deleted** from `jobs` after success (normal queue behavior). You cannot query historical job performance from the `jobs` table. Use slog structured logs or add a `job_history` table if retrospective job analytics are needed.

## Metric Queries

### API Usage Metrics

```sql
-- Daily request volume with cache performance
SELECT DATE(created_at) AS day,
       COUNT(*) AS total_requests,
       COUNT(CASE WHEN cache_hit = 'hit' THEN 1 END) AS cache_hits,
       ROUND(COUNT(CASE WHEN cache_hit = 'hit' THEN 1 END)::numeric / NULLIF(COUNT(*), 0) * 100, 1) AS hit_rate_pct
FROM mls_proxy_audit_logs
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY day;
```

### Dataset Coverage

```sql
-- Listings coverage by dataset (data quality metric)
SELECT dataset_slug,
       COUNT(*) AS total,
       COUNT(list_price) AS with_price,
       COUNT(coordinates) AS with_geom,
       COUNT(flood_zone_code) AS with_flood
FROM listings
GROUP BY dataset_slug;
```

### Replication Health

```sql
-- Sync freshness per dataset
SELECT dataset_slug,
       last_sync_finished_at,
       replication_in_progress,
       last_modification_timestamp
FROM listing_sync_cursors;
```

## Funnel Analysis

### Full User Funnel

```sql
-- From invitation to active API consumer
WITH invited AS (SELECT id, email, created_at FROM users),
     with_domain AS (SELECT user_id, MIN(created_at) AS domain_at FROM domains GROUP BY user_id),
     verified AS (SELECT user_id, MIN(verified_at) AS verified_at FROM domains WHERE verification_status = 'verified' GROUP BY user_id),
     with_token AS (SELECT domain_id, MIN(created_at) AS token_at FROM tokens GROUP BY domain_id),
     active AS (SELECT DISTINCT domain_slug FROM mls_proxy_audit_logs WHERE created_at > NOW() - INTERVAL '7 days')
SELECT
  (SELECT COUNT(*) FROM invited) AS total_users,
  (SELECT COUNT(*) FROM with_domain) AS added_domain,
  (SELECT COUNT(*) FROM verified) AS verified_domain,
  (SELECT COUNT(*) FROM with_token) AS created_token,
  (SELECT COUNT(*) FROM active) AS active_last_7d;
```

## Data Quality Monitoring

### DO: Use existing stats endpoint for monitoring

`GET /api/v1/bridge/stats` (see `internal/service/sync/stats.go`) provides real-time replication health. Use it as a data quality signal source.

### DON'T: Query listings table for real-time analytics

The `listings` table can be large. For real-time analytics, prefer aggregate counts from `mls_proxy_audit_logs` with appropriate time filters rather than full table scans on `listings`.

### Pattern: Detect Data Gaps from User Feedback

When users report "missing listings" or "stale data":

1. Check `listing_sync_cursors.last_sync_finished_at` — is it recent?
2. Check `listings` count by `dataset_slug` — compare to expected volume
3. Check `replica_pages` for stuck `pending`/`processing` rows
4. Check worker logs for fetch/persist errors

```sql
-- Stuck replica pages (replication issue indicator)
SELECT provider, dataset, status, COUNT(*), MIN(created_at) AS oldest
FROM replica_pages
WHERE status IN ('pending', 'processing')
  AND created_at < NOW() - INTERVAL '1 hour'
GROUP BY provider, dataset, status;
```

## Related Skills

- See the **queue-postgresql** skill for job monitoring patterns
- See the **cache-postgres** skill for cache performance analysis
- See the **geospatial** skill for PostGIS data quality checks