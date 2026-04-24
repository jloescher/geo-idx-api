# Strategy & Monetization

When to use this reference: When evaluating pricing model changes, tier restructuring, or new revenue streams beyond core subscriptions—ensuring alignment with the existing Pro/Smart/Ultra/Mega ladder and MLS data costs.

## Patterns

**Teaser as Freemium Gateway**
The existing 3-listing cap in `BridgeTeaser` serves as a cost-controlled free tier. Domain-authenticated requests cost minimal infrastructure (cache hit on `listings_cache`) while demonstrating value. Monetize lift via `idx:full` token upgrades, not by removing the teaser—it's a permanent acquisition channel.

**Metered API Differentiation**
Ultra ($179) includes 2M calls; Mega ($449) is unlimited. Position this as "Scale" vs "Enterprise" in `SubscriptionCatalog` descriptions. Overages should auto-bill via Stripe metered billing—configure `STRIPE_PRICE_IDX_API_OVERAGE_METERED` for seamless revenue capture without manual invoices.

**GHL Marketplace Network Effects**
Each GHL location install creates lock-in via `ghl_registered_urls` and widget embeds. The OAuth token exchange (`LocationTokenService`) binds Quantyra to the location's CRM—high switching costs justify premium Mega pricing for agencies with many locations.

## Warning

MLS data costs scale with actual API calls to Bridge Data Output, not cached responses. The `LISTINGS_CACHE_TTL` (15 minutes) and `GIS_ORIGIN_MAX_DAYS_*` settings control cost but must be balanced against user experience. Aggressive monetization that drives high API usage without corresponding subscription revenue erodes margin—monitor Bridge egress costs per tier.