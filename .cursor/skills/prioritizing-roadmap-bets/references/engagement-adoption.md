# Engagement & Adoption

## When to use

Scoring initiatives that drive recurring API usage, widget embed proliferation, or subscription tier upgrades. Apply this when betting on features that increase requests to `/api/v1/*`, expand GHL widget deployments, or move subscribers from Pro to Ultra tiers.

## Project-relevant patterns

**Listings cache hit ratio**: The `LISTINGS_CACHE_TTL` (15 min) and `RefreshDomainListingsCacheJob` define engagement velocity. High-impact bets often improve cache freshness or expand cached endpoints — but effort scales with the `listings_cache` table migration complexity and Bridge API rate limits.

**Widget surface expansion**: New widget types (search, lead-form, showcase in `routes/ghl-widget.php`) drive adoption but require `GhlLeadMapping` updates and lead sync job scaling. Score effort by the `SyncLeadToGhlJob` queue throughput and GHL API timeout handling in `GhlApiClient`.

**Teaser gating thresholds**: The `SubscriptionCatalog` defines teaser limits by tier. Bets that adjust gating (e.g., raising teaser cap for Smart tier) have high revenue impact but risk MLS compliance if audit logging in `bridge_proxy_audit_logs` is bypassed or delayed.

## Warning

Avoid optimizing for raw request volume without qualifying intent. The GIS proxy (`/api/v1/gis`) generates high engagement metrics (parcel overlays) but carries no MLS data — betting on GIS expansion may distract from Bridge proxy monetization that actually drives subscription value.