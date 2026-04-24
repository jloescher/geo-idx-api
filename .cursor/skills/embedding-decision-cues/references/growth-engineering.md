# Growth Engineering

When to use this reference: When building viral loops, referral mechanisms, or self-serve expansion features that accelerate user acquisition and revenue growth without sales intervention.

## Patterns

**Widget Virality Loop**
The `/widget/loader.js` embed creates implicit distribution—each installation on a GHL location's website exposes Quantyra to that agent's visitors. Ensure widget config includes subtle branding and upgrade CTAs that route to `IDX_PLATFORM_URL` with `?ref=widget` attribution. Track widget-generated trials separately from organic in `bridge_proxy_audit_logs.request_type`.

**Land-and-Expand Billing**
Design `SubscriptionCheckoutController` to default to annual billing (20% savings) while making monthly frictionless to select. The expansion path Pro→Smart→Ultra should require no re-authentication—use existing `ghl_oauth_tokens` to upgrade Stripe subscriptions in-place via `SubscriptionSyncService`.

**API Overage Metering**
Ultra tier includes 2M API calls/month with metered overage. Implement soft warnings at 80% usage surfaced in dashboard and widget contexts—"You've used 1.6M of 2M calls this month." This creates natural upgrade pressure to Mega before hard limits trigger.

## Warning

Growth features must respect the `GHL_WEBHOOK_REQUIRE_SIGNATURE` enforcement. Viral loops that trigger webhook processing without signature validation create compliance gaps. Never bypass `VerifyGhlWebhookSignature` middleware for convenience in high-volume growth experiments.