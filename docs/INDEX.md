# Quantyra GeoIDX — Documentation Index

Central index for all documentation in this repository. Implementation code lives under `idx-api/`, `geo-web/`, and `mobile/`; **reference and integration guides** live here under `docs/`.

---

## Quick links

| Document | Description |
|----------|-------------|
| [Bridge / MLS API](bridge-api-documentation.md) | Bridge Data Output API reference (Stellar MLS proxy usage). |
| [IDX-API Bridge proxy](idx-api-bridge-proxy.md) | Secured Bridge proxy: `/api/v1/*`, `?domain=`, auth, listings cache (15m), JSON photo URL rewrite to **idx-images**, `/images/*` disk + immutable CDN headers, audit, env, Docker. |
| [GoHighLevel OAuth (vendor)](gohighlevel-oauth-documentation.md) | Curated GHL OAuth 2.0, token refresh, scopes, webhooks (reference from marketplace docs). |
| [GHL Marketplace integration](ghl-marketplace-integration.md) | Quantyra implementation in **idx-api**: OAuth, onboarding, widgets, API routes, jobs. |
| [GHL deployment & operations](ghl-deployment-and-operations.md) | Docker, Dokploy, migrations, queues, scheduling. |
| [Docker builds (monorepo)](../docker/README.md) | Production Dockerfiles under `docker/`; **repository root** build context for Dokploy (`idx-api`, `geo-web`, `idx-images`). |
| [GHL environment variables](ghl-environment-variables.md) | All `GHL_*`, `IDX_*`, and related env vars for idx-api and compose. |
| [GHL database schema](ghl-database-schema.md) | PostgreSQL tables created for GHL, leads, audit, widgets. |
| [GHL API & routes reference](ghl-api-routes-reference.md) | HTTP routes, auth, widgets, curl examples. |

---

## Design & planning (internal)

| Path | Purpose |
|------|---------|
| [superpowers/specs/2026-04-22-ghl-marketplace-integration-design.md](superpowers/specs/2026-04-22-ghl-marketplace-integration-design.md) | Approved product/design spec for the GHL app. |
| [superpowers/plans/2026-04-22-ghl-marketplace-implementation.md](superpowers/plans/2026-04-22-ghl-marketplace-implementation.md) | Implementation plan checklist. |

---

## Official external references

- [GHL Marketplace — Getting Started](https://marketplace.gohighlevel.com/docs/oauth/GettingStarted)
- [OAuth 2.0](https://marketplace.gohighlevel.com/docs/Authorization/OAuth2.0/)
- [Create Marketplace App](https://marketplace.gohighlevel.com/docs/oauth/CreateMarketplaceApp/)
- [Scopes](https://marketplace.gohighlevel.com/docs/Authorization/Scopes/)
- [Webhook category](https://marketplace.gohighlevel.com/docs/category/webhook)

---

## Monorepo layout (read-only summary)

| Path | Role |
|------|------|
| `idx-api/` | Laravel 13 + Octane: **secured Bridge MLS proxy** (`/api/v1/*`, images), **GHL Marketplace app**, widgets, webhooks. |
| `geo-web/` | Public multi-domain IDX sites (not documented in this index beyond repo README). |
| `mobile/` | Flutter app (teaser mode per Stellar MLS until Exhibit A amendment). |
| `scripts/` | Build and deploy helpers. |

For a full product overview, see the root [README.md](../README.md). **Docker / Dokploy:** all production Dockerfiles live under **`docker/`** with monorepo root build context — see [docker/README.md](../docker/README.md).
