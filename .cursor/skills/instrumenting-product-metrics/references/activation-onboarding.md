# Activation & Onboarding

Use when tracking user progression from initial OAuth through active widget installation to identify drop-off points in the GHL Marketplace funnel.

## Patterns

**Track OAuth to Active Install Funnel**

```sql
-- Funnel stages: authorize → callback → register-urls → active
SELECT 
    date_trunc('day', t.created_at) as day,
    t.user_type,
    COUNT(DISTINCT t.ghl_company_id) as oauth_started,
    COUNT(DISTINCT CASE WHEN t.status = 'active' THEN t.ghl_company_id END) as tokens_active,
    COUNT(DISTINCT r.id) as urls_registered,
    COUNT(DISTINCT CASE WHEN l.status = 'active' THEN l.ghl_location_id END) as locations_active
FROM ghl_oauth_tokens t
LEFT JOIN ghl_registered_urls r ON t.id = r.ghl_oauth_token_id
LEFT JOIN ghl_installed_locations l ON t.id = l.ghl_oauth_token_id
GROUP BY day, user_type;
```

**Measure Time-to-Value by Lead Type**

```php
// After first lead captured, mark activation complete
$firstLead = \App\Ghl\Sync\Models\QuantyraLead::where('ghl_location_id', $locationId)->first();
$location = \App\Ghl\OAuth\Models\GhlInstalledLocation::where('ghl_location_id', $locationId)->first();

if ($firstLead && $location) {
    $hoursToActivation = $location->created_at->diffInHours($firstLead->created_at);
    // Log or emit metric for cohort analysis
}
```

**Subscription Activation State Machine**

| State | Trigger | Tracking |
|-------|---------|----------|
| `none` | Initial install | Baseline |
| `trial` | Checkout session created | Stripe webhook |
| `active` | Payment confirmed | `invoice.payment_succeeded` |
| `past_due` | Payment failed | `invoice.payment_failed` |
| `cancelled` | Subscription ended | `customer.subscription.deleted` |

## Warning

Do not rely solely on `ghl_oauth_tokens.status` for activation—agency tokens (Company type) may show active while the sub-location install is incomplete. Always verify `ghl_installed_locations.status = 'active'` and at least one registered URL exists for funnel completion metrics.