# Strategy and Monetization for Partner Ecosystems

## When to use
When designing pricing tiers, usage limits, and value-capture mechanisms that align partner success with revenue growth.

## Project-relevant patterns

**Teaser as a revenue lever**
Cap non-full responses at 3 items (Bridge listings) or 40 features (GIS parcels). This creates natural upgrade pressure without hard paywalls—users see the data exists but need `idx:full` ability or a subscription tier for complete access.

**Tiered subscription alignment**
Map feature limits to subscription status stored in `ghl_installed_locations.subscription_status`: Pro (3 domains), Smart (5 domains + OTP), Ultra (unlimited + 2M calls), Mega (SLA). Update status via Stripe webhook → `SubscriptionSyncService` → GHL tag sync.

**Lead sync as value-add**
Inbound widget leads create `quantyra_leads` rows that dispatch `SyncLeadToGhlJob` to push contacts/opportunities into the partner's GHL CRM. This makes your integration stickier—their CRM fills with leads you generated, increasing switching costs.

## Warning
Avoid metering on upstream API calls you don't control. The Bridge proxy caches aggressively; billing on raw Bridge requests would undercharge cached domains and overcharge filtered queries. Meter on your own API surface (`mls_request_count`) or use Stripe's metered billing for actual usage events you log.
=====