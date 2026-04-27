# Quantyra IDX API

Laravel 13 + Octane service powering Quantyra's Bridge MLS proxy, GoHighLevel (GHL) Marketplace integration, GIS parcel/geometry proxy, lead ingestion, and secured image proxy delivery. The service sits between real estate MLS data (Bridge Data Output / Stellar MLS), public ArcGIS parcel sources, GHL CRM widgets, and subscriber dashboards. Three public surfaces: **idx.quantyralabs.cc** (app/marketing), **idx-api.quantyralabs.cc** (API/widgets), **idx-images.quantyralabs.cc** (image proxy).

## Tech Stack

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| Runtime | PHP | 8.3+ | Server language (8.5 in production Docker) |
| Web Server | FrankenPHP (via Laravel Octane) | 2.x | High-performance concurrent request handling |
| Framework | Laravel | 13.x | Application skeleton, routing, ORM, queues |
| Language | PHP | 8.3+ | Strong typing, constructor promotion, PHP attributes for model fillable/hidden |
| Database | SQLite (local) / PostgreSQL (prod) | - | Eloquent ORM; local uses `:memory:` SQLite, production uses pgsql |
| Frontend | Livewire 3 + Blade + Tailwind CSS 4 | 3.x / 4.x | Server-rendered marketing pages and subscriber dashboard |
| Build | Vite | 8.x | CSS/JS bundling with Tailwind plugin |
| Auth | Laravel Sanctum + Fortify | 4.x / 1.36.x | API tokens for server-to-server; login/register/2FA views |
| Billing | Laravel Cashier (Stripe) | 16.x | Subscription tiers: Pro ($39/mo), Smart ($79/mo), Ultra ($179/mo), Mega ($449/mo) with metered overage |
| Observability | Pulse + Telescope + Debugbar | 1.x / 5.x / 4.x | Production metrics, local debugging, rate-limited behind HTTP Basic Auth |

## Quick Start

```bash
# Prerequisites: PHP 8.3+, Composer, Node 20+, SQLite (local dev)

# Installation
cp .env.example .env
composer install
php artisan key:generate
php artisan migrate

# Frontend assets (Livewire/Blade dashboard + marketing)
npm install
npm run build

# Development (server + queue + logs + Vite HMR in parallel)
composer dev

# Run tests (uses in-memory SQLite by default)
composer test

# Code formatting (Laravel Pint)
vendor/bin/pint
```

## Project Structure

```
idx-api/
├── app/
│   ├── Actions/Fortify/          # User creation, password reset, profile update (5 files)
│   ├── Billing/
│   │   └── SubscriptionCatalog.php  # Plan definitions, Stripe price IDs, teaser lead counts
│   ├── Console/Commands/          # 5 Artisan commands (GHL, GIS, Stripe, token management)
│   ├── Ghl/                       # GoHighLevel Marketplace — 44 PHP files across 16 subdirs
│   │   ├── Api/Clients/           # GhlApiClient — HTTP wrapper with auto-audit
│   │   ├── Http/                  # Install flow, GHL API proxy, embed redirect
│   │   ├── OAuth/                 # OAuth 2.0 authorize/callback/refresh + URL registration
│   │   ├── Services/              # GhlAuditService — dual-channel audit logging
│   │   ├── Sync/                  # Lead sync, subscription sync, tags, contacts, opportunities
│   │   ├── Webhooks/              # Dispatcher + handlers (install, uninstall, CRM events)
│   │   └── Widgets/               # JS embed surfaces + 3-phase middleware chain
│   ├── Http/
│   │   ├── Controllers/
│   │   │   ├── Api/               # BridgeProxyController, ImageProxyController
│   │   │   ├── Billing/           # SubscriptionCheckoutController
│   │   │   ├── Marketing/         # SalesPageController
│   │   │   ├── DashboardController.php
│   │   │   ├── DashboardApiTokenController.php
│   │   │   └── GisProxyController.php
│   │   ├── Middleware/            # DomainOrTokenAuth, VerifyGhlWebhookSignature, ProtectMonitoringDashboard
│   │   ├── Requests/              # GisProxyRequest (bbox/coordinate validation)
│   │   └── Responses/Auth/        # LoginResponse, RegisterResponse (redirect to dashboard)
│   ├── Jobs/                      # RefreshDomainListingsCacheJob, RefreshGisSourceMetadataJob, PersistGisGeoJsonBackupJob
│   ├── Livewire/Marketing/        # SalesLandingPage (pricing, billing toggle, login modal)
│   ├── Models/                    # User, Domain, ListingsCache, BridgeProxyAuditLog, GisCache, GisSourceState
│   ├── Providers/                 # AppServiceProvider, FortifyServiceProvider, TelescopeServiceProvider
│   └── Services/
│       ├── Bridge/                # BridgeHttpService, ListingsCacheService, BridgeTeaser, BridgeImageUrlRewriter, BridgeProxyAuditLogger
│       ├── GisProxyService.php    # Multi-tier ArcGIS proxy with 3-level caching + teaser
│       └── GisSourceMetadataService.php  # ArcGIS layer fingerprinting + generation invalidation
├── bootstrap/
├── config/                        # 20 config files: bridge, ghl, billing, gis, idx/idx_urls, fortify, pulse, octane, etc.
├── database/
│   ├── factories/                 # UserFactory
│   ├── migrations/                # 20 core + 9 GHL (in ghl/) = 29 total
│   └── seeders/                   # DomainSeeder, GeoWebInternalTokenSeeder, GhlConfigSeeder
├── docs/                          # 13+ docs: API guides, GHL integration, deployment, Stripe, GIS
├── public/
├── resources/
│   ├── css/app.css               # Tailwind entry
│   ├── js/app.js                 # Vite entry (Alpine/livewire)
│   └── views/                    # 13 Blade templates (auth, dashboard, marketing, livewire, ghl, leadconnector)
├── routes/
│   ├── api.php                   # Sanctum-auth: /auth/token, /leadconnector/*; domain.token: /api/v1/* (Bridge + GIS)
│   ├── web.php                   # marketing.sales, dashboard, billing.checkout
│   ├── ghl-web.php               # GHL OAuth flows, install wizard, webhooks
│   ├── ghl-widget.php            # Widget JS loader, surfaces, config, lead ingest, CORS preflight
│   └── console.php               # Scheduled tasks (hourly GHL, 15m Bridge, weekly GIS)
├── scripts/
│   ├── docker-dev.sh             # Docker dev up-watch / down
│   └── stripe-dev.sh             # Stripe webhook listen / trigger-checkout-completed
├── tests/
│   ├── Feature/                  # 10 tests: Auth, Billing, Bridge, Dashboard, Gis, Ghl, Marketing, etc.
│   ├── Unit/                     # 4 tests: Bridge rewriter, auth response, dashboard controller
│   └── TestCase.php              # Guard against non-ephemeral test databases
├── Dockerfile.idx-api            # FrankenPHP + PHP 8.5 Alpine production image (port 8000)
├── Dockerfile.idx-images         # Nginx Alpine edge image for image proxy (port 8080)
├── Dockerfile.dev                # Dev image with Xdebug, watch mode support
├── docker-compose.dev.yml        # Dev compose: idx-api-dev + idx-images-dev + cloudflared-dev (tunnel)
├── nginx.idx-images.conf         # Nginx reverse proxy config for idx-images
├── composer.json
├── package.json
├── phpunit.xml                   # In-memory SQLite, sync queue, fake Bridge/Stripe keys
└── vite.config.js                # Tailwind plugin, HMR configured for Cloudflare tunnel hosts
```

