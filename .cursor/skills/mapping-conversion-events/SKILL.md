---
name: mapping-conversion-events
description: Maps funnel events, conversion tracking, and success signals across the IDX API including GHL OAuth installs, widget lead submissions, subscription checkouts, and Bridge proxy access patterns.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Mapping Conversion Events Skill

Tracks user journey events from initial widget interaction through GHL OAuth installation to subscription activation. Conversion signals live in database tables (`ghl_oauth_tokens`, `quantyra_leads`, `ghl_installed_locations`, `bridge_proxy_audit_logs`), Stripe webhooks, and application logs.

## Quick Start

Identify conversion event locations:
```bash
# GHL OAuth funnel tables
grep -r "ghl_oauth_tokens\|ghl_installed_locations\|quantyra_leads" database/migrations/

# Widget lead ingestion
grep -r "widget.*leads\|QuantyraLead" app/Ghl/

# Subscription events (Stripe/Cashier)
grep -r "Subscription\|subscription_status" app/ --include="*.php"

# Bridge proxy audit trail
grep -r "bridge_proxy_audit_logs\|BridgeProxyAuditLog" app/
```

Trace a location's journey:
```bash
# Find OAuth token by location ID
sqlite3 database/database.sqlite "SELECT id, ghl_location_id, status, created_at FROM ghl_oauth_tokens WHERE ghl_location_id = 'LOC_ID'"

# Check registered URLs and widget config
sqlite3 database/database.sqlite "SELECT widget_api_key, primary_url FROM ghl_registered_urls WHERE ghl_oauth_token_id = TOKEN_ID"

# View lead submissions for location
sqlite3 database/database.sqlite "SELECT lead_type, created_at FROM quantyra_leads WHERE ghl_location_id = 'LOC_ID'"
```

## Key Concepts

**OAuth Install Funnel** (`routes/ghl-web.php`)
- Entry: `GET /leadconnector/install` → Blade landing
- Redirect: `GET /oauth/leadconnector/authorize` → GHL chooselocation
- Conversion: `GET /oauth/leadconnector/callback` → token exchange
- Activation: `POST /leadconnector/register-urls` → widget key issued
- Success signal: `ghl_oauth_tokens.status = 'active'` with `ghl_installed_locations.subscription_status`

**Widget Lead Conversion** (`routes/ghl-widget.php`)
- Entry: `GET /widget/search|lead-form|showcase/{apiKey}`
- Conversion: `POST /widget/api/leads` → creates `quantyra_leads` row
- Success signal: `ghl_sync_logs.sync_status = 'success'` (contact created in GHL)

**Subscription Tiers** (`app/Billing/SubscriptionCatalog.php`)
- Pro ($39/mo): 3 domains, teaser gating
- Smart ($79/mo): 5 domains, full GHL app, OTP
- Ultra ($179/mo): Unlimited, 2M API calls, dev keys
- Mega ($449/mo): SLA, custom branding
- Success signal: Stripe `checkout.completed` → `ghl_installed_locations.subscription_status = 'active'`

**Bridge Proxy Teaser Gates** (`app/Services/Bridge/BridgeTeaser.php`)
- Entry: `GET /api/v1/listings` with domain or token auth
- Teaser applied: Non-`idx:full` requests capped at 3 listings
- Full access: Sanctum token with `idx:full` ability
- Audit trail: `bridge_proxy_audit_logs.request_type = 'listings'`

## Common Patterns

**Track GHL install completions:**
Check `ghl_webhook_events` for `type = 'INSTALL'` or `APPINSTALL`, correlated with `ghl_oauth_tokens.created_at`.

**Monitor lead submission flow:**
```sql
SELECT 
    ql.lead_type,
    ql.created_at as submitted_at,
    gsl.sync_status,
    gsl.ghl_contact_id
FROM quantyra_leads ql
LEFT JOIN ghl_sync_logs gsl ON gsl.quantyra_lead_id = ql.id
WHERE ql.ghl_location_id = 'TARGET_LOCATION';
```

**Identify teaser-to-full conversion:**
Compare `bridge_proxy_audit_logs` entries where `token_name IS NULL` (domain/teaser) versus `token_name = 'geo-web-internal'` (full access).

**Subscription churn detection:**
Join `ghl_installed_locations` with Stripe webhook events; `subscription_status` transitions `active` → `cancelled` or `past_due`.

**Widget API key validation flow:**
`widget_api_key` (prefix `qh_`) in `ghl_registered_urls` → middleware validates Origin against `primary_url`/`additional_urls` → CORS headers appended on response.