# Distribution

## When to use

Expanding reach through the GHL Marketplace, widget embed adoption, and API access patterns. Reference when onboarding new locations or troubleshooting domain registration issues.

## Patterns

**GHL Marketplace Distribution** (`routes/ghl-web.php`)
- Canonical URLs: `https://idx-api.quantyralabs.cc/leadconnector/install`
- OAuth redirect: `/oauth/leadconnector/callback` exchanges code for token
- Scopes: Space-separated in `GHL_SCOPES` env var; must match GHL app dashboard
- Webhook delivery: `POST /webhooks/leadconnector` with optional signature verification

**Widget Embed Distribution** (`routes/ghl-widget.php`)
- Loader script: `/widget/loader.js` with `data-api-key`, `data-location-id`, `data-widget`
- Origin validation: Middleware checks `Origin`/`Referer` against `ghl_registered_urls.primary_url`/`additional_urls`
- CORS: Preflight `OPTIONS /widget/api/leads` responds with allowed origins
- Embed surfaces: `search`, `lead-form`, `showcase` with consistent API key auth

**API Access Patterns** (`routes/api.php`)
- Bridge proxy: `/api/v1/*` with `domain.token` middleware (domain slug or Bearer token)
- GHL proxy: `/api/leadconnector/*` with `AuthenticateGhlLocation` middleware
- GIS proxy: `/api/v1/gis` with identical auth to Bridge routes

## Warning

Agency tokens (Company user type) without a stored `ghl_location_id` require explicit `?location_id=` parameter on all API calls. The `LocationTokenService` can exchange agency tokens for location-specific tokens via `/oauth/locationToken`.