# Growth Engineering

## When to use
Use when implementing viral loops, usage-based pricing, or self-service expansion features in the IDX platform.

## Patterns

### Widget Viral Distribution
JS widgets spread through embed codes with attribution tracking:

```html
<!-- loader.js pattern - distributed via GHL Marketplace -->
<script src="https://idx-api.quantyralabs.cc/widget/loader.js"
        data-api-key="qh_..."
        data-location-id="..."
        data-widget="search">
</script>
```

Each widget load validates Origin against registered URLs, creating a viral loop where each install drives new domain registrations.

### Usage-Based Expansion
Metered billing for API overages encourages organic growth:

```php
// SubscriptionCatalog - metered overage on Ultra/Mega
'ultra' => [
    'monthly_price_id' => env('STRIPE_PRICE_IDX_ULTRA_MONTHLY'),
    'features' => ['2M_API_CALLS', 'UNLIMITED_DOMAINS'],
    'metered' => [
        'price_id' => env('STRIPE_PRICE_IDX_API_OVERAGE_METERED'),
        'unit' => 'per_1000_calls',
    ],
],
```

### Self-Service Token Management
Dashboard API token rotation reduces support burden:

```php
// DashboardApiTokenController - idx:full token self-management
public function store(Request $request)
{
    $token = $request->user()->createToken('geo-web-custom', ['idx:full']);
    
    return response()->json([
        'token' => $token->plainTextToken, // Shown once
        'abilities' => ['idx:full'],
        'last_used_at' => null,
    ]);
}
```

## Pitfall
Avoid auto-scaling subscription tiers without explicit user consent. A surprise $449 Mega bill creates churn. Always gate upgrades behind explicit checkout confirmation.