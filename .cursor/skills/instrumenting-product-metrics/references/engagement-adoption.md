# Engagement & Adoption

Use when measuring how actively locations use MLS data, widgets, and API features after onboarding.

## Patterns

**MLS Request Velocity by Location**

```sql
-- Weekly engagement tiering
SELECT 
    ghl_location_id,
    COUNT(*) as total_requests,
    COUNT(DISTINCT date_trunc('day', logged_at)) as active_days,
    AVG(listing_count) as avg_listings_per_request,
    CASE 
        WHEN COUNT(*) > 100 AND COUNT(DISTINCT date_trunc('day', logged_at)) >= 5 THEN 'power'
        WHEN COUNT(*) > 10 THEN 'active'
        ELSE 'low'
    END as engagement_tier
FROM bridge_proxy_audit_logs
WHERE logged_at > NOW() - INTERVAL '7 days'
GROUP BY ghl_location_id;
```

**Widget Feature Adoption Matrix**

```php
// Track which widget types are configured and used
$configs = \App\Ghl\Widgets\Models\GhlWidgetConfig::whereIn('ghl_location_id', $locationIds)->get();

$adoption = [
    'search_widget' => $configs->where('search_enabled', true)->count(),
    'lead_form' => $configs->where('lead_form_enabled', true)->count(),
    'showcase' => $configs->where('showcase_enabled', true)->count(),
    'custom_theme' => $configs->whereNotNull('widget_theme')->count(),
];
```

**Lead Capture Conversion Rate**

```sql
-- Leads per MLS request (engagement quality indicator)
SELECT 
    a.ghl_location_id,
    COUNT(DISTINCT a.id) as mls_requests,
    COUNT(DISTINCT l.id) as leads_captured,
    ROUND(COUNT(DISTINCT l.id)::numeric / NULLIF(COUNT(DISTINCT a.id), 0), 2) as conversion_rate
FROM ghl_installed_locations a
LEFT JOIN quantyra_leads l ON a.ghl_location_id = l.ghl_location_id
WHERE a.created_at > NOW() - INTERVAL '30 days'
GROUP BY a.ghl_location_id;
```

## Warning

High MLS request counts may indicate cache misses rather than genuine engagement. Cross-reference `bridge_proxy_audit_logs.request_type` with `listings_cache.last_updated` to distinguish organic usage from cache refresh jobs.