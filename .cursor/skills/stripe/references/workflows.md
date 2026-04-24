# Stripe Development Workflows

**When to use:** Follow these workflows when setting up billing, testing webhooks locally, or provisioning Stripe products for new environments.

## Local Webhook Development

1. Start Docker dev services: `./scripts/docker-dev.sh up-watch`
2. In another terminal, start Stripe forwarding: `./scripts/stripe-dev.sh listen`
3. Copy the printed webhook signing secret to `.env` as `STRIPE_WEBHOOK_SECRET`
4. Trigger a test event: `./scripts/stripe-dev.sh trigger-checkout-completed`

## Provision Stripe Catalog

Run once per environment to create products and prices:

```bash
php artisan billing:provision-stripe-catalog
```

This creates:
- Pro ($39/mo, $374/yr), Smart ($79/mo, $758/yr)
- Ultra ($179/mo, $1,718/yr), Mega ($449/mo, $4,310/yr)
- Metered overage price for API calls

## Subscription Status Sync to GHL

When a Stripe event updates a subscription:

```php
// In your webhook handler or job
(new \App\Ghl\Sync\Services\SubscriptionSyncService())
    ->syncStatus($stripeSubscription, $ghlLocationId);
```

This writes to `ghl_installed_locations.subscription_status` using tags from `config('ghl.subscription_tags')`.

## ⚠️ Pitfall

**Price ID drift:** After provisioning, update `config/billing.php` with the actual Stripe Price IDs. The `STRIPE_PRICE_IDX_*` env vars must match your Stripe account exactly—mismatched IDs cause checkout failures with "price not found" errors.