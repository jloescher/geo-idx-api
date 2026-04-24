# Strategy Monetization

## When to use
Design revenue models that align access levels with subscription tiers while managing API costs through caching, metering, and feature gating.

## Patterns

**Metered API Overage**
Ultra and Mega tiers include 2M API calls/month base. Configure `STRIPE_PRICE_IDX_API_OVERAGE_METERED` for per-call billing beyond tier limits. The `BridgeProxyAuditLogger` records per-domain usage for usage-based billing reconciliation.

**Progressive Feature Disclosure**
Map capabilities to `idx:access` vs `idx:full` Sanctum abilities. Domain-authenticated traffic (widget embeds) always receives `idx:access` with teaser limits. Only dashboard-generated tokens with `idx:full` bypass `BridgeTeaser` truncation, creating clear upgrade incentive from widget-only to full platform.

**Data Access Tiering**
GIS parcels (public government data) flow freely with only teaser coordinate precision reduction. Bridge MLS data requires domain registration or token auth. This two-class system provides value in free/public tiers while reserving premium MLS aggregation for paid subscriptions.

## Warning
The `ghl_installed_locations.lead_count` column tracks usage but is not automatically reset on subscription cancellation. Implement purge logic for reactivated accounts to avoid offering "unlimited leads" to accounts with inflated historical counts from previous billing periods.