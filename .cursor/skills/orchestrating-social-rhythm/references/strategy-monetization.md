# Strategy & Monetization

## When to use
Reference when designing pricing tiers, feature gates, or revenue optimization strategies for the IDX platform.

## Patterns

### Feature Matrix by Tier
Clear capability ladders drive upgrade decisions:

| Feature | Pro ($39) | Smart ($79) | Ultra ($179) | Mega ($449) |
|---------|-----------|-------------|--------------|-------------|
| Domains | 3 | 5 | Unlimited | Unlimited |
| GHL Integration | Basic | Full | Full | White-label |
| API Calls/mo | 10K | 100K | 2M | Unlimited |
| OTP Methods | Email | Email + Phone | Email + Phone | Custom |
| Teaser Limit | 3 listings | 3 listings | Full access | Full access |

### Teaser as Freemium Hook
The 3-listing teaser in Bridge proxy creates product-qualified leads:

```php
// BridgeProxyController - teaser applied post-cache
if (!$hasFullAccess && isset($data['value'])) {
    $originalCount = count($data['value']);
    $data['value'] = array_slice($data['value'], 0, 3);
    $data['meta']['teaser'] = true;
    $data['meta']['total_available'] = $originalCount;
    $data['meta']['upgrade_url'] = config('idx.platform_url') . '/checkout';
}
```

### Annual Prepay Incentives
20% discount for annual billing reduces churn and improves cash flow:

```php
// SubscriptionCheckoutController - annual pricing
$annualPriceId = match($plan) {
    'pro' => env('STRIPE_PRICE_IDX_PRO_YEARLY'),
    'smart' => env('STRIPE_PRICE_IDX_SMART_YEARLY'),
    'ultra' => env('STRIPE_PRICE_IDX_ULTRA_YEARLY'),
    'mega' => env('STRIPE_PRICE_IDX_MEGA_YEARLY'),
};
```

## Pitfall
Never remove teaser gating from lower tiers without compensating pricing changes. The teaser is a core conversion mechanism—removing it without price increases cannibalizes upgrade revenue.