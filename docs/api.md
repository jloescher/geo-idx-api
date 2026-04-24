# idx-api HTTP API overview

This document summarizes **versioned JSON APIs** exposed by the Laravel idx-api service (Octane + FrankenPHP ready).

## Authenticated IDX / Bridge proxy (`/api/v1`)

All routes in `routes/api.php` under the `v1` prefix use the `domain.token` middleware (domain header / referer host **or** Sanctum bearer with `idx:access` / `idx:full`).

Core resources: listings, agents, offices, RESO Property, members, public parcels bridge, etc. See `docs/idx-api-bridge-proxy.md` and `docs/bridge-api-documentation.md`.

## GIS public overlay (`/api/v1/gis`)

**New:** Florida **public government** parcel GeoJSON proxy for Leaflet overlays—**not MLS data**.

- **Routes:** `GET /api/v1/gis`, `GET /api/v1/mls/{mlsCode}/gis`
- **Docs:** [`docs/gis-api.md`](gis-api.md) (OpenAPI-style parameters, examples for Pinellas / Tampa bbox, failover, caching, revenue notes).

Use this alongside `/api/v1/listings` with the same viewport parameters for a single map flow.
