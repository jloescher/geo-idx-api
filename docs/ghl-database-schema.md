# GHL Integration — Database Schema (PostgreSQL)

All tables below are created by migrations in **`idx-api/database/migrations/ghl/`** and loaded via `AppServiceProvider::loadMigrationsFrom()`. Use PostgreSQL in production (docker-compose / Patroni); local tests may use SQLite.

---

## `ghl_oauth_tokens`

Stores **Company** or **Location** OAuth tokens for the marketplace app.

| Column | Notes |
|--------|--------|
| `ghl_company_id`, `ghl_location_id` | GHL identifiers; `location_id` nullable for pure agency token. |
| `access_token`, `refresh_token` | Laravel **encrypted** at rest. |
| `access_token_hash` | `sha256(plain access token)`; unique; used for Bearer API lookup. |
| `user_type` | `Company` or `Location`. |
| `expires_at`, `refresh_expires_at` | Token lifetimes. |
| `scopes` | Granted scope string. |
| `status` | e.g. `active`, `revoked`. |
| `deleted_at` | Soft deletes. |

---

## `ghl_installed_locations`

Per-location install metadata and **subscription / teaser** counters.

| Column | Notes |
|--------|--------|
| `ghl_oauth_token_id` | FK → `ghl_oauth_tokens` (nullable if webhook arrives before OAuth row). |
| `ghl_company_id`, `ghl_location_id` | Unique pair. |
| `subscription_status` | `none`, `trial`, `active`, `past_due`, `cancelled`. |
| `mls_request_count`, `lead_count` | Teaser / usage counters. |
| `status` | `active` / `uninstalled`. |

---

## `ghl_registered_urls`

**MLS compliance:** registered HTTPS origins for widget and embed traffic.

| Column | Notes |
|--------|--------|
| `primary_url`, `additional_urls` (JSON) | Allowed origins (prefix match for Origin header). |
| `widget_api_key` | Public embed key (`qh_` prefix). |
| `integration_type` | `ghl_website`, `external_website`, or `both`. |
| `mls_agreement_acknowledged` | User attestation flag. |

---

## `ghl_widget_configs`

Per-location widget branding and gate defaults.

| Column | Notes |
|--------|--------|
| `widget_theme`, colors, `font_family` | UI hints for future rich widgets. |
| `gate_after_views`, `require_otp` | Align with geo-web gating philosophy. |

---

## `quantyra_leads`

Inbound leads from widgets (and future sources).

| Column | Notes |
|--------|--------|
| `ghl_location_id` | Target GHL location. |
| `lead_type` | Maps to `ghl_lead_mappings.quantyra_lead_type`. |
| `payload` | JSON document (name, email, etc.). |

---

## `ghl_sync_logs`

CRM sync audit trail.

| Column | Notes |
|--------|--------|
| `quantyra_lead_id` | FK → `quantyra_leads`. |
| `ghl_contact_id`, `ghl_opportunity_id` | IDs returned by GHL API when successful. |
| `sync_status` | `pending`, `success`, `failed`, `retrying`. |
| `request_payload`, `response_payload` | JSON snapshots for debugging. |

---

## `ghl_webhook_events`

Inbox for marketplace webhooks.

| Column | Notes |
|--------|--------|
| `webhook_id` | Unique when GHL sends id; else synthetic hash may be used upstream. |
| `event_type` | Normalized type string. |
| `payload` | Full JSON body. |
| `processing_status` | `received`, `processing`, `processed`, `failed`. |

---

## `ghl_audit_logs`

Stellar MLS–oriented **enhanced audit** (API + webhook correlation).

| Column | Notes |
|--------|--------|
| `logged_at` | Event time. |
| `api_endpoint`, `request_method` | Logical endpoint or webhook tag. |
| `latency_ms`, `response_status` | GHL HTTP client metrics where applicable. |
| `is_mls_data_access`, `compliance_verified` | Compliance flags. |

---

## `ghl_lead_mappings`

Maps **`quantyra_lead_type`** → contact/opportunity/tags behavior.

Seeded by **`GhlConfigSeeder`** (`php artisan db:seed --class=GhlConfigSeeder`).

| Column | Notes |
|--------|--------|
| `creates_contact`, `creates_opportunity` | Booleans. |
| `opportunity_pipeline`, `opportunity_stage` | Optional GHL pipeline/stage ids when known. |
| `default_tags` | JSON array of tag names. |
| `domain_tag_prefix` | When true, append domain tag from lead row. |

---

## ER summary (logical)

```
ghl_oauth_tokens 1──* ghl_registered_urls 1──* ghl_widget_configs
        │
        └──* ghl_installed_locations

quantyra_leads 1──* ghl_sync_logs

ghl_webhook_events (standalone inbox)

ghl_audit_logs ──? ghl_oauth_tokens
```

---

## Related

- [ghl-marketplace-integration.md](ghl-marketplace-integration.md)
- [ghl-deployment-and-operations.md](ghl-deployment-and-operations.md)