## Architecture Overview

The service has four primary subsystems:

### 1. Bridge MLS Proxy (`/api/v1/*`)

Proxies Bridge Data Output (Stellar MLS) API with domain-based or Sanctum token authentication. Key behaviors:
- **DomainOrTokenAuth middleware** resolves identity from `X-Domain-Slug` header, `?domain=` query, or referer header — falling back to Bearer token authentication
- **Teaser gating**: non-full-access requests get listings capped at 3 items (revenue lever)
- **Image URL rewriting**: Bridge photo URLs rewritten to `idx-images` public URLs before response
- **Listings cache**: 15-minute TTL collection cache per domain in PostgreSQL; skipped when `?filters=` present
- **Audit logging**: every proxied request logged with domain, token, endpoint, listing count

### 2. GoHighLevel Marketplace Integration (`/leadconnector/*`, `/oauth/*`, `/webhooks/*`, `/widget/*`)

Full GHL Marketplace app implementing OAuth 2.0 install flow, webhook ingestion, and JS embed widgets:
- **OAuth flow**: authorize -> callback -> token exchange -> location registration -> URL registration -> installation complete
- **Token management**: hourly scheduled refresh job; agency-to-location token exchange via `LocationTokenService`
- **Webhooks**: signature-verified event ingestion with handler pattern (install, uninstall, 9 CRM event types — audit-only)
- **Widgets**: API-key gated surfaces (search, lead form, showcase) with 3-phase middleware (key validate -> origin validate -> CORS append); lead ingest POST endpoint
- **Lead sync**: QuantyraLead ingestion -> GhlLeadMapping resolution -> contact + optional opportunity creation
- **Audit logging**: dual-channel (database + file at `storage/logs/ghl_audit.log`)

### 3. GIS Parcel/Geometry Proxy (`/api/v1/gis`, `/api/v1/mls/{mlsCode}/gis`)

Public ArcGIS feature server proxy for Florida county parcel data (zero MLS data). Three-tier caching with generation-based invalidation:
- **Source failover**: Florida statewide (FGIO) -> Pinellas County -> Hillsborough County -> degraded OSM fallback
- **3-level cache**: Laravel Cache (900s edge) -> PostgreSQL `gis_cache` (30-90 day origin) -> async filesystem backup
- **Generation invalidation**: Weekly metadata probe fingerprints ArcGIS layer `?f=json`; cache rows store generation at write time
- **Teaser mode** (non-`idx:full`): caps features at 40, rounds coordinates to ~11m precision, strips non-essential properties
- **Bbox guard**: maximum span cap (0.35 degrees) prevents abusive mega-queries

### 4. Billing & Subscriptions (Stripe via Laravel Cashier)

Tiered subscription model with metered API overage and 14-day trials:

| Plan | Monthly | Annual | Key Features |
|------|---------|--------|-------------|
| Pro | $39/mo | $374/yr (20% off) | 3 domains, teaser gating, basic GHL app |
| Smart | $79/mo | $758/yr | 5 domains, full GHL app, phone + email OTP |
| Ultra | $179/mo | $1,718/yr | Unlimited domains, 2M API calls/mo, developer keys |
| Mega | $449/mo | $4,310/yr | Unlimited everything, custom branding, SLA targets |

- **Catalog**: `SubscriptionCatalog` reads from `config/billing.php` with Stripe price IDs
- **Checkout**: `SubscriptionCheckoutController` creates Stripe Checkout sessions with trial days
- **Dashboard**: subscription status, API usage counters, Sanctum token management, widget install count
- **Webhook**: Stripe events update subscription status, synced to GHL via `SubscriptionSyncService`

```
┌──────────────┐        ┌──────────────┐        ┌──────────────┐
│  idx.*       │        │  idx-api.*   │        │ idx-images.* │
│  (App/Mktg)  │        │  (API/Widgets)│       │  (Nginx)     │
├──────────────┤        ├──────────────┤        ├──────────────┤
│ Livewire     │        │ Bridge Proxy │        │ Edge cache   │
│ Dashboard    │        │ GIS Proxy    │   ──▶  │ -> idx-api   │
│ Sales page   │        │ GHL OAuth    │        │ /images/*    │
│ Billing UI   │        │ GHL Webhooks │        └──────────────┘
└──────┬───────┘        │ GHL Widgets  │
       │                └──────┬───────┘
       │                       │
       ▼                       ▼
┌──────────────┐        ┌──────────────┐        ┌──────────────┐
│   Stripe     │        │  Bridge Data │        │  ArcGIS      │
│   (Cashier)  │        │  Output MLS  │        │  Parcel Src  │
└──────────────┘        └──────────────┘        └──────────────┘
```

