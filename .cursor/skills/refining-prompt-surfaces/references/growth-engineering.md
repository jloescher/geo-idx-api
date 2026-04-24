# Growth Engineering

## When to use
Building viral loops, referral systems, or expansion features in the IDX platform—especially around GHL agency workflows and widget sharing.

## Patterns

**Agency-to-location token exchange**
GHL Company (agency) tokens can spawn Location tokens via `LocationTokenService`. When an agency installs, prompt: "Install GeoIDX for all your locations?" Use the `/oauth/locationToken` endpoint to exchange tokens, then auto-create `ghl_registered_urls` entries for each location's default domain. One agency install → many active widgets.

**Widget lead viral hints**
Widget configs include `gate_after_views` and `require_otp`. After lead submission, show a "Powered by GeoIDX" badge with a referral link to `IDX_PLATFORM_URL`. Track clicks via `referrer` query params. The widget surfaces at `/widget/showcase/{apiKey}` can include sample listings with "Get this on your site" CTAs for organic growth.

**GIS parcel teaser for dwell time**
The `/api/v1/gis` endpoint returns parcel overlays even for `idx:access` (teaser) tokens. Use this in marketing: "See property boundaries for every listing" increases map interaction time before OTP gates trigger. The `GisProxyService` handles 3-tier caching—GIS features don't count against Bridge API quotas.

## Warning
The `RefreshDomainListingsCacheJob` runs every 15 minutes per active domain. Growth features that trigger many domain registrations can spike queue depth. Monitor `LISTINGS_CACHE_TTL` (default 900s) and queue worker capacity—backlogged token refresh jobs cause stale listing data, hurting conversion.