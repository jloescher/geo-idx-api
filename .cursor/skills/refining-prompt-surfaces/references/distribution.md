# Distribution

## When to use
Deploying widget embeds, sharing GHL Marketplace app, or routing traffic between idx.quantyralabs.cc and idx-api.quantyralabs.cc surfaces.

## Patterns

**Widget loader distribution**
The `/widget/loader.js` route returns the embed script. Users paste this into GHL websites or external sites. The loader must:
- Load from `IDX_API_PUBLIC_URL` (not `APP_URL` directly)
- Include `data-api-key` and `data-location-id` attributes
- Use `defer` attribute to avoid blocking host page render

**GHL Marketplace app URLs**
Three canonical URLs are required in the GHL developer dashboard:
- Installation: `https://idx-api.quantyralabs.cc/leadconnector/install`
- OAuth callback: `https://idx-api.quantyralabs.cc/oauth/leadconnector/callback`
- Webhooks: `https://idx-api.quantyralabs.cc/webhooks/leadconnector`

Use `route('leadconnector.install')` etc. in documentation—don't hardcode hostnames in case of staging environments.

**Image proxy routing**
`idx-images.quantyralabs.cc` serves `/images/*` via Nginx reverse-proxy to `idx-api:8000`. When distributing listing photos in emails or external shares, always use `IDX_IMAGES_PUBLIC_URL` URLs—never expose `api.bridgedataoutput.com` direct links. The `BridgeImageUrlRewriter` handles this in JSON responses.

## Warning
Widget embeds validate Origin headers against registered URLs in `ghl_registered_urls`. If a user installs the widget on an unregistered domain, the API key will 403. Always prompt for URL registration during GHL onboarding—don't assume the GHL location domain matches the widget host.