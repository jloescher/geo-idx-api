# idx-api HTTP API overview

This document summarizes **versioned JSON APIs** exposed by the **Go** idx-api service (Fiber on port **8000**).

## OpenAPI 3.1 + Swagger UI

- **Spec document (OpenAPI 3.1):** `GET /openapi.json`
- **Swagger UI:** `GET /swagger`
- **Spec source file:** `docs/yaak-api-collection.json`

The Swagger page loads the canonical `openapi: "3.1.0"` document served by the API host.

## Authenticated IDX / Bridge proxy (`/api/v1`)

All routes under `/api/v1` use **domain + token** middleware (`internal/api/middleware/domain_token.go`): domain header / Referer host **or** Bearer PAT with `idx:access` or `idx:full`, plus domain binding for token calls. Authenticated `/api/v1` traffic receives **full** Bridge and GIS payloads (no subscription-tier teaser caps).

Core resources: listings, agents, offices, RESO Property, members, public parcels bridge, **structured search**, etc. See [`docs/idx-api-bridge-proxy.md`](idx-api-bridge-proxy.md) and [`docs/bridge-api-documentation.md`](bridge-api-documentation.md).

### New: Comparables, investor analysis & home value estimation (`POST /api/v1/comps/run`)

Comps endpoint supports standard sale-comps modes (`A`–`E`), investor modes (`rent_hold_cashflow`, `flip_vs_hold`, `appraiser_simulation`), BPO mode (`bpo`) with URAR-style market-derived adjustments, and **home value estimation** (`home_value`) for off-market properties using Google Maps geocoding.

- Uses the same `domain.token` middleware and dataset resolution path as other `/api/v1` routes
- Investor modes, BPO, and home value are available to all callers that pass `domain.token` (verified domain or PAT + domain slug)
- Home value mode accepts owner-provided property details (address, bedrooms, bathrooms, condition, renovations) and returns an estimated value with confidence scoring
- Renovation credits are dynamically derived from market data (not static)
- Includes garage/parking, view, subdivision, and MLS area matching extensions

See [`docs/comps-api.md`](comps-api.md) for request/response details and mode behavior.

### New: Listing pricing enrichment (`GET /api/v1/listings`, `GET /api/v1/listings/{listingId}`)

Listings responses now include:
- a top-level `pricing` object (`status`, `as_of`, and quote matrix), and
- `pricing_converted` on each listing item with fiat + digital asset conversions derived from `ListPrice`.

Refresh pipeline details:
- CoinGecko quotes are refreshed on a schedule via queue job `crypto.refresh_pricing` (`cmd/scheduler` → `cmd/worker`).
- Snapshots stored in PostgreSQL `crypto_price_snapshots`.
- Read path does not call CoinGecko; listing enrichment uses cache/DB only.

### Structured Search endpoint (`POST /api/v1/search`)

The search endpoint accepts JSON payloads with filter criteria and returns paginated results with computed statistics. **Routing is hybrid:** Active/Pending inventory is served from the local PostGIS **`listings`** mirror when possible; **Closed** (and some special filters) use live Bridge OData; requests that mix Active/Pending and Closed merge both sources before pagination.

**Features:**
- Multi-dataset support (validated against domain's `allowed_mls_datasets`)
- Structured filters: location (cities, counties, states), price range, beds/baths, property types, features (pool, waterfront), etc.
- **PostGIS mirror leg** (Active/Pending): `low_risk_floodzone` → `listings.flood_zone_code`; `min_monthly_fees` / `max_monthly_fees` → `listings.estimated_total_monthly_fees` (Beaches: association fees normalized to monthly at persist — see [Spark integration](spark/idx-api-integration.md#normalized-mirror-columns-persist--replication-updates))
- **Hybrid routing:** mirror for Active/Pending; Bridge for Closed-only or unsupported statuses; split merge when both appear in `status` / `statuses`
- OData cursor pagination via `@odata.nextLink` (Bridge leg; mirror leg uses SQL offset/limit)
- 15-minute result caching (same cache mechanism as listings)
- No plan-based teaser gating (internal deployment)
- Image URL rewriting to `idx-images` host

See [IDX-API Bridge proxy — Search endpoint](idx-api-bridge-proxy.md#search-endpoint-post-apiv1search) for request body, filter mapping, and the routing table.

### How to obtain a Bearer token for `/api/v1`

| Source | Abilities | Works with `domain.token`? |
|--------|-----------|----------------------------|
| **Auto-issued Production** (GeoIDX Setup — first domain verification) | **`idx:full`** | Yes — auto-created on first TXT verify; shown once in Setup panel. |
| **Staging** (Setup panel or `POST /dashboard/api-tokens/staging`) | **`idx:full`** | Yes — one-click Staging key for preview/staging sites (same domain slug, different Bearer). |
| **Custom named** (API Keys panel or `POST /dashboard/api-tokens`) | **`idx:full`** | Yes — for additional server or tooling calls to `/api/v1/*`. |
| **`POST /api/auth/token`** (email + password + `device_name`) | **`idx:full`** | Yes for `/api/v1/*` when paired with domain identification as above (same as other PATs). |

All tokens require **`X-Domain-Slug`** or **`?domain=`** with a verified domain on the same account. Dashboard PATs are minted with **`idx:full`**. Details: [IDX-API Bridge proxy — Dashboard API keys](idx-api-bridge-proxy.md#dashboard-api-keys).

## GIS public overlay (`/api/v1/gis`)

**New:** Florida **public government** parcel GeoJSON proxy for Leaflet overlays—**not MLS data**.

- **Routes:** `GET /api/v1/gis`, `GET /api/v1/mls/{mlsCode}/gis`
- **Docs:** [`docs/gis-api.md`](gis-api.md) (OpenAPI-style parameters, examples for Pinellas / Tampa bbox, failover, caching, compliance notes).
- **Caching:** Short **Laravel edge** TTL plus long-lived **Postgres origin** rows (per-source max age in days), invalidated when weekly metadata probes bump `gis_source_states.generation` or when you run `gis:clear-cache`.

Use this alongside `/api/v1/listings` with the same viewport parameters for a single map flow.
