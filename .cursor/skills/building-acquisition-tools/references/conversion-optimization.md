# Conversion Optimization

## When to use
Optimize the path from free tool usage to paid subscription by strategically gating MLS data, implementing progressive disclosure, and removing friction at upgrade boundaries.

## Patterns

**Teaser-to-Full Upgrade Flow**
Non-`idx:full` requests receive capped listings (3 items via `BridgeTeaser`) while cached full data stays canonical. Apply teaser logic after cache retrieval so upgrades reveal full data instantly without cache invalidation. The `gate_after_views` column in `ghl_widget_configs` controls when to prompt for registration.

**Geography Dwell Strategy**
Use the GIS proxy (`/api/v1/gis`) to display public Florida parcel overlays without MLS compliance burden. Public cadastral data keeps users engaged on the map (increasing time-on-site) before OTP or registration gates appear. Chain GIS calls with `/api/v1/listings` only after conversion.

**Widget Gating Configuration**
Configure `ghl_widget_configs.require_otp` per location. The three-phase middleware chain (key validate → origin validate → CORS) allows embedding anywhere while enforcing gates at the data layer. Lead forms bypass the OTP gate but require valid `api_key` and matching Origin header.

## Warning
Never cache teaser-truncated data to PostgreSQL—the `listings_cache` table stores full Bridge responses. Teaser logic runs after decompression so upgrading from `idx:access` to `idx:full` requires no cache purge.