# GoHighLevel Marketplace Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship full GHL Marketplace OAuth, webhooks, lead sync, MLS URL registration, JS widgets, audit logging, and Dokploy-ready env in `idx-api` only.

**Architecture:** Modular `App\Ghl\{OAuth,Sync,Webhooks,Api,Widgets}` under `app/Ghl/`, PostgreSQL tables per spec, Laravel HTTP client to `services.leadconnectorhq.com`, queued webhook processing and token refresh, Bearer `access_token_hash` lookup for `/api/leadconnector/*`.

**Tech Stack:** Laravel 13, Octane/Swoole, Sanctum (optional future), PostgreSQL, Cashier (Stripe webhook hook point stub), `Illuminate\Support\Facades\Http`.

**Spec:** `docs/superpowers/specs/2026-04-22-ghl-marketplace-integration-design.md`

**URLs:** IDX `idx.quantyralabs.cc`, API `idx-api.quantyralabs.cc`, images `idx-images.quantyralabs.cc`.

---

## Phase A: Database & config

- [ ] Add migrations under `database/migrations/ghl/` + `quantyra_leads`.
- [ ] Add `config/ghl.php` aggregating oauth/sync/webhooks/widgets + `IDX_PLATFORM_URL`.
- [ ] Extend root `.env.example` with `GHL_*` and `IDX_PLATFORM_URL`.

## Phase B: OAuth + onboarding UI

- [ ] `OAuthService`, `TokenRefreshService`, `LocationTokenService`.
- [ ] Controllers: authorize, callback, refresh (admin secret), URL registration, install, installation-complete.
- [ ] Middleware: `ValidateOAuthState`, `RequiresPendingGhlInstallSession`.
- [ ] Blade views for install, register URLs, complete + session keys.

## Phase C: API client, sync, jobs

- [ ] `GhlApiClient` + `ContactClient`, `OpportunityClient` (minimal v2 paths).
- [ ] `LeadSyncService`, `ContactCreator`, `OpportunityCreator`, `TagManager`, `SubscriptionSyncService` (tag sync stub).
- [ ] Jobs: `SyncLeadToGhlJob`, `RefreshGhlTokensJob`, `ProcessGhlWebhookJob`, `BulkLocationTokenJob` (stub dispatch).

## Phase D: Webhooks + audit

- [ ] `WebhookSignatureService` (HMAC optional via `GHL_WEBHOOK_SECRET`), `WebhookDispatcher`, handlers (INSTALL/UNINSTALL + CRM events → persist + audit).
- [ ] `GhlAuditService` → `ghl_audit_logs` + optional file append.

## Phase E: HTTP routes + widgets

- [ ] Register `routes/api.php`, `routes/ghl-web.php`, `routes/ghl-widget.php` in `bootstrap/app.php`.
- [ ] `GhlApiController` + `AuthenticateGhlLocation` middleware (Bearer → token row).
- [ ] Widget controllers: `loader.js`, config/search/lead-form/showcase HTML+JS, `POST /widget/api/leads`.
- [ ] `ValidateWidgetApiKey`, `ValidateWidgetOrigin`, `WidgetRateLimit`.
- [ ] `GET /leadconnector/embed/{locationId}` → redirect to `IDX_PLATFORM_URL/embed/{id}`.

## Phase F: Ops + tests

- [ ] `routes/console.php`: schedule token refresh hourly.
- [ ] `GhlConfigSeeder` for `ghl_lead_mappings`.
- [ ] Feature tests: OAuth URL build, webhook stores event, widget origin rejected, API auth.
- [ ] Update `docker-compose.yml` (`idx-api` GHL env), `Dockerfile.idx-api` comment if needed.

---

## Self-review (plan vs spec)

| Spec area | Covered by |
|-----------|------------|
| Hybrid OAuth | Authorize `user_type`, token exchange, `LocationTokenService` |
| Contacts/Opp/Tags | `LeadSyncService` + clients + mappings seeder |
| Stripe external | `SubscriptionSyncService` + route for Cashier hook (document in code) |
| Max scopes | `config('ghl.oauth.scopes')` |
| Full webhooks | Dispatcher + all handler classes |
| Enhanced audit | `GhlAuditService` + `ghl_audit_logs` |
| URL registration | `ghl_registered_urls` + controllers |
| JS widgets | `routes/ghl-widget.php` + middleware |
| idx vs idx-api URLs | `config('ghl.urls')` |

**Gaps explicitly deferred:** Full Stripe webhook controller (only service + TODO route comment until Cashier wiring exists); exact GHL contact v2 field names may need tuning against live API.
