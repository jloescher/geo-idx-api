---
name: stripe
description: Manages Stripe billing, Laravel Cashier subscriptions, and webhook handling for the Quantyra IDX API
allowed-tools: [Read, Edit, Write, Glob, Grep, Bash]
---

# Stripe Skill

This skill manages Stripe subscription billing via Laravel Cashier, including checkout sessions, webhook handling, and price catalog management for tiered plans (Pro, Smart, Ultra, Mega).

## Quick Start

```bash
# Start Stripe webhook forwarding locally
./scripts/stripe-dev.sh listen

# Fire a test checkout completion event
./scripts/stripe-dev.sh trigger-checkout-completed

# Provision Stripe products and prices for all plans
php artisan billing:provision-stripe-catalog
```

## Key Concepts

- **Webhook endpoint**: `{APP_URL}/stripe/webhook` (default Cashier path)
- **Signing secrets**: Use Dashboard secret for production/staging; use `stripe listen --print-secret` for local CLI forwarding
- **Subscription tiers**: Pro ($39/mo), Smart ($79/mo), Ultra ($179/mo), Mega ($449/mo) with metered API overage
- **Token management**: Internal geo-web token via `php artisan idx-api:issue-geo-web-token`

## Common Patterns

- **Environment setup**: Set `STRIPE_KEY`, `STRIPE_SECRET`, `STRIPE_WEBHOOK_SECRET` in `.env`
- **Price IDs**: Store in `config/billing.php` with keys like `STRIPE_PRICE_IDX_PRO_MONTHLY`
- **Checkout flow**: `SubscriptionCheckoutController` creates Stripe Checkout sessions with trial days
- **Webhook handling**: Cashier routes webhook to controllers; verify `Stripe-Signature` header
- **GHL sync**: `SubscriptionSyncService` updates `ghl_installed_locations.subscription_status` on Stripe events