# Distribution

## When to use
Deploying widgets across multiple domains, managing embed codes, or troubleshooting CORS failures. Use when expanding to new GHL locations or external websites.

## Patterns

### Origin Coverage Checklist
Before launch, verify `ghl_registered_urls.primary_url` and `additional_urls` include all variants: www/non-www, http/https, and any subdirectory paths where the widget loads. The 3-phase middleware rejects mismatched Origins with 403.

```bash
# Test origin validation
curl -X OPTIONS "/widget/api/leads?api_key=qh_..." \
  -H "Origin: https://client-domain.com"
```

### Widget Loader Embed Pattern
Distribute the loader.js snippet with `data-api-key`, `data-location-id`, and `data-widget` attributes. The loader fetches config from `/widget/config/{apiKey}` and injects the appropriate surface (search, lead-form, showcase).

### Cross-Domain Lead Routing
Use the MLS-scoped endpoint `/api/v1/mls/{mlsCode}/gis` alongside listings to align parcel data with the correct MLS region, ensuring widget context matches the registered domain's market.

## Warning
Subdomain mismatches block submissions silently. If the widget works on `www.client.com` but fails on `client.com`, check that both variants are in `additional_urls` JSON array—CORS preflight fails closed for security.