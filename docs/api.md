# idx-api HTTP API overview

This document summarizes **versioned JSON APIs** exposed by the Laravel idx-api service (Octane + FrankenPHP ready).

## Authenticated IDX / Bridge proxy (`/api/v1`)

All routes in `routes/api.php` under the `v1` prefix use the `domain.token` middleware (domain header / referer host **or** Sanctum bearer with `idx:access` / `idx:full`).

Core resources: listings, agents, offices, RESO Property, members, public parcels bridge, **structured search**, etc. See [`docs/idx-api-bridge-proxy.md`](idx-api-bridge-proxy.md) and [`docs/bridge-api-documentation.md`](bridge-api-documentation.md).

### New: Listing pricing enrichment (`GET /api/v1/listings`, `GET /api/v1/listings/{listingId}`)

Listings responses now include:
- a top-level `pricing` object (`status`, `as_of`, and quote matrix), and
- `pricing_converted` on each listing item with fiat + digital asset conversions derived from `ListPrice`.

Refresh pipeline details:
- CoinGecko quotes are refreshed every 10 minutes by scheduled dispatch of `RefreshCryptoPricingJob`.
- The job updates both PostgreSQL (`crypto_price_snapshots`) and Laravel cache (`coingecko.pricing.matrix`).
- Read path does not call CoinGecko; listing enrichment uses cache/DB only.

### New: Structured Search endpoint (`POST /api/v1/search`)

The search endpoint accepts JSON payloads with filter criteria, translates them to Bridge RESO OData queries, and returns paginated results with computed statistics.

**Features:**
- Multi-dataset support (validated against domain's `allowed_mls_datasets`)
- Structured filters: location (cities, counties, states), price range, beds/baths, property types, features (pool, waterfront), etc.
- OData cursor pagination via `@odata.nextLink`
- 15-minute result caching (same cache mechanism as listings)
- Teaser gating for non-full-access plans
- Image URL rewriting to `idx-images` host

See [IDX-API Bridge proxy — Search endpoint](idx-api-bridge-proxy.md#search-endpoint-post-apiv1search) for full request/response format and filter mapping.

### How to obtain a Bearer token for `/api/v1`

| Source | Abilities | Works with `domain.token`? |
|--------|-----------|----------------------------|
| **Subscriber dashboard** (GeoIDX dashboard → API Keys) | **`idx:access`** on **Ultra**, **`idx:full`** on **Mega** | Yes — intended for server or tooling calls to `/api/v1/*`. |
| **`POST /api/auth/token`** (email + password + `device_name`) | `idx:read`, `idx:search` | **No** for Bridge proxy routes — those abilities are not accepted by `DomainOrTokenAuth`. Use dashboard keys or an internal `idx:full` token (e.g. `php artisan idx-api:issue-geo-web-token` / `GeoWebInternalTokenSeeder`). |

Ultra dashboard keys behave like domain traffic (**teaser** list caps). Mega keys receive **full** payloads. Details: [IDX-API Bridge proxy — Subscriber dashboard API keys](idx-api-bridge-proxy.md#subscriber-dashboard-api-keys-ultra-and-mega).

### Stripe test subscribers (all four plans)

To create **Pro, Smart, Ultra, and Mega** users with **active Stripe test subscriptions** (for dashboard login and checkout QA), use:

```bash
php artisan billing:seed-test-users
```

Requirements, emails, passwords, payment-method options, and **example successful Artisan output** (INFO lines, summary table, shared password): [`docs/stripe-laravel-cashier.md` — Seed billing test users](stripe-laravel-cashier.md#seed-billing-test-users-in-stripe-test-mode).

## GIS public overlay (`/api/v1/gis`)

**New:** Florida **public government** parcel GeoJSON proxy for Leaflet overlays—**not MLS data**.

- **Routes:** `GET /api/v1/gis`, `GET /api/v1/mls/{mlsCode}/gis`
- **Docs:** [`docs/gis-api.md`](gis-api.md) (OpenAPI-style parameters, examples for Pinellas / Tampa bbox, failover, caching, revenue notes).
- **Caching:** Short **Laravel edge** TTL plus long-lived **Postgres origin** rows (per-source max age in days), invalidated when weekly metadata probes bump `gis_source_states.generation` or when you run `gis:clear-cache`.

Use this alongside `/api/v1/listings` with the same viewport parameters for a single map flow.
