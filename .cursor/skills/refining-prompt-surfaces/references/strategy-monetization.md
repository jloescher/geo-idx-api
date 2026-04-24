# Strategy & Monetization

## When to use
Adjusting pricing tiers, adding usage-based billing, or aligning GHL Marketplace positioning with IDX platform subscriptions.

## Patterns

**Metered API overage billing**
The Ultra and Mega tiers include 2M API calls/month. Beyond that, Stripe metered billing kicks in via `STRIPE_PRICE_IDX_API_OVERAGE_METERED`. Expose usage counters in the subscriber dashboard using `BridgeProxyAuditLog` aggregations. Warn at 80%: "You've used 1.6M calls this month. Overage: $0.005 per 1K calls."

**GHL location vs independent billing**
GHL Marketplace users may also subscribe directly via `idx.quantyralabs.cc`. The `ghl_installed_locations.subscription_status` field tracks GHL-side state; `users` table with Cashier tracks direct subscriptions. Avoid double-billing: check both sources before showing upgrade CTAs. Use `SubscriptionSyncService` to reconcile status.

**Teaser as monetization lever**
Domain-authenticated requests (no Sanctum token) always get teaser responses: 3 listings max, simplified GIS polygons. This is intentional—don't increase teaser limits. Instead, optimize the upgrade prompt timing: after 2 teaser views in one session, show the "Get full access" modal with the `SubscriptionCatalog` plan comparison.

## Warning
Stripe webhook signing secrets differ between Dashboard endpoints and CLI forwarding (`stripe listen`). The `STRIPE_WEBHOOK_SECRET` env var must match the delivery method. Mismatched secrets cause webhook verification failures, which block subscription status updates and can leave users without access after payment.