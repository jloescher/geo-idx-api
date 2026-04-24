# Product Analytics

When to use: Tracking onboarding funnel conversion, feature adoption, and revenue-critical events for the GHL Marketplace integration.

## Project-Relevant Patterns

### Audit Log as Event Stream
`GhlAuditService` records structured events to `ghl_audit_logs` with consistent schema: `event_type`, `location_id`, `metadata_json`, `created_at`. Use this as the source of truth for funnel analysis: `OAUTH_INITIATED` → `OAUTH_COMPLETED` → `URL_REGISTERED` → `WIDGET_LOADED` → `FIRST_LEAD_CAPTURED`.

### Bridge Proxy Request Telemetry
Every proxied request logs to `bridge_proxy_audit_logs` with `domain_slug`, `listing_count`, `response_time_ms`, and `cache_hit`. Aggregate this for capacity planning and to identify high-traffic domains that may need tier upgrades.

### Subscription Event Correlation
Cashier webhooks update subscription status, but the source of truth for revenue analytics is the union of: Stripe webhook events (payment succeeded), `ghl_installed_locations.subscription_status` (current state), and `SubscriptionCheckoutController` session metadata (UTM source, referrer). Correlate via `location_id` foreign key.

## Pitfall
Do not use `GhlOAuthToken::count()` as a growth metric. Tokens are created on OAuth callback, but many never complete URL registration. Always join against `ghl_registered_urls` or filter by `widget_embedded = true` for active user counts.