### Key Modules

| Module | Location | Purpose |
|--------|----------|---------|
| `BridgeProxyController` | `app/Http/Controllers/Api/` | Central MLS proxy — 25+ Bridge endpoints (web, RESO OData, doc-style) |
| `GisProxyController` | `app/Http/Controllers/` | ArcGIS parcel proxy with MLS-scoped routing (`showForMls()`) |
| `GisProxyService` | `app/Services/` | Multi-tier ArcGIS proxy — 3-level cache, failover chain, teaser mode (615 lines) |
| `GisSourceMetadataService` | `app/Services/` | Layer fingerprinting via `?f=json` — generation-based cache invalidation |
| `BridgeHttpService` | `app/Services/Bridge/` | HTTP client wrapper — URL building, header forwarding, timeout management |
| `BridgeTeaser` | `app/Services/Bridge/` | JSON teaser — truncates listing arrays to N items for non-full-access |
| `BridgeImageUrlRewriter` | `app/Services/Bridge/` | Rewrites Bridge CDN URLs to `idx-images` public URLs in JSON bodies |
| `DomainOrTokenAuth` | `app/Http/Middleware/` | Dual auth: domain slug identification OR Sanctum token with ability checks |
| `GhlApiClient` | `app/Ghl/Api/Clients/` | Central HTTP wrapper for all GHL REST calls with auto-audit |
| `OAuthService` | `app/Ghl/OAuth/Services/` | GHL OAuth URL building, code exchange, token persistence |
| `TokenRefreshService` | `app/Ghl/OAuth/Services/` | Proactive token refresh before expiry |
| `LocationTokenService` | `app/Ghl/OAuth/Services/` | Agency-to-location token exchange via `/oauth/locationToken` |
| `LeadSyncService` | `app/Ghl/Sync/Services/` | Syncs Quantyra leads to GHL contacts + opportunities |
| `WebhookDispatcher` | `app/Ghl/Webhooks/Services/` | Routes GHL webhooks by type to install/uninstall/CRM handlers |
| `SubscriptionCatalog` | `app/Billing/` | Plan definitions, Stripe price IDs, teaser lead counts |
| `SalesLandingPage` | `app/Livewire/Marketing/` | Livewire marketing component with billing interval toggle |

## Development Guidelines

### Code Style
- **PHP**: Laravel Pint (PSR-12), 4-space indent, UTF-8, LF line endings (`.editorconfig`)
- **File naming**: PascalCase for classes (`BridgeProxyController.php`, `OAuthService.php`), matching PSR-4 autoloading
- **Code naming**: camelCase for methods and properties (`webUrl()`, `server_token`), PascalCase for class names
- **Models**: Use PHP 8 attributes for `$fillable` and `$hidden` (`#[Fillable([...])]`, `#[Hidden([...])]`), `casts()` method (not `$casts` property)
- **Controllers**: Constructor property promotion for DI (`public function __construct(private readonly Service $svc) {}`)
- **Services**: Explicit constructor with readonly properties; DI via Laravel container
- **Routes**: Import-style with `use` statements (no string-based controller references); named routes with `->name()`
- **Revenue impact comments**: Key business logic marked with `Revenue impact:` comments explaining monetization rationale
- **Config files**: Use `env()` directly (not `config()`) when required from other config files to avoid cache-breaking

### Import Order (PHP)
1. External classes (Illuminate, Symfony, Laravel packages)
2. App models
3. App services/controllers/middleware
4. Support classes (Closure, RuntimeException)

### Database Conventions
- Core migrations: `database/migrations/` (users, domains, listings_cache, audit, gis, cashier)
- GHL migrations: `database/migrations/ghl/` — loaded via `loadMigrationsFrom()` in `AppServiceProvider`
- Migration files: `YYYY_MM_DD_HHMMSS_description.php` format
- Models use `#[Fillable]` and `#[Hidden]` PHP 8 attributes

### Testing Patterns
- **Feature tests**: `tests/Feature/` — use `RefreshDatabase`, `Http::fake()` for external APIs, assert on JSON
- **Unit tests**: `tests/Unit/` — pure logic (URL rewriter, controller logic) without database
- **Test safety**: `TestCase::setUp()` guards against non-ephemeral databases (requires SQLite `:memory:` or `ALLOW_DESTRUCTIVE_TEST_DB=true`)
- **phpunit.xml**: in-memory SQLite, sync queue, fake Bridge/Stripe keys, PULSE/TELESCOPE disabled
- **Factories**: `User::factory()->create()`; direct model creation for seed data
- **Config setup**: Tests set config values in `setUp()` (bridge host, dataset, tokens)

### Scheduled Tasks (routes/console.php)
| Task | Schedule | Queue |
|------|----------|-------|
| `ghl:refresh-tokens` | Hourly, no overlap | default |
| `bridge-listings-cache-refresh` | Every 15 min per active domain, no overlap | default |
| `gis-source-metadata-probe` | Monday 6:30am, no overlap | `config('gis.queue')` |

## Environment Variables

### Core

| Variable | Required | Description |
|----------|----------|-------------|
| `APP_KEY` | Yes | Laravel encryption key |
| `APP_URL` | Yes | Base URL (becomes IDX_API_PUBLIC_URL by default) |
| `DB_CONNECTION` | Yes | `sqlite` (local) or `pgsql` (prod) |

### Public URLs

| Variable | Required | Description |
|----------|----------|-------------|
| `IDX_PLATFORM_URL` | Yes | Public app URL (idx.quantyralabs.cc) |
| `IDX_API_PUBLIC_URL` | Yes | Public API URL (defaults to APP_URL) |
| `IDX_IMAGES_PUBLIC_URL` | Yes | Public image proxy URL (idx-images.quantyralabs.cc) |
| `IDX_PLATFORM_HOSTS` | Dev | Comma-separated allowed hosts for platform |
| `IDX_API_HOSTS` | Dev | Comma-separated allowed hosts for API |

### Bridge MLS

