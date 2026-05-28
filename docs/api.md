# idx-api HTTP API overview

This document summarizes the currently registered HTTP surface in the Go Fiber app (`internal/api/routes.go` + dashboard/static mounts).

## Canonical machine-readable API doc

- OpenAPI source file: `docs/yaak-api-collection.json` (synced into the API embed via `make openapi-sync`)
- Live spec: [`GET /openapi.json`](/openapi.json) (public, no auth)
- Interactive explorer: [`GET /swagger`](/swagger) (Swagger UI; loads `/openapi.json`)

## Route groups

- Infrastructure (public): `/healthz`, `/readyz`, `/health/replicas`, `/metrics`, `/openapi.json`, `/swagger`
- Marketing/static (public): `/`, `/static/*`
- API auth: `/api/auth/token`, `/api/auth/user`
- Versioned API: `/api/v1/*` (MLS proxy, GIS, search, comps, operations)
- Admin API (session-cookie protected): `/api/v1/admin/*`
- Image proxy: `/images/:listingKey/:photoId`
- Dashboard HTML/session flows: `/login`, `/logout`, `/dashboard/*`, `/invite/:token`

## Authentication model (code-aligned)

- `DomainToken` middleware protects `/api/v1/*` and `/images/*`.
- Token mode: `Authorization: Bearer <token>` with `idx:access` or `idx:full`, plus `X-Domain-Slug` header or `?domain=` query.
- Domain mode (no bearer): domain resolved from `X-Domain-Slug`, `?domain=`, or `Referer` host.
- `MLSAccess` middleware resolves allowed dataset/feed for `/api/v1/*` except GIS bypass paths.
- Admin API routes (`/api/v1/admin/*`) require dashboard `session_id` cookie via `SessionAuthMiddleware`.

## Token issuance

`POST /api/auth/token` accepts:

- `email` (string)
- `password` (string)

It returns a token payload:

- `token` (plaintext PAT)
- `abilities` (currently includes `idx:full`)

## Detailed endpoint reference

For a full route-by-route table (methods, auth, handlers, and behavior notes), see `docs/routes-reference.md`.

### Search (`POST /api/v1/search`)

Hybrid mirror + upstream search. Active/Pending defaults to PostGIS; Closed uses live RESO. Public results exclude non-IDX / non-display listings (see `docs/listings-mirror.md`). Full field list, aliases, and routing: [idx-api-bridge-proxy.md — Search endpoint](idx-api-bridge-proxy.md#search-endpoint-post-apiv1search).

**Parsing:** JSON numbers/booleans are preferred; string-encoded numbers (`"250000"`) and booleans (`"true"`) are accepted. Empty strings omit a filter. Invalid types return `400` with `invalid search body: <detail>`.

**Pagination:** `limit` / `skip` or `page.limit` / `page.skip`. Response: `{ "results", "hasMore", "nextSkip" }`.

**Dataset:** `?dataset=stellar|beaches` query param (not JSON body). On Bridge/Spark **web/RESO proxy** routes, `dataset` is used for IDX routing only and is **not** forwarded to the upstream MLS API.

**Results shape:** PostGIS search returns typed scalar RESO fields (no `Media`, `Room`, `Unit`, `OpenHouse`, or merged custom fields). Use property/detail proxy routes for full navigation payloads.

| JSON field | Description |
|------------|-------------|
| `min_price`, `max_price`, `min_beds`, `max_beds`, `min_baths`, `min_sqft`, `max_sqft` | Price, beds, baths, living area filters (aliases: `beds_min`, `living_area_min`, …). |
| `city` | Geography filter: expands via GIS autocomplete, then `LIKE` on `listings.city` / `county_or_parish` (not exact equality). |
| `county_or_parish` | County display name or slug from autocomplete; same OR-LIKE geography expansion. |
| `statuses` | e.g. `Active`, `Pending`, `Closed`. Blank array entries are ignored. |
| `low_risk_floodzone` | When true, mirror leg requires `low_risk_flood_zone_yn` (FEMA-enriched). |
| `min_monthly_fees`, `max_monthly_fees` | Filter `estimated_total_monthly_fees`. |
| `remarks_query` | Optional full-text search on typed `public_remarks` (`plainto_tsquery`, English). |
| `focus_areas`, `sort`, `sort_dir`, `geo` | **Not implemented** — ignored if sent; use `city` / `county_or_parish` for geography. |
