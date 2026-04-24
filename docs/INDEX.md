# Quantyra GeoIDX — Documentation Index

Central index for all documentation in this project. Implementation code lives in this repository root, and **reference and integration guides** live under `docs/`.

---

## Quick links

| Document | Description |
|----------|-------------|
| [Bridge / MLS API](bridge-api-documentation.md) | Bridge Data Output API reference (Stellar MLS proxy usage). |
| [IDX-API Bridge proxy](idx-api-bridge-proxy.md) | Secured Bridge proxy: `/api/v1/*`, `?domain=`, auth, listings cache (15m), JSON photo URL rewrite to **idx-images**, `/images/*` disk + immutable CDN headers, audit, env, Docker. |
| [GoHighLevel OAuth (vendor)](gohighlevel-oauth-documentation.md) | Curated GHL OAuth 2.0, token refresh, scopes, webhooks (reference from marketplace docs). |
| [GHL Marketplace integration](ghl-marketplace-integration.md) | Quantyra implementation in **idx-api**: OAuth, onboarding, widgets, API routes, jobs. |
| [GHL deployment & operations](ghl-deployment-and-operations.md) | Docker, Dokploy, migrations, queues, scheduling. |
| [Docker builds](../README.md) | Production Dockerfiles in this project (`Dockerfile.idx-api`, `Dockerfile.idx-images`) with project-root build context. |
| [GHL environment variables](ghl-environment-variables.md) | All `GHL_*`, `IDX_*`, and related env vars for idx-api and compose. |
| [Stripe & Laravel Cashier](stripe-laravel-cashier.md) | `STRIPE_*`, `CASHIER_*`, webhook URLs, Dashboard vs CLI signing secrets, local forwarding. |
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

## Project layout summary

| Path | Role |
|------|------|
| `app/`, `routes/`, `config/`, `database/` | Laravel 13 + Octane: **secured Bridge MLS proxy** (`/api/v1/*`, images), **GHL Marketplace app**, widgets, webhooks, and **Stripe / Cashier** billing (when enabled). |
| `docs/` | Product, integration, deployment, and operations documentation. |
| `tests/` | Feature and unit test coverage for Bridge and GHL flows. |
| `Dockerfile.idx-api`, `Dockerfile.idx-images` | Production container images for API and image edge. |

For a full product overview, see the root [README.md](../README.md). **Docker / Dokploy:** build from project root using [`Dockerfile.idx-api`] and [`Dockerfile.idx-images`] as documented in [README.md](../README.md).

## Dev run commands

- Docker dev up/watch: `./scripts/docker-dev.sh up-watch`
- Docker dev down: `./scripts/docker-dev.sh down`
- Stripe webhook forwarding: VS Code task `Stripe Dev: Listen` (or `./scripts/stripe-dev.sh listen`)