| Variable | Required | Description |
|----------|----------|-------------|
| `BRIDGE_API_KEY` | Yes | Bridge Data Output API key |
| `BRIDGE_HOST` | Yes | Bridge API base URL (default: api.bridgedataoutput.com) |
| `BRIDGE_DATASET` | No | MLS dataset (default: `stellar`) |
| `BRIDGE_PATH_PREFIX` | No | e.g. `api/v2` |
| `BRIDGE_RESO_ROOT` | No | e.g. `reso/odata` |
| `BRIDGE_LISTING_PHOTO_PATH` | No | Path template for photos |
| `BRIDGE_IMAGE_REWRITE_HOSTS` | No | Extra hostnames for URL rewriting |
| `BRIDGE_TIMEOUT` | No | HTTP timeout (default: 30) |
| `LISTINGS_CACHE_TTL` | No | Cache TTL in seconds (default: 900) |
| `IMAGE_CACHE_PATH` | No | Image storage root (Docker: /var/cache/geoidx/images) |
| `IMAGE_CACHE_TTL` | No | Origin re-fetch TTL (default: 86400) |

### GHL Marketplace

| Variable | Required | Description |
|----------|----------|-------------|
| `GHL_CLIENT_ID` | Yes | GHL Marketplace app client ID |
| `GHL_CLIENT_SECRET` | Yes | GHL Marketplace app client secret |
| `GHL_REDIRECT_URI` | Yes | OAuth callback URL |
| `GHL_WEBHOOK_REQUIRE_SIGNATURE` | No | HMAC verification toggle (default: true) |
| `GHL_WEBHOOK_SECRET` | No | Webhook signing secret |
| `GHL_ADMIN_REFRESH_TOKEN` | No | Admin token for manual refresh endpoint |
| `GHL_SCOPES` | No | OAuth scopes (see `.env.example` for full list) |
| `GHL_AUDIT_LOG_ENABLED` | No | File audit logging toggle |
| `GHL_AUDIT_LOG_PATH` | No | Audit log file path (default: storage/logs/ghl_audit.log) |

### GIS Parcel Proxy

| Variable | Required | Description |
|----------|----------|-------------|
| `GIS_EDGE_CACHE_TTL` | No | Laravel Cache edge TTL (default: 900) |
| `GIS_ORIGIN_MAX_DAYS_PRIMARY` | No | Postgres origin max age for statewide (default: 90) |
| `GIS_ORIGIN_MAX_DAYS_COUNTY` | No | Postgres origin max age for county (default: 30) |
| `GIS_METADATA_TIMEOUT` | No | Metadata probe HTTP timeout (default: 12) |
| `GIS_QUEUE` | No | Queue for GIS jobs (default: default) |
| `GIS_QUEUE_BACKUP_WRITES` | No | Async filesystem backup (default: true) |
| `GIS_TEASER_MAX_FEATURES` | No | Feature cap for non-full-access (default: 40) |
| `GIS_TEASER_COORD_DECIMALS` | No | Coordinate precision for teaser (default: 4, ~11m) |
| `GIS_MAX_BBOX_SPAN_DEG` | No | Max bbox span to prevent abuse (default: 0.35) |
| `GIS_FLORIDA_MLS_CODES` | No | Comma-separated MLS codes (default: stellar) |

### Stripe / Billing

| Variable | Required | Description |
|----------|----------|-------------|
| `STRIPE_KEY` | Yes (billing) | Stripe publishable key |
| `STRIPE_SECRET` | Yes (billing) | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | Yes (billing) | Webhook signing secret |
| `STRIPE_PRICE_IDX_*` | Yes (billing) | Price IDs for Pro/Smart/Ultra/Mega (monthly + yearly) |
| `STRIPE_PRICE_IDX_API_OVERAGE_METERED` | No | Metered overage price ID |
| `CASHIER_CURRENCY` | Recommended | ISO currency (e.g. `usd`) |

### Internal / Ops

| Variable | Required | Description |
|----------|----------|-------------|
| `IDX_API_INTERNAL_TOKEN` | No | Sanctum PAT for geo-web server-to-server |
| `DEBUGBAR_ENABLED` | Dev | Debug bar toggle |
| `TELESCOPE_ENABLED` | Dev | Telescope toggle |
| `PULSE_ENABLED` | Dev | Pulse metrics toggle |
| `MONITORING_DASHBOARD_USERNAME` | Prod | HTTP Basic Auth for Telescope/Pulse |
| `MONITORING_DASHBOARD_PASSWORD` | prod | HTTP Basic Auth password |
| `CLOUDFLARED_TOKEN` | Dev | Cloudflare tunnel token for dev |

## Available Commands

| Command | Description |
|---------|-------------|
| `composer dev` | Start server + queue + pail logs + Vite in parallel |
| `composer test` | Run full test suite (clears config first) |
| `composer setup` | Fresh install: composer, env, key, migrate, npm |
| `php artisan octane:start` | Start Octane with FrankenPHP (production) |
| `php artisan serve` | Start PHP dev server |
| `vendor/bin/pint` | Format code with Laravel Pint |
| `./scripts/docker-dev.sh up-watch` | Docker dev with hot reload (Compose watch) |
| `./scripts/docker-dev.sh down` | Stop Docker dev containers |
| `./scripts/stripe-dev.sh listen` | Forward Stripe webhooks locally |
| `./scripts/stripe-dev.sh trigger-checkout-completed` | Fire test checkout event |
| `php artisan ghl:refresh-tokens` | Manually trigger GHL token refresh |
| `php artisan billing:provision-stripe-catalog` | Create Stripe products + prices for all plans |
| `php artisan gis:probe-sources` | Probe ArcGIS layer metadata (inline or queued) |
| `php artisan gis:clear-cache` | Clear GIS cache (all or by source) |
| `php artisan idx-api:issue-geo-web-token` | Create/rotate geo-web-internal Sanctum token |

## Docker Deployment

### Production

Two images built from project root:

| Image | Dockerfile | Base | Port | Entry point |
|-------|-----------|------|------|-------------|
| idx-api | `Dockerfile.idx-api` | FrankenPHP + PHP 8.5 Alpine | 8000 | `php artisan octane:start --server=frankenphp` |
| idx-images | `Dockerfile.idx-images` | Nginx 1.27 Alpine | 8080 | Nginx reverse-proxy to idx-api:8000 |

