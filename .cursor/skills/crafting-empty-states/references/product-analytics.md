# Product Analytics Reference

## Contents
- Available Signals
- Activation Metrics
- Query Patterns
- Anti-Patterns

## Available Signals

This project has no client-side analytics SDK, no event tracking, and no A/B testing framework. All analytics come from PostgreSQL queries against the existing schema.

### Data Sources

| Table | Signal | Relevant For |
|-------|--------|--------------|
| `users` | Account creation timestamp | Activation funnel |
| `domains` | Domain count, verification status | Setup completion |
| `personal_access_tokens` | Token count, creation date | API key adoption |
| `audit_logs` | Authenticated request count, endpoints used | Engagement depth |
| `invitations` | Invite creation and acceptance | Invite flow conversion |

## Activation Metrics

### Funnel: Invite → First API Call

```sql
-- Step 1: Invitations sent
SELECT COUNT(*) FROM invitations;

-- Step 2: Invitations accepted (user exists)
SELECT COUNT(*) FROM users WHERE created_at > (SELECT MIN(created_at) FROM invitations);

-- Step 3: Domain added
SELECT COUNT(DISTINCT user_id) FROM domains;

-- Step 4: Domain verified
SELECT COUNT(DISTINCT user_id) FROM domains WHERE verification_status = 'verified';

-- Step 5: Token used (has audit log entries)
SELECT COUNT(DISTINCT tokenable_id) FROM personal_access_tokens t
JOIN audit_logs a ON a.token_id = t.id;
```

### Time-to-First-Domain

```sql
SELECT u.id,
       u.created_at AS user_created,
       MIN(d.created_at) AS first_domain,
       EXTRACT(EPOCH FROM (MIN(d.created_at) - u.created_at)) / 60 AS minutes_to_domain
FROM users u
LEFT JOIN domains d ON d.user_id = u.id
GROUP BY u.id, u.created_at
ORDER BY u.created_at DESC
LIMIT 20;
```

### Verification Success Rate

```sql
SELECT
    COUNT(*) AS total_domains,
    COUNT(*) FILTER (WHERE verification_status = 'verified') AS verified,
    COUNT(*) FILTER (WHERE verification_status = 'pending') AS pending,
    ROUND(100.0 * COUNT(*) FILTER (WHERE verification_status = 'verified') / COUNT(*), 1) AS pct_verified
FROM domains;
```

## Query Patterns for Dashboard Metrics

### Active Users (30-day)

```sql
SELECT COUNT(DISTINCT tokenable_id) AS active_users
FROM personal_access_tokens t
JOIN audit_logs a ON a.token_id = t.id
WHERE a.created_at > NOW() - INTERVAL '30 days';
```

### Endpoint Popularity

```sql
SELECT split_part(a.path, '?', 1) AS endpoint, COUNT(*) AS hits
FROM audit_logs a
WHERE a.created_at > NOW() - INTERVAL '7 days'
GROUP BY 1
ORDER BY 2 DESC
LIMIT 20;
```

### Domain Dataset Distribution

```sql
SELECT mls_dataset, COUNT(*) FROM domains GROUP BY mls_dataset;
```

## Anti-Patterns

- **NEVER** add Google Analytics, PostHog, or any third-party analytics script to the dashboard. The dashboard is invite-only and serves a developer audience. Use SQL queries against existing tables.
- **AVOID** adding event tracking middleware that writes per-request analytics rows. The `audit_logs` table already captures authenticated requests — use it.
- **NEVER** store analytics in application memory (maps, counters). This is a multi-DC deployment with multiple API instances. In-memory data is incorrect.
- **AVOID** querying `audit_logs` in the dashboard handler for real-time metrics display. The table can grow large. Aggregate into a materialized view or summary table if dashboard metrics are needed.

## Related Skills

- See the **postgres** skill for query optimization and materialized views
- See the **auth-api-token** skill for audit log schema
- See the **queue-postgresql** skill for background job metrics