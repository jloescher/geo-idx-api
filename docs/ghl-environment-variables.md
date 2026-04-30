# GHL & IDX — Environment Variables Reference

Variables are set in **`.env`** for this project (and mirrored in host/runtime environment for `docker-compose` where noted). Defaults in `config/ghl.php` apply when env is missing.

---

## Quantyra public URLs

| Variable | Description | Example |
|----------|-------------|---------|
| `IDX_PLATFORM_URL` | IDX App host (GHL + independent dashboard): public IDX, checkout, embeds. | `https://idx.quantyralabs.cc` |
| `IDX_API_PUBLIC_URL` | IDX API host: REST endpoints and JS embed widgets (often same as `APP_URL`). | `https://idx-api.quantyralabs.cc` |
| `IDX_IMAGES_PUBLIC_URL` | IDX image rewrite proxy (MLS images via approved Bridge rewrite). | `https://idx-images.quantyralabs.cc` |
| `APP_URL` | Laravel **idx-api** URL; used for default `GHL_REDIRECT_URI` fragment. | `https://idx-api.quantyralabs.cc` |

---

## GHL OAuth & API

| Variable | Required | Description |
|----------|----------|-------------|
| `GHL_CLIENT_ID` | Yes (production) | Marketplace app client id. |
| `GHL_CLIENT_SECRET` | Yes | Client secret; never expose to browsers. |
| `GHL_REDIRECT_URI` | Yes | Must match GHL app **exactly**; default `{APP_URL}/oauth/leadconnector/callback`. |
| `GHL_AUTHORIZE_URL` | No | Default `https://marketplace.gohighlevel.com/oauth/chooselocation`. |
| `GHL_TOKEN_URL` | No | Default `https://services.leadconnectorhq.com/oauth/token`. |
| `GHL_LOCATION_TOKEN_URL` | No | Default `https://services.leadconnectorhq.com/oauth/locationToken`. |
| `GHL_SCOPES` | No | Space-separated scopes; see `config/ghl.php` default list. |
| `GHL_DEFAULT_USER_TYPE` | No | `Location` or `Company` for token exchange default. |
| `GHL_API_BASE_URL` | No | Default `https://services.leadconnectorhq.com`. |
| `GHL_API_VERSION` | No | API `Version` header, default `2021-07-28`. |
| `GHL_API_TIMEOUT` | No | HTTP timeout seconds. |
| `GHL_API_MAX_RETRIES` | No | Reserved for client retries. |

---

## GHL webhooks

| Variable | Description |
|----------|-------------|
| `GHL_WEBHOOK_REQUIRE_SIGNATURE` | `true`/`false`; when `false`, signature is skipped (local only). |
| `GHL_WEBHOOK_SECRET` | HMAC secret; defaults to `GHL_CLIENT_SECRET` in config if unset. |
| `GHL_WEBHOOK_SIGNATURE_HEADER` | Default `X-GHL-Signature`. |

---

## GHL admin & queues

| Variable | Description |
|----------|-------------|
| `GHL_ADMIN_REFRESH_TOKEN` | Shared secret for `POST /oauth/leadconnector/refresh` header `X-Quantyra-Admin-Token`. |
| `GHL_QUEUE_SYNC` | Queue name for lead sync (default `default`). |
| `GHL_QUEUE_WEBHOOKS` | Queue name for webhook jobs. |
| `GHL_QUEUE_MAINTENANCE` | Queue name for token refresh jobs. |

---

## GHL subscription tags (Stripe → GHL future use)

| Variable | Default |
|----------|---------|
| `GHL_SUBSCRIPTION_TAG_ACTIVE` | `quantyra-active` |
| `GHL_SUBSCRIPTION_TAG_PAST_DUE` | `quantyra-past-due` |
| `GHL_SUBSCRIPTION_TAG_CANCELLED` | `quantyra-cancelled` |
| `GHL_SUBSCRIPTION_TAG_TRIAL` | `quantyra-trial` |

---

## Audit

| Variable | Description |
|----------|-------------|
| `GHL_AUDIT_LOG_ENABLED` | `true`/`false`. |
| `GHL_AUDIT_LOG_PATH` | Filesystem path for secondary file log; default `storage/logs/ghl_audit.log` in app. |

---

## Widgets

| Variable | Description |
|----------|-------------|
| `GHL_WIDGET_RATE_LIMIT` | Requests per minute per throttle bucket (default `120`). |

---

## Geocoding

| Variable | Required | Description |
|----------|----------|-------------|
| `GOOGLE_MAPS_GEOCODING_API_KEY` | Yes (home value) | Google Maps Geocoding API key for resolving street addresses to coordinates. |
| `GEOCODING_PROVIDER` | No | Geocoding provider name (default `google`). |
| `GEOCODING_CACHE_TTL` | No | Cache TTL in seconds for geocoding results (default `2592000` = 30 days). |
| `GEOCODING_TIMEOUT` | No | HTTP timeout for geocoding requests in seconds (default `5`). |

---

## docker-compose (project root)

The `idx-api` service passes through the variables above; set them in the host `.env` consumed by Compose.

**Build:** images are built from the **project root** with **`Dockerfile.idx-api`** and **`Dockerfile.idx-images`**. See **[README.md](../README.md)**.

---

## Bridge MLS proxy (idx-api)

`BRIDGE_*` (including **`BRIDGE_IMAGE_REWRITE_HOSTS`** for JSON URL rewriting), `LISTINGS_CACHE_TTL`, `IMAGE_CACHE_*`, `IDX_IMAGES_PUBLIC_URL`, `IDX_API_INTERNAL_TOKEN`, and related variables are documented in **[idx-api-bridge-proxy.md](idx-api-bridge-proxy.md)** (full tables, **`images`** disk, Cloudflare-oriented cache headers, and routing notes). The root `docker-compose.yml` `idx-api` service also passes Bridge-related env defaults for Dokploy.

---

## Password hashing

`HASHING_DRIVER` controls the password hashing algorithm (default `argon2id`). See `config/hashing.php`.

| Variable | Default | Description |
|----------|---------|-------------|
| `HASHING_DRIVER` | `argon2id` | Hashing algorithm: `bcrypt`, `argon`, or `argon2id`. |
| `BCRYPT_ROUNDS` | `12` | Rounds for bcrypt (used when driver is `bcrypt`). |
| `HASHING_VERIFY` | `true` | Whether to verify hashes after creation for timing-attack protection. |
| `ARGON_MEMORY` | `65536` | Memory cost in KiB for argon2id. |
| `ARGON_THREADS` | `1` | Thread count for argon2id. |
| `ARGON_TIME` | `4` | Time cost for argon2id. |

---

## Stripe / Laravel Cashier

`STRIPE_KEY`, `STRIPE_SECRET`, `STRIPE_WEBHOOK_SECRET`, `CASHIER_CURRENCY`, `CASHIER_CURRENCY_LOCALE`, webhook URL (`/stripe/webhook` by default), and Stripe CLI forwarding are documented in **[stripe-laravel-cashier.md](stripe-laravel-cashier.md)**.

---

## Files to keep in sync

| File | Role |
|------|------|
| `.env.example` | Developer template for this project. |
| `.env` | Runtime environment values for local/dev/deploy builds. |