```bash
docker build -f Dockerfile.idx-api -t quantyra/idx-api:latest .
docker build -f Dockerfile.idx-images -t quantyra/idx-images:latest .
```

### Development

```bash
./scripts/docker-dev.sh up-watch   # idx-api-dev + idx-images-dev + cloudflared-dev
./scripts/docker-dev.sh down       # Stop all dev services
```

Dev compose (`docker-compose.dev.yml`) includes Xdebug support (`XDEBUG_MODE`, `client_host=host.docker.internal`), file watching via Compose `develop.watch`, and Cloudflare tunnel for public HTTPS access.

## Testing

- **16 test files** across Feature (10) and Unit (4) suites
- Tests use in-memory SQLite with sync queue driver
- External APIs faked via `Http::fake()` (Bridge, Stripe)
- `TestCase::setUp()` enforces ephemeral database safety guard
- Coverage includes Bridge proxy security, image proxy headers, GIS probe/proxy, GHL marketplace flow, billing checkout gates, dashboard API tokens, marketing/sales pages

## Additional Resources

- @docs/INDEX.md — Full documentation index (API guides, GHL integration, deployment, Stripe billing)
- @docs/idx-api-bridge-proxy.md — Bridge proxy architecture, auth flow, cache strategy, image rewrite
- @docs/ghl-marketplace-integration.md — GHL OAuth, widgets, webhooks, sync implementation details
- @docs/stripe-laravel-cashier.md — Stripe subscription catalog, webhook handling, local development
- @docs/ghl-environment-variables.md — All GHL_, IDX_, and related env vars
- @docs/ghl-database-schema.md — PostgreSQL table reference for GHL subsystem
- @docs/ghl-api-routes-reference.md — Curl examples and response notes for GHL API
- @docs/bridge-api-documentation.md — Bridge Data Output upstream API reference
- @docs/gis-api.md — GIS parcel/geometry proxy documentation
- @README.md — Project overview and Docker build instructions

## Skill Usage Guide

When working on tasks involving these technologies, invoke the corresponding skill:

| Skill | Invoke When |
|-------|-------------|
| livewire | Manages Livewire 3 reactive components and Blade integration |
| postgresql | Handles PostgreSQL schema, migrations, and query patterns |
| frontend-design | Applies UI design with Livewire, Blade, Tailwind CSS 4, and Alpine.js |
| laravel | Manages Laravel 13 routing, ORM, queues, and service providers |
| stripe | Manages Stripe billing, Cashier subscriptions, and webhook handling |
| docker | Configures Docker multi-stage builds, FrankenPHP, and Compose workflows |
| php | Enforces PHP 8.3+ patterns, strict typing, and constructor promotion |
| tailwind | Applies Tailwind CSS 4 styling and utility patterns |
| scoping-feature-work | Breaks features into MVP slices and acceptance criteria |
| mapping-user-journeys | Maps in-app journeys and identifies friction points in code |
| improving-activation-flow | Optimizes activation steps and time-to-value milestones |
| crafting-empty-states | Creates empty states and onboarding affordances |
| designing-onboarding-paths | Designs onboarding paths, checklists, and first-run UI |
| prioritizing-roadmap-bets | Ranks initiatives using impact, effort, and risk signals |
| orchestrating-feature-adoption | Plans feature discovery, nudges, and adoption flows |
| writing-release-notes | Drafts release notes tied to shipped features |
| triaging-user-feedback | Routes feedback into backlog and quick wins |
| clarifying-market-fit | Aligns ICP, positioning, and value narrative for on-page messaging |
| running-product-experiments | Sets up product experiments and rollout checks |
| structuring-offer-ladders | Frames plan tiers, value ladders, and upgrade logic |
| generating-growth-hypotheses | Generates channel experiments and growth loops |
| instrumenting-product-metrics | Defines product events, funnels, and activation metrics |
| framing-release-stories | Builds launch narratives, assets, and rollout checklists |
| crafting-page-messaging | Writes conversion-focused messaging for pages and key CTAs |
| planning-editorial-arcs | Defines content themes, briefs, and editorial cadence |
| designing-lifecycle-messages | Designs onboarding and lifecycle email sequences |
| designing-inapp-guidance | Builds tooltips, tours, and contextual guidance |
| tightening-brand-voice | Refines copy for clarity, tone, and consistency |
| embedding-decision-cues | Applies behavioral cues that improve conversion decisions |
| accelerating-first-run | Improves onboarding sequence and time-to-value |
| streamlining-signup-steps | Reduces friction in signup and trial activation |
| orchestrating-social-rhythm | Plans social content beats and distribution rhythm |
| reducing-form-falloff | Improves lead capture forms to reduce drop-off |
| tuning-landing-journeys | Improves landing page flow, hierarchy, and conversion paths |
| refining-prompt-surfaces | Optimizes banners, modals, and in-app prompts |
| strengthening-upgrade-moments | Improves upgrade prompts and paywall messaging |
| mapping-conversion-events | Defines funnel events, tracking, and success signals |
| designing-variation-tests | Plans A/B experiments and measurement plans |
| engineering-referral-loops | Designs referral or partner loop mechanics |
| building-acquisition-tools | Designs lead magnets or free tools for acquisition |
| adding-structured-signals | Adds structured data for rich results |
| scaling-template-pages | Builds scalable, template-driven search pages |
| calibrating-paid-campaigns | Aligns paid acquisition with landing pages and pixels |
| inspecting-search-coverage | Audits technical and on-page search coverage |
| building-compare-hubs | Creates comparison and alternative pages for discovery |

## Laravel Boost Guidelines

### Foundation Rules

The Laravel Boost guidelines are specifically curated by Laravel maintainers for this application and should be followed closely.

#### Foundational Context

This application is a Laravel application and its main Laravel ecosystem package versions are:

- php - 8.5
- laravel/cashier (CASHIER) - v16
- laravel/framework (LARAVEL) - v13
- laravel/octane (OCTANE) - v2
- laravel/prompts (PROMPTS) - v0
- laravel/sanctum (SANCTUM) - v4
- livewire/livewire (LIVEWIRE) - v3
- laravel/boost (BOOST) - v2
- laravel/mcp (MCP) - v0
- laravel/pail (PAIL) - v1
- laravel/pint (PINT) - v1
- phpunit/phpunit (PHPUNIT) - v12

