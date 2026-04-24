# Product Analytics

Use when defining metrics, dashboards, and automated reports for stakeholder visibility into platform health and growth.

## Patterns

**Core Metrics View**

```sql
-- Weekly business health snapshot
SELECT 
    date_trunc('week', logged_at) as week,
    COUNT(DISTINCT domain_slug) as active_domains,
    COUNT(DISTINCT ghl_location_id) as active_locations,
    SUM(listing_count) as total_listings_served,
    AVG(CASE WHEN listing_count <= 3 THEN 1 ELSE 0 END) as teaser_rate,
    COUNT(DISTINCT CASE WHEN request_type = 'image' THEN ip_address END) as unique_image_visitors
FROM bridge_proxy_audit_logs
GROUP BY week
ORDER BY week DESC;
```

**Cohort Retention by Install Week**

```sql
-- Location retention: % still active N weeks after install
WITH cohorts AS (
    SELECT 
        ghl_location_id,
        date_trunc('week', created_at) as cohort_week
    FROM ghl_installed_locations
    WHERE status = 'active'
)
SELECT 
    c.cohort_week,
    COUNT(DISTINCT c.ghl_location_id) as cohort_size,
    COUNT(DISTINCT CASE WHEN a.logged_at > c.cohort_week + INTERVAL '1 week' THEN c.ghl_location_id END) as week_1_active,
    COUNT(DISTINCT CASE WHEN a.logged_at > c.cohort_week + INTERVAL '4 weeks' THEN c.ghl_location_id END) as week_4_active
FROM cohorts c
LEFT JOIN bridge_proxy_audit_logs a ON a.domain_slug IN (
    SELECT domain_slug FROM ghl_registered_urls WHERE ghl_oauth_token_id = (
        SELECT id FROM ghl_oauth_tokens WHERE ghl_location_id = c.ghl_location_id
    )
)
GROUP BY c.cohort_week;
```

**Revenue-At-Risk Alerts**

```sql
-- Locations with usage drop before renewal
SELECT 
    l.ghl_location_id,
    l.subscription_status,
    l.mls_request_count as total_requests,
    COUNT(a.id) as requests_last_7_days,
    CASE 
        WHEN l.mls_request_count > 100 AND COUNT(a.id) = 0 THEN 'high_risk_churn'
        WHEN COUNT(a.id) < 5 THEN 'low_engagement'
        ELSE 'healthy'
    END as health_status
FROM ghl_installed_locations l
LEFT JOIN bridge_proxy_audit_logs a ON l.ghl_location_id = a.user_id::text
    AND a.logged_at > NOW() - INTERVAL '7 days'
WHERE l.subscription_status IN ('active', 'trial')
GROUP BY l.ghl_location_id, l.subscription_status, l.mls_request_count;
```

## Warning

Teaser vs full access distorts engagement metrics—always segment or normalize by `idx:full` token capability. Domain-authenticated requests with teaser caps will show artificially low listing counts and may appear as low engagement when they are actually active paid users.