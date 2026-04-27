# GoHighLevel Marketplace Integration (Quantyra GeoIDX)

This document describes the **Quantyra-built** GoHighLevel (GHL) Marketplace application that runs inside **`idx-api`**. Vendor OAuth semantics are summarized in [gohighlevel-oauth-documentation.md](gohighlevel-oauth-documentation.md); this file focuses on **our routes, flows, and behavior**.

---

## URL architecture

Canonical Quantyra production layout:

| Surface | Production URL | Purpose |
|---------|----------------|---------|
| IDX App | `https://idx.quantyralabs.cc` | IDX app for GHL and independent dashboard (public IDX, subscribe/checkout, full embed). |
| IDX API | `https://idx-api.quantyralabs.cc` | IDX API for HTTP endpoints and JS embed widgets (OAuth callbacks, webhooks, GHL REST proxies, widget loader). |
| IDX images | `https://idx-images.quantyralabs.cc` | IDX image rewrite proxy (MLS image CDN in front of approved Bridge paths). |

Configure via `IDX_PLATFORM_URL`, `IDX_API_PUBLIC_URL`, `IDX_IMAGES_PUBLIC_URL`, and `APP_URL` (see [ghl-environment-variables.md](ghl-environment-variables.md)).

---

## High-level architecture

- **Modular PHP namespace:** `App\Ghl\` under `idx-api/app/Ghl/` with areas: `OAuth`, `Sync`, `Webhooks`, `Api`, `Widgets`, and shared `Services` (audit).
- **PostgreSQL:** Encrypted OAuth tokens, installed locations, registered MLS domains, widget keys, sync logs, webhook inbox, audit rows, lead type mappings, `quantyra_leads`.
- **Queues:** Lead sync, webhook processing, token refresh (Laravel queue; default connection from `QUEUE_CONNECTION`).
- **MLS compliance:** After OAuth, users **register HTTPS origins** used for embeds; widget middleware enforces **Origin** against those URLs. Audit events are written for GHL API and webhook activity.

---

## OAuth 2.0 (Authorization Code Grant)

Public HTTP paths use the **`leadconnector`** prefix (whitelabel); implementation still lives under `App\Ghl\`.

### Endpoints (idx-api)

| Step | Route | Description |
|------|--------|-------------|
| Installation landing | `GET /leadconnector/install` | Blade UI; links to authorize. |
| Authorize | `GET /oauth/leadconnector/authorize` | Builds state in session, redirects to GHL `chooselocation` with `client_id`, `redirect_uri`, `scope`, `state`. Optional query: `user_type=Company` or `Location` (default from `GHL_DEFAULT_USER_TYPE`). |
| Callback | `GET /oauth/leadconnector/callback` | Validates `state`, exchanges `code` at `POST https://services.leadconnectorhq.com/oauth/token` (`application/x-www-form-urlencoded`, `Accept: application/json`), persists tokens, sets session `ghl_pending_oauth_token_id`. |
| Token refresh (admin) | `POST /oauth/leadconnector/refresh` | Header `X-Quantyra-Admin-Token` must match `GHL_ADMIN_REFRESH_TOKEN`; body `token_id` for server-side refresh job path. |

### Token storage

- **Model:** `App\Ghl\OAuth\Models\GhlOAuthToken`
- **Encryption:** Laravel `encrypted` casts on `access_token` and `refresh_token`.
- **Bearer lookup:** `access_token_hash = sha256(plain_access_token)` for `GET /api/leadconnector/*` without storing plaintext for lookup.
- **Hybrid installs:** Supports **Company** (agency) and **Location** tokens. Agency → location token exchange is implemented in `LocationTokenService` calling `POST /oauth/locationToken` (see vendor doc).

### Post-OAuth: MLS domain registration

| Route | Method | Purpose |
|-------|--------|---------|
| `/leadconnector/register-urls` | GET / POST | Requires session `ghl_pending_oauth_token_id`. Collects primary + additional **https** URLs, integration type, MLS acknowledgment; optional **manual GHL location id** when the token has no `locationId` (agency flow). Creates `ghl_registered_urls` + `ghl_widget_configs` and a **widget API key** (`qh_…`). |
| `/leadconnector/installation-complete` | GET | Shows embed snippets and upgrade link to IDX platform. |

---

## Protected GHL API (Bearer token)