#### Skills Activation

Activate the relevant domain skill whenever working in that domain:

- cashier-stripe-development for Laravel Cashier Stripe integration and billing workflows.
- laravel-best-practices for Laravel PHP code changes, reviews, and refactors.
- livewire-development for any Livewire-specific component or reactivity work.

#### Conventions

- Follow existing code conventions and check sibling files for established patterns.
- Use descriptive variable and method names.
- Reuse existing components before creating new ones.

#### Verification Scripts

- Do not create verification scripts or use tinker where tests already cover the behavior.

#### Application Structure and Architecture

- Stick to existing directory structure; do not add new top-level folders without approval.
- Do not change dependencies without approval.

#### Frontend Bundling

- If frontend changes are not visible, run `npm run build`, `npm run dev`, or `composer run dev`.

#### Documentation Files

- Only create documentation files when explicitly requested.

#### Replies

- Keep explanations concise and focused on what matters.

### Boost Rules

#### Laravel Boost Tools

- Prefer Laravel Boost MCP tools over manual shell/file-read alternatives when applicable.
- Use database-query for read-only database queries.
- Use database-schema before writing migrations or models.
- Use get-absolute-url before sharing project URLs.
- Use browser-logs for recent browser errors/exceptions.

#### Searching Documentation (Important)

- Use search-docs before making code changes.
- Pass a packages array when package scope is known.
- Use broad, topic-focused queries and avoid package names in query strings.

Search syntax:

1. Word queries use AND logic with stemming (`rate limit`).
2. Quoted phrases match exact adjacency (`"infinite scroll"`).
3. Combine words and phrases (`middleware "rate limit"`).
4. Use multiple queries for OR logic (`queries=["authentication", "middleware"]`).

#### Artisan

- Run Artisan directly via CLI (`php artisan route:list`, `php artisan list`, `php artisan [command] --help`).
- Use route list filters like `--method`, `--name`, `--path`, `--except-vendor`, `--only-vendor`.
- Read config values with `php artisan config:show key`.
- Read environment variables from `.env`.

#### Tinker

- Prefer tests and existing Artisan commands over tinker.
- Use single quotes for shell safety:
  - `php artisan tinker --execute 'Your::code();'`
  - `php artisan tinker --execute 'User::where("active", true)->count();'`

### PHP Rules

- Always use curly braces for control structures.
- Use PHP 8 constructor property promotion where appropriate.
- Use explicit parameter and return types.
- Use TitleCase enum keys.
- Prefer PHPDoc over inline comments except for unusually complex logic.
- Use array-shape type definitions in PHPDoc when useful.

### Deployment Rules

- Laravel Cloud is the preferred fast path for deploying and scaling production Laravel applications.

### Test Enforcement Rules

- Every change must be programmatically tested (new or updated tests).
- Run the minimum relevant tests using `php artisan test --compact`.

### Laravel Core Rules

#### Do Things the Laravel Way

- Use `php artisan make:*` commands for framework artifacts.
- Use `php artisan make:class` for generic PHP classes.
- Pass `--no-interaction` to Artisan generation commands.

#### Model Creation

- When creating models, also create useful factories and seeders as needed.

#### APIs and Eloquent Resources

- Default to API Resources and API versioning unless existing project conventions differ.

#### URL Generation

- Prefer named routes and the route() helper.
- `APP_URL` is the canonical base URL for absolute URLs in all environments. Never hardcode environment domains (e.g. `idx.quantyralabs.cc`) in application code.
- When you need host-aware absolute links/redirects, derive them from `APP_URL` (or config values sourced from it). Use relative routes (`route(..., false)`) when host coupling should be avoided.

#### Testing

- Use factories when creating models in tests.
- Follow existing Faker style (`$this->faker` or `fake()`).
- Use `php artisan make:test` (feature by default, `--unit` when needed).

#### Vite Error

- For Vite manifest errors, run `npm run build` or `npm run dev` / `composer run dev`.

### Octane Rules

- Octane reuses bootstrapped state across requests; singleton state can persist.
- Use scoped bindings where appropriate.
- Do not inject container/request/config directly into singleton constructors; use resolver closures.
- Avoid appending to static properties across requests.

### Livewire Rules

- Build reactive interfaces in PHP with Livewire, optionally Alpine for client interactions.
- Keep state server-side and validate/authorize in actions as with HTTP requests.

### Pint Rules

- If PHP files change, run `vendor/bin/pint --dirty --format agent`.
- Do not use `vendor/bin/pint --test --format agent`.

### PHPUnit Rules

- Write tests as PHPUnit classes.
- Convert Pest tests to PHPUnit where encountered.
- Run singular relevant tests after updates.
- Ask whether to run full suite after related tests pass.
- Cover happy paths, failure paths, and edge cases.
- Do not remove test files without approval.

#### Running Tests

- All tests: `php artisan test --compact`
- File: `php artisan test --compact tests/Feature/ExampleTest.php`
- Filter: `php artisan test --compact --filter=testName`

===

<laravel-boost-guidelines>
=== foundation rules ===

# Laravel Boost Guidelines

The Laravel Boost guidelines are specifically curated by Laravel maintainers for this application. These guidelines should be followed closely to ensure the best experience when building Laravel applications.

## Foundational Context

This application is a Laravel application and its main Laravel ecosystems package & versions are below. You are an expert with them all. Ensure you abide by these specific packages & versions.

- php - 8.5
- laravel/cashier (CASHIER) - v16
- laravel/fortify (FORTIFY) - v1
- laravel/framework (LARAVEL) - v13
- laravel/octane (OCTANE) - v2
- laravel/prompts (PROMPTS) - v0
- laravel/pulse (PULSE) - v1
- laravel/sanctum (SANCTUM) - v4
- laravel/telescope (TELESCOPE) - v5
- livewire/livewire (LIVEWIRE) - v4
- livewire/volt (VOLT) - v1
- laravel/boost (BOOST) - v2
- laravel/mcp (MCP) - v0
- laravel/pail (PAIL) - v1
- laravel/pint (PINT) - v1
- phpunit/phpunit (PHPUNIT) - v12
- tailwindcss (TAILWINDCSS) - v4

