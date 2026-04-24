# Stripe Cashier Patterns

**When to use:** Reference these patterns when implementing billing flows, handling Stripe webhooks, or managing subscription state in the Quantyra IDX API.

## Subscription Checkout Session

Create a Checkout session with trial and tiered pricing:

```php
$session = auth()->user()->billing->checkout([
    $priceId => 1,
], [
    'success_url' => config('idx-urls.platform') . '/dashboard?checkout=success',
    'cancel_url' => config('idx-urls.platform') . '/dashboard?checkout=cancel',
    'subscription_data' => ['trial_days' => 14],
]);
```

## Webhook Event Handling

Handle Cashier webhooks in your controller or job:

```php
$event = \Stripe\Webhook::constructEvent(
    $payload, $sigHeader, config('services.stripe.webhook_secret')
);

switch ($event->type) {
    case 'customer.subscription.created':
        $this->syncToGhl($event->data->object);
        break;
    case 'invoice.payment_succeeded':
        // Update usage counters
        break;
}
```

## Price ID Catalog Reference

Access tiered plans via `SubscriptionCatalog`:

```php
$catalog = new \App\Billing\SubscriptionCatalog();
$plans = $catalog->getPlans(); // Pro, Smart, Ultra, Mega with monthly/yearly prices

// Check feature gates
$plan = $catalog->getPlan('ultra');
if ($plan->hasFeature('custom_branding')) { ... }
```

## ⚠️ Pitfall

**Webhook secret mismatch:** `STRIPE_WEBHOOK_SECRET` from Dashboard (`whsec_...`) differs from CLI `stripe listen --print-secret`. Never commit the CLI secret to `.env` for production—Dashboard endpoints use a different signing key and events will fail signature verification.