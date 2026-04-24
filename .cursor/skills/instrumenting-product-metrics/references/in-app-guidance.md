# In-App Guidance

Use when implementing contextual help, progressive disclosure, and feature announcements within the IDX dashboard and GHL widget flows.

## Patterns

**Contextual Onboarding Checklist**

```php
// Dashboard controller: determine next setup step
public function getOnboardingStep($location)
{
    if (!$location->registeredUrls()->exists()) {
        return ['step' => 'register_urls', 'priority' => 'high'];
    }
    if ($location->subscription_status === 'none') {
        return ['step' => 'start_trial', 'priority' => 'high'];
    }
    if ($location->mls_request_count < 5) {
        return ['step' => 'preview_listings', 'priority' => 'medium'];
    }
    if ($location->lead_count === 0) {
        return ['step' => 'embed_widget', 'priority' => 'medium'];
    }
    return ['step' => 'complete', 'priority' => 'low'];
}
```

**Widget Configuration Hints**

```php
// Inline guidance based on current config
$hints = [];
if (empty($config->widget_theme)) {
    $hints[] = ['type' => 'tip', 'message' => 'Customize your widget theme to match your brand colors.'];
}
if ($config->gate_after_views === null) {
    $hints[] = ['type' => 'recommendation', 'message' => 'Enable lead gating after 3 listing views to increase conversion.'];
}
if ($location->subscription_status === 'trial' && $daysRemaining <= 3) {
    $hints[] = ['type' => 'urgent', 'message' => 'Trial expires soon—upgrade to keep full access.'];
}
```

**Progressive Feature Disclosure**

```sql
-- Show advanced features only after basic adoption
SELECT 
    ghl_location_id,
    CASE 
        WHEN mls_request_count > 20 AND lead_count > 0 THEN 'advanced_analytics'
        WHEN mls_request_count > 5 THEN 'custom_filters'
        ELSE 'basic_search'
    END as appropriate_feature_tier
FROM ghl_installed_locations;
```

## Warning

Avoid showing guidance modals during active widget lead capture flows—detect `widget/api/leads` endpoint usage in the last 5 minutes before displaying non-urgent messages to prevent conversion disruption.