All routes are prefixed with **`/api`** (Laravel `api.php`) → **`/api/leadconnector/*`**.

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/leadconnector/leads` | `Authorization: Bearer <GHL access_token>` |
| GET | `/api/leadconnector/leads/{id}` | Same |
| GET | `/api/leadconnector/subscriptions` | Same |
| GET | `/api/leadconnector/stats` | Same |
| GET | `/api/leadconnector/config` | Same |

**Agency tokens** without a stored `ghl_location_id` must pass `?location_id=` for scoping where applicable.

Middleware: `App\Ghl\Http\Middleware\AuthenticateGhlLocation`.

---

## Webhooks

| Route | Method | Middleware |
|-------|--------|------------|
| `/webhooks/leadconnector` | POST | `VerifyGhlWebhookSignature`, throttle |

Behavior:

1. Persist (or dedupe by) `webhookId` in `ghl_webhook_events`.
2. Dispatch `ProcessGhlWebhookJob` (async if queue worker running).
3. `WebhookDispatcher` routes by normalized `type`: `INSTALL` / `APPINSTALL`, `UNINSTALL` / `APPUNINSTALL`, CRM events (contacts, opportunities, notes, tasks) to handlers; CRM events are audit-logged.

**Signature (configurable):** When `GHL_WEBHOOK_REQUIRE_SIGNATURE=true`, raw body is verified with `hash_hmac('sha256', body, GHL_WEBHOOK_SECRET)` against header `GHL_WEBHOOK_SIGNATURE_HEADER` (default `X-GHL-Signature`). **Confirm against live GHL** when your app is registered; adjust if the marketplace uses a different scheme.

---

## JS widgets (embeddable)

| Route | Purpose |
|-------|---------|
| `GET /widget/loader.js` | Returns loader script; use `data-api-key`, `data-location-id`, `data-widget`. |
| `GET /widget/config/{apiKey}` | JSON widget + theme hints. |
| `GET /widget/search|lead-form|showcase/{apiKey}` | HTML shells (MLS data must still flow through approved Bridge proxy paths). |
| `OPTIONS /widget/api/leads?api_key=` | CORS preflight for cross-origin POST. |
| `POST /widget/api/leads` | Creates `quantyra_leads`, dispatches `SyncLeadToGhlJob`. |

Middleware chain: validate API key → validate **Origin** (or Referer) against registered URLs → append `Access-Control-Allow-Origin` → throttle.

---

## Lead sync → GHL CRM

- **Model:** `App\Ghl\Sync\Models\QuantyraLead`
- **Job:** `SyncLeadToGhlJob` → `LeadSyncService` uses `GhlLeadMapping` rows (seeded by `GhlConfigSeeder`) to create contacts (and optionally opportunities) via `GhlApiClient` → HighLevel REST (`/contacts/`, `/opportunities/` with `Version` header).
- **Stripe:** `SubscriptionSyncService` + `SyncSubscriptionStatusJob` update `ghl_installed_locations.subscription_status`; **wire Cashier webhooks** when billing is connected.

---

## Audit & Stellar MLS

- **Table:** `ghl_audit_logs` (plus optional file channel via `GHL_AUDIT_LOG_PATH`).
- **Service:** `App\Ghl\Services\GhlAuditService` records endpoint, method, latency, HTTP status, location/company ids, token id, MLS-related flags.

---

## Migrations & seeding

Migrations live in `database/migrations/ghl/` and are registered in `AppServiceProvider` via `loadMigrationsFrom()`.

```bash
cd idx-api
php artisan migrate
php artisan db:seed --class=GhlConfigSeeder
```

Schema overview: [ghl-database-schema.md](ghl-database-schema.md).

---

## Related documents

- [ghl-api-routes-reference.md](ghl-api-routes-reference.md) — Curl examples and response notes.
- [ghl-deployment-and-operations.md](ghl-deployment-and-operations.md) — Docker, Dokploy, workers.
- [../README.md](../README.md) — Project Dockerfiles (`Dockerfile.*`) and build context.
- [ghl-environment-variables.md](ghl-environment-variables.md) — Env tables.
- [superpowers/specs/2026-04-22-ghl-marketplace-integration-design.md](superpowers/specs/2026-04-22-ghl-marketplace-integration-design.md) — Full design decisions.