## Skills Activation

This project has domain-specific skills available. You MUST activate the relevant skill whenever you work in that domain—don't wait until you're stuck.

- `cashier-stripe-development` — Handles Laravel Cashier Stripe integration including subscriptions, webhooks, Stripe Checkout, invoices, charges, refunds, trials, coupons, metered billing, and payment failure handling. Triggered when a user mentions Cashier, Billable, IncompletePayment, stripe_id, newSubscription, Stripe subscriptions, or billing. Also applies when setting up webhooks, handling SCA/3DS payment failures, testing with Stripe test cards, or troubleshooting incomplete subscriptions, CSRF webhook errors, or migration publish issues.
- `fortify-development` — ACTIVATE when the user works on authentication in Laravel. This includes login, registration, password reset, email verification, two-factor authentication (2FA/TOTP/QR codes/recovery codes), profile updates, password confirmation, or any auth-related routes and controllers. Activate when the user mentions Fortify, auth, authentication, login, register, signup, forgot password, verify email, 2FA, or references app/Actions/Fortify/, CreateNewUser, UpdateUserProfileInformation, FortifyServiceProvider, config/fortify.php, or auth guards. Fortify is the frontend-agnostic authentication backend for Laravel that registers all auth routes and controllers. Also activate when building SPA or headless authentication, customizing login redirects, overriding response contracts like LoginResponse, or configuring login throttling. Do NOT activate for Laravel Passport (OAuth2 API tokens), Socialite (OAuth social login), or non-auth Laravel features.
- `laravel-best-practices` — Apply this skill whenever writing, reviewing, or refactoring Laravel PHP code. This includes creating or modifying controllers, models, migrations, form requests, policies, jobs, scheduled commands, service classes, and Eloquent queries. Triggers for N+1 and query performance issues, caching strategies, authorization and security patterns, validation, error handling, queue and job configuration, route definitions, and architectural decisions. Also use for Laravel code reviews and refactoring existing Laravel code to follow best practices. Covers any task involving Laravel backend PHP code patterns.
- `pulse-development` — Handles Laravel Pulse setup, configuration, and custom card development. Activates when installing Pulse; configuring the dashboard or authorization gate; setting up recorders and filtering; building custom Livewire cards; optimizing with Redis ingest or sampling; or when the user mentions /pulse, pulse:check, pulse:work, Pulse::record(), or application monitoring.
- `livewire-development` — Use for any task or question involving Livewire. Activate if user mentions Livewire, wire: directives, or Livewire-specific concepts like wire:model, wire:click, wire:sort, or islands, invoke this skill. Covers building new components, debugging reactivity issues, real-time form validation, drag-and-drop, loading states, migrating from Livewire 3 to 4, converting component formats (SFC/MFC/class-based), and performance optimization. Do not use for non-Livewire reactive UI (React, Vue, Alpine-only, Inertia.js) or standard Laravel forms without Livewire.
- `volt-development` — Develops single-file Livewire components with Volt. Activates when creating Volt components, converting Livewire to Volt, working with @volt directive, functional or class-based Volt APIs; or when the user mentions Volt, single-file components, functional Livewire, or inline component logic in Blade files.
- `tailwindcss-development` — Always invoke when the user's message includes 'tailwind' in any form. Also invoke for: building responsive grid layouts (multi-column card grids, product grids), flex/grid page structures (dashboards with sidebars, fixed topbars, mobile-toggle navs), styling UI components (cards, tables, navbars, pricing sections, forms, inputs, badges), adding dark mode variants, fixing spacing or typography, and Tailwind v3/v4 work. The core use case: writing or fixing Tailwind utility classes in HTML templates (Blade, JSX, Vue). Skip for backend PHP logic, database queries, API routes, JavaScript with no HTML/CSS component, CSS file audits, build tool configuration, and vanilla CSS.

## Conventions

- You must follow all existing code conventions used in this application. When creating or editing a file, check sibling files for the correct structure, approach, and naming.
- Use descriptive names for variables and methods. For example, `isRegisteredForDiscounts`, not `discount()`.
- Check for existing components to reuse before writing a new one.

## Verification Scripts

- Do not create verification scripts or tinker when tests cover that functionality and prove they work. Unit and feature tests are more important.

## Application Structure & Architecture

- Stick to existing directory structure; don't create new base folders without approval.
- Do not change the application's dependencies without approval.

## Frontend Bundling

- If the user doesn't see a frontend change reflected in the UI, it could mean they need to run `npm run build`, `npm run dev`, or `composer run dev`. Ask them.

## Documentation Files

- You must only create documentation files if explicitly requested by the user.

## Replies

- Be concise in your explanations - focus on what's important rather than explaining obvious details.

=== boost rules ===

# Laravel Boost

## Tools

- Laravel Boost is an MCP server with tools designed specifically for this application. Prefer Boost tools over manual alternatives like shell commands or file reads.
- Use `database-query` to run read-only queries against the database instead of writing raw SQL in tinker.
- Use `database-schema` to inspect table structure before writing migrations or models.
- Use `get-absolute-url` to resolve the correct scheme, domain, and port for project URLs. Always use this before sharing a URL with the user.
- Use `browser-logs` to read browser logs, errors, and exceptions. Only recent logs are useful, ignore old entries.

## Searching Documentation (IMPORTANT)

- Always use `search-docs` before making code changes. Do not skip this step. It returns version-specific docs based on installed packages automatically.
- Pass a `packages` array to scope results when you know which packages are relevant.
- Use multiple broad, topic-based queries: `['rate limiting', 'routing rate limiting', 'routing']`. Expect the most relevant results first.
- Do not add package names to queries because package info is already shared. Use `test resource table`, not `filament 4 test resource table`.

### Search Syntax

