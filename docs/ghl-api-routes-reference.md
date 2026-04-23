# GHL Integration — HTTP API & Routes Reference

All paths below are served by **`idx-api`** unless noted. Production base URL: `https://idx-api.quantyralabs.cc` (or your `APP_URL`).

---

## Web routes (`routes/ghl-web.php`)

Middleware: **`web`** (session, CSRF on POST except where excluded).

| Method | Path | Name | Notes |
|--------|------|------|--------|
| GET | `/leadconnector/install` | `leadconnector.install` | Marketplace **Installation URL** target; start here. |
| GET | `/oauth/leadconnector/authorize` | `leadconnector.oauth.authorize` | Redirect to GHL. Query: `user_type=Company` optional. |
| GET | `/oauth/leadconnector/callback` | `leadconnector.oauth.callback` | OAuth redirect handler; CSRF applies (GET exempt). |
| POST | `/oauth/leadconnector/refresh` | `leadconnector.oauth.refresh` | Server token refresh; header `X-Quantyra-Admin-Token: {GHL_ADMIN_REFRESH_TOKEN}`, body `token_id`. |
| GET | `/leadconnector/register-urls` | `leadconnector.register-urls` | Requires pending install session. |
| POST | `/leadconnector/register-urls` | `leadconnector.register-urls.store` | MLS URL registration form. |
| GET | `/leadconnector/installation-complete` | `leadconnector.installation-complete` | Post-registration embed instructions. |
| POST | `/webhooks/leadconnector` | `leadconnector.webhooks` | GHL marketplace webhooks; **CSRF excluded** in `bootstrap/app.php`. |
| GET | `/leadconnector/embed/{locationId}` | `leadconnector.embed` | **302** redirect to `IDX_PLATFORM_URL/embed/{locationId}`. |

---

## API routes (`routes/api.php`)

Laravel prefixes these with **`/api`**. Middleware: **`api`** stack + **`AuthenticateGhlLocation`**.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/leadconnector/leads` | Paginated `quantyra_leads` for `location_id` (from token or `?location_id=`). |
| GET | `/api/leadconnector/leads/{id}` | Single lead; scoped to location when token has location. |
| GET | `/api/leadconnector/subscriptions` | Subscription snapshot + `upgrade_url` on IDX platform. |
| GET | `/api/leadconnector/stats` | Teaser stats (`mls_request_count`, `lead_count`, gating flags). |
| GET | `/api/leadconnector/config` | Widget config + `ghl_lead_mappings` rows. |

**Authentication:** Header `Authorization: Bearer <GHL access_token>`  
The access token must match a row in `ghl_oauth_tokens` via `sha256(token)` = `access_token_hash`, `status = active`, and `expires_at` in the future.

---

## Widget routes (`routes/ghl-widget.php`)

Loaded in `bootstrap/app.php` **`then`** callback with **`api`** middleware (no `/api` prefix).

| Method | Path | Middleware highlights |
|--------|------|-------------------------|
| GET | `/widget/loader.js` | Throttle only. Route name: `leadconnector.widget.loader`. |
| GET | `/widget/config/{apiKey}` | API key + Origin + CORS + throttle. |
| GET | `/widget/search/{apiKey}` | Same. |
| GET | `/widget/lead-form/{apiKey}` | Same. |
| GET | `/widget/showcase/{apiKey}` | Same. |
| OPTIONS | `/widget/api/leads` | Query `api_key`; validates registered origin for preflight. |
| POST | `/widget/api/leads` | JSON body includes `api_key`, `lead_type`, optional name/email/phone, etc. |

---

## Example requests

### Install page

```bash
curl -sS -o /dev/null -w "%{http_code}" https://idx-api.quantyralabs.cc/leadconnector/install
```

### Stats (after OAuth)

```bash
curl -sS \
  -H "Authorization: Bearer YOUR_GHL_ACCESS_TOKEN" \
  "https://idx-api.quantyralabs.cc/api/leadconnector/stats?location_id=YOUR_LOCATION_ID"
```

### Webhook (local, signature disabled)

```bash
curl -sS -X POST https://idx-api.quantyralabs.cc/webhooks/leadconnector \
  -H "Content-Type: application/json" \
  -d '{"type":"INSTALL","webhookId":"wh-test-1","companyId":"c1","locationId":"l1","userId":"u1"}'
```

### Widget lead (cross-origin; Origin must match registered URL)

```bash
curl -sS -X POST https://idx-api.quantyralabs.cc/widget/api/leads \
  -H "Origin: https://your-registered-domain.com" \
  -H "Content-Type: application/json" \
  -d '{"api_key":"qh_...","lead_type":"showing_request","first_name":"A","last_name":"B","email":"a@b.com"}'
```

### CORS preflight

```bash
curl -sS -X OPTIONS "https://idx-api.quantyralabs.cc/widget/api/leads?api_key=qh_..." \
  -H "Origin: https://your-registered-domain.com"
```

---

## GHL Marketplace form fields (checklist)

Use these exact production URLs when creating the app in the GHL developer dashboard:

| Field | Value |
|-------|--------|
| Redirect URI | `https://idx-api.quantyralabs.cc/oauth/leadconnector/callback` |
| Installation URL | `https://idx-api.quantyralabs.cc/leadconnector/install` |
| Webhook URL | `https://idx-api.quantyralabs.cc/webhooks/leadconnector` |
| Scopes | Match `GHL_SCOPES` in [ghl-environment-variables.md](ghl-environment-variables.md) |

---

## Source map (implementation)

| Area | Path under `idx-api/app/Ghl/` |
|------|-------------------------------|
| OAuth controllers | `OAuth/Controllers/` |
| OAuth services | `OAuth/Services/` |
| HTTP + GHL API | `Http/`, `Api/Clients/` |
| Sync + leads | `Sync/` |
| Webhooks | `Webhooks/` |
| Widgets | `Widgets/` |
| Audit | `Services/GhlAuditService.php` |
