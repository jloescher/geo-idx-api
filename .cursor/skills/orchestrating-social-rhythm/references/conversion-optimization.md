# Conversion Optimization

## When to use
Apply these patterns when designing lead capture flows, gated content access, or trial-to-paid conversion paths in the IDX API ecosystem.

## Patterns

### Teaser Gating with Value Promise
The Bridge proxy uses a 3-item teaser cap for non-full-access requests. The response includes just enough data to demonstrate value while compelling upgrade:

```php
// BridgeTeaser.php - Revenue impact: teaser cap drives subscription upgrades
if (!$hasFullAccess && is_array($data)) {
    return array_slice($data, 0, config('bridge.teaser_limit', 3));
}
```

Combine with explicit upgrade CTAs in the dashboard showing "Unlock 50+ listings in this area" to maximize conversion intent.

### Progressive Profiling via Widget Leads
Widget lead forms capture minimal fields initially (email + lead_type), then enrich via GHL contact sync:

```php
// QuantyraLead → GHL contact pipeline
$lead = QuantyraLead::create([
    'ghl_location_id' => $locationId,
    'lead_type' => 'showing_request', // Maps to GhlLeadMapping behavior
    'payload' => ['email' => $email, 'first_name' => $firstName],
]);
SyncLeadToGhlJob::dispatch($lead)->onQueue('sync');
```

### Trial Timeboxing with Usage Hooks
14-day trials include soft warnings at 7 days and hard expiration. Surface usage metrics (API calls, widget loads) to create urgency:

```php
// SubscriptionCheckoutController - trial with metered overage
$checkout = $user->newSubscription('default', $priceId)
    ->trialDays(14)
    ->allowPromotionCodes()
    ->checkout([
        'success_url' => route('dashboard') . '?subscription=active',
        'cancel_url' => route('dashboard'),
    ]);
```

## Pitfall
Never gate critical MLS compliance features (audit logging, Origin validation) behind subscription tiers—regulators expect these at all tiers. Only gate data volume and advanced features.