1. Use words for auto-stemmed AND logic: `rate limit` matches both "rate" AND "limit".
2. Use `"quoted phrases"` for exact position matching: `"infinite scroll"` requires adjacent words in order.
3. Combine words and phrases for mixed queries: `middleware "rate limit"`.
4. Use multiple queries for OR logic: `queries=["authentication", "middleware"]`.

## Artisan

- Run Artisan commands directly via the command line (e.g., `php artisan route:list`). Use `php artisan list` to discover available commands and `php artisan [command] --help` to check parameters.
- Inspect routes with `php artisan route:list`. Filter with: `--method=GET`, `--name=users`, `--path=api`, `--except-vendor`, `--only-vendor`.
- Read configuration values using dot notation: `php artisan config:show app.name`, `php artisan config:show database.default`. Or read config files directly from the `config/` directory.
- To check environment variables, read the `.env` file directly.

## Tinker

- Execute PHP in app context for debugging and testing code. Do not create models without user approval, prefer tests with factories instead. Prefer existing Artisan commands over custom tinker code.
- Always use single quotes to prevent shell expansion: `php artisan tinker --execute 'Your::code();'`
  - Double quotes for PHP strings inside: `php artisan tinker --execute 'User::where("active", true)->count();'`

=== php rules ===

# PHP

- Always use curly braces for control structures, even for single-line bodies.
- Use PHP 8 constructor property promotion: `public function __construct(public GitHub $github) { }`. Do not leave empty zero-parameter `__construct()` methods unless the constructor is private.
- Use explicit return type declarations and type hints for all method parameters: `function isAccessible(User $user, ?string $path = null): bool`
- Use TitleCase for Enum keys: `FavoritePerson`, `BestLake`, `Monthly`.
- Prefer PHPDoc blocks over inline comments. Only add inline comments for exceptionally complex logic.
- Use array shape type definitions in PHPDoc blocks.

=== deployments rules ===

# Deployment

- Laravel can be deployed using [Laravel Cloud](https://cloud.laravel.com/), which is the fastest way to deploy and scale production Laravel applications.

=== tests rules ===

# Test Enforcement

- Every change must be programmatically tested. Write a new test or update an existing test, then run the affected tests to make sure they pass.
- Run the minimum number of tests needed to ensure code quality and speed. Use `php artisan test --compact` with a specific filename or filter.

=== laravel/core rules ===

# Do Things the Laravel Way

- Use `php artisan make:` commands to create new files (i.e. migrations, controllers, models, etc.). You can list available Artisan commands using `php artisan list` and check their parameters with `php artisan [command] --help`.
- If you're creating a generic PHP class, use `php artisan make:class`.
- Pass `--no-interaction` to all Artisan commands to ensure they work without user input. You should also pass the correct `--options` to ensure correct behavior.

### Model Creation

- When creating new models, create useful factories and seeders for them too. Ask the user if they need any other things, using `php artisan make:model --help` to check the available options.

## APIs & Eloquent Resources

- For APIs, default to using Eloquent API Resources and API versioning unless existing API routes do not, then you should follow existing application convention.

## URL Generation

- When generating links to other pages, prefer named routes and the `route()` function.

## Testing

- When creating models for tests, use the factories for the models. Check if the factory has custom states that can be used before manually setting up the model.
- Faker: Use methods such as `$this->faker->word()` or `fake()->randomDigit()`. Follow existing conventions whether to use `$this->faker` or `fake()`.
- When creating tests, make use of `php artisan make:test [options] {name}` to create a feature test, and pass `--unit` to create a unit test. Most tests should be feature tests.

## Vite Error

- If you receive an "Illuminate\Foundation\ViteException: Unable to locate file in Vite manifest" error, you can run `npm run build` or ask the user to run `npm run dev` or `composer run dev`.

=== octane/core rules ===

# Octane

- Octane boots the application once and reuses it across requests, so singletons persist between requests.
- The Laravel container's `scoped` method may be used as a safe alternative to `singleton`.
- Never inject the container, request, or config repository into a singleton's constructor; use a resolver closure or `bind()` instead:

```php
// Bad
$this->app->singleton(Service::class, fn (Application $app) => new Service($app['request']));

// Good
$this->app->singleton(Service::class, fn () => new Service(fn () => request()));
```

- Never append to static properties, as they accumulate in memory across requests.

=== livewire/core rules ===

# Livewire

- Livewire allow to build dynamic, reactive interfaces in PHP without writing JavaScript.
- You can use Alpine.js for client-side interactions instead of JavaScript frameworks.
- Keep state server-side so the UI reflects it. Validate and authorize in actions as you would in HTTP requests.

=== volt/core rules ===

# Livewire Volt

- Single-file Livewire components: PHP logic and Blade templates in one file.
- Always check existing Volt components to determine functional vs class-based style.
- IMPORTANT: Always use `search-docs` tool for version-specific Volt documentation and updated code examples.
- IMPORTANT: Activate `volt-development` every time you're working with a Volt or single-file component-related task.

=== pint/core rules ===

# Laravel Pint Code Formatter

- If you have modified any PHP files, you must run `vendor/bin/pint --dirty --format agent` before finalizing changes to ensure your code matches the project's expected style.
- Do not run `vendor/bin/pint --test --format agent`, simply run `vendor/bin/pint --format agent` to fix any formatting issues.

=== phpunit/core rules ===

# PHPUnit

- This application uses PHPUnit for testing. All tests must be written as PHPUnit classes. Use `php artisan make:test --phpunit {name}` to create a new test.
- If you see a test using "Pest", convert it to PHPUnit.
- Every time a test has been updated, run that singular test.
- When the tests relating to your feature are passing, ask the user if they would like to also run the entire test suite to make sure everything is still passing.
- Tests should cover all happy paths, failure paths, and edge cases.
- You must not remove any tests or test files from the tests directory without approval. These are not temporary or helper files; these are core to the application.

## Running Tests

- Run the minimal number of tests, using an appropriate filter, before finalizing.
- To run all tests: `php artisan test --compact`.
- To run all tests in a file: `php artisan test --compact tests/Feature/ExampleTest.php`.
- To filter on a particular test name: `php artisan test --compact --filter=testName` (recommended after making a change to a related file).

</laravel-boost-guidelines>
