# Quantyra IDX API

Laravel 13 + Octane service powering Quantyra's Bridge MLS proxy, GIS parcel/geometry proxy, authenticated user dashboard (domains, API keys, MLS feed scope), and secured image proxy delivery. The service sits between real estate MLS data (Bridge Data Output / Stellar MLS), public ArcGIS parcel sources, and customer tooling. Three public surfaces: **idx.quantyralabs.cc** (app/marketing), **idx-api.quantyralabs.cc** (API), **idx-images.quantyralabs.cc** (image proxy).

## Tech Stack

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| Runtime | PHP | 8.5 | Server language |
| Web Server | FrankenPHP (via Laravel Octane) | 2.x | High-performance concurrent request handling |
| Framework | Laravel | 13.x | Application skeleton, routing, ORM, queues |
| Language | PHP | 8.5 | Strong typing, constructor promotion, PHP attributes for model fillable/hidden |
| Database | PostgreSQL | - | Eloquent ORM; development, staging (shared), and production (dedicated) all use `pgsql` |
| Frontend | Livewire + Blade + Tailwind CSS 4 | 4.x / 4.x | Server-rendered marketing home, user dashboard, Filament dashboard shell |
| Build | Vite | 8.x | CSS/JS bundling with Tailwind plugin |
| Auth | Laravel Sanctum + Fortify | 4.x / 1.36.x | API tokens; login + **invite-only** registration (no public self-signup), password reset, 2FA |
| Observability | Pulse + Telescope + Debugbar | 1.x / 5.x / 4.x | Production metrics, local debugging, rate-limited behind HTTP Basic Auth |

## Quick Start

```bash
# Prerequisites: PHP 8.5+, Composer, Node 20+, PostgreSQL (local development database)

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

# Run tests (PostgreSQL from `.env`; see README + tests/bootstrap.php; PostGIS requires Postgres)
composer test

# Code formatting (Laravel Pint)
vendor/bin/pint
```

## Project Structure

```
idx-api/
├── app/
│   ├── Actions/Fortify/          # User creation, password reset, profile update
│   ├── Console/Commands/         # GIS, token management, MLS utilities
│   ├── Filament/                 # User dashboard (Filament v4)
│   ├── Http/
│   │   ├── Controllers/
│   │   │   ├── Api/              # BridgeProxyController, ImageProxyController
│   │   │   ├── Marketing/        # SalesPageController → marketing home
│   │   │   ├── Dashboard*.php    # User dashboard + domains + MLS scope + API tokens
│   │   │   └── GisProxyController.php
│   │   ├── Middleware/           # DomainOrTokenAuth, mls.access, ProtectMonitoringDashboard
│   │   └── Requests/
│   ├── Jobs/                     # Listings cache refresh, GIS metadata, backups
│   ├── Models/                   # User, Domain, ListingsCache, Bridge proxy audit, GIS cache
│   ├── Providers/
│   └── Services/                 # Bridge/, Gis*
├── config/                       # bridge, gis, mls, idx/idx_urls, fortify, pulse, octane, etc.
├── database/migrations/
├── docs/
├── routes/                       # web.php (platform + API hosts), api.php (v1 Bridge + GIS)
├── tests/
└── Dockerfile.* / docker-compose.dev.yml
```

## Architecture Overview

The service has three primary subsystems:

### 1. Bridge MLS Proxy (`/api/v1/*`)

Proxies Bridge Data Output with domain-based or Sanctum token authentication. Key behaviors:
- **DomainOrTokenAuth** + **mls.access** resolve identity and feed access
- **Teaser gating**: non-full-access requests cap listings (revenue lever)
- **Image URL rewriting**: Bridge photo URLs rewritten to `idx-images` public URLs
- **Listings cache**: TTL collection cache per domain; purge/housekeeping jobs for stale rows
- **Audit logging**: proxied requests logged where enabled

### 2. GIS Parcel/Geometry Proxy (`/api/v1/gis`, `/api/v1/mls/{mlsCode}/gis`)

Public ArcGIS feature server proxy for Florida parcel data. Three-tier caching with generation-based invalidation, source failover, teaser mode for non-full tokens, and bbox limits.

### 3. Platform & user dashboard

- **Marketing home** (`/`) on platform hosts — static Blade intro and links to login / dashboard
- **User dashboard** (`/dashboard` and Filament `/filament-dashboard`) — domains, TXT verification, MLS datasets per domain, Sanctum API tokens

```
┌──────────────┐        ┌──────────────┐        ┌──────────────┐
│  idx.*       │        │  idx-api.*   │        │ idx-images.* │
│  (App/Mktg)  │        │  (API)       │        │  (Nginx)     │
├──────────────┤        ├──────────────┤        ├──────────────┤
│ User dashboard │        │ Bridge Proxy │        │ Edge cache   │
│ (Blade +       │        │ GIS Proxy    │   ──▶  │ -> idx-api   │
│  Filament)     │        │              │        │ /images/*    │
└──────────────┘        └──────┬───────┘        └──────────────┘
                               │
                               ▼
                       ┌──────────────┐        ┌──────────────┐
                       │  Bridge MLS  │        │  ArcGIS      │
                       │              │        │  Parcel Src  │
                       └──────────────┘        └──────────────┘
```

### Key modules

| Module | Location | Purpose |
|--------|----------|---------|
| `BridgeProxyController` | `app/Http/Controllers/Api/` | MLS proxy — web, RESO OData, doc-style endpoints |
| `GisProxyController` | `app/Http/Controllers/` | ArcGIS parcel proxy, MLS-scoped routing |
| `GisProxyService` | `app/Services/` | Multi-tier GIS proxy, cache, failover, teaser |
| `BridgeHttpService` | `app/Services/Bridge/` | HTTP client, URL building, timeouts |
| `DomainOrTokenAuth` | `app/Http/Middleware/` | Domain slug and/or Sanctum token auth |

## Development Guidelines

### Code Style
- **PHP**: Laravel Pint (PSR-12), 4-space indent, UTF-8, LF line endings (`.editorconfig`)
- **File naming**: PascalCase for classes (`BridgeProxyController.php`, `GisProxyService.php`), matching PSR-4 autoloading
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
- Core migrations: `database/migrations/` (users, domains, listings_cache, audit, gis, MLS registry)
- Migration files: `YYYY_MM_DD_HHMMSS_description.php` format
- Models use `#[Fillable]` and `#[Hidden]` PHP 8 attributes where applicable

### Testing Patterns
- **Feature tests**: `tests/Feature/` — use `RefreshDatabase`, `Http::fake()` for external APIs, assert on JSON
- **Unit tests**: `tests/Unit/` — pure logic without database where possible
- **Test safety**: `TestCase::setUp()` refuses non-whitelisted databases unless `ALLOW_DESTRUCTIVE_TEST_DB=true` (allowed without the flag: `pgsql` with `DB_DATABASE` `testing` or `idx_api_testing`). `DB_*` comes from `.env` via `tests/bootstrap.php`, not from `phpunit.xml`.
- **phpunit.xml**: `tests/bootstrap.php` loads `.env` for `DB_*`; sync queue, fake Bridge keys, PULSE/TELESCOPE disabled
- **Factories**: `User::factory()->create()`; direct model creation for seed data
- **Config setup**: Tests set config values in `setUp()` (bridge host, dataset, tokens)

### Scheduled Tasks (routes/console.php)
| Task | Schedule | Queue |
|------|----------|-------|
| `mls-listings-cache-refresh` | Every 15 min via `mls:refresh-cache` (per domain × enabled feed), no overlap | default |
| `gis-source-metadata-probe` | Monday 6:30am, no overlap | `config('gis.queue')` |

## Environment Variables

### Core

| Variable | Required | Description |
|----------|----------|-------------|
| `APP_KEY` | Yes | Laravel encryption key |
| `APP_URL` | Yes | Base URL (becomes IDX_API_PUBLIC_URL by default) |
| `DB_CONNECTION` | Yes | `pgsql` (development, staging, production) |

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
| `php artisan gis:probe-sources` | Probe ArcGIS layer metadata (inline or queued) |
| `php artisan gis:clear-cache` | Clear GIS cache (all or by source) |
| `php artisan idx-api:issue-geo-web-token` | Create/rotate geo-web-internal Sanctum token |

## Docker Deployment

### Production and staging (Coolify / VPS)

FrankenPHP base (published to GHCR): **[`Dockerfile.frankenphp-base.production`](Dockerfile.frankenphp-base.production)** / **[`.staging`](Dockerfile.frankenphp-base.staging)**. Application images: **[`Dockerfile.production`](Dockerfile.production)** / **[`.staging`](Dockerfile.staging)** (`FROM` GHCR base via `FRANKENPHP_BASE_IMAGE`; targets `octane`, `queue-worker`, `scheduler`).

| Image | Dockerfile | Base | Port | Default process |
|-------|-----------|------|------|------------------|
| idx-api (web) | `Dockerfile.production` (target `octane`) | FrankenPHP + PHP 8.5 Alpine + intl | 8000 | Octane + FrankenPHP |
| idx-api (worker) | same file (target `queue-worker`) | (same image) | — | `queue:work` (Postgres `jobs` table) |
| idx-api (scheduler) | same file (target `scheduler`) | (same image) | — | `schedule:work` |
| idx-api staging (web / worker / scheduler) | `Dockerfile.staging` (targets `octane`, `queue-worker`, `scheduler`) | FrankenPHP + Xdebug | 8000 | same process layout as production |
| idx-images (staging + production) | `Dockerfile.idx-images` | Nginx 1.27 Alpine | 8080 | Reverse-proxy to `idx-api:8000` |

Build examples:

```bash
docker build -f Dockerfile.frankenphp-base.production -t ghcr.io/<owner>/<repo>-frankenphp:production .
docker build -f Dockerfile.production --target octane \
  --build-arg FRANKENPHP_BASE_IMAGE=ghcr.io/<owner>/<repo>-frankenphp:production \
  -t quantyra/idx-api:latest .
docker build -f Dockerfile.idx-images -t quantyra/idx-images:latest .
```

**Coolify (FrankenPHP base + Dockerfile app build):** GHA [`.github/workflows/docker-publish.yml`](.github/workflows/docker-publish.yml) publishes **`ghcr.io/<owner>/<repo>-frankenphp:production|staging`** from [`Dockerfile.frankenphp-base.production`](Dockerfile.frankenphp-base.production) / [`.staging`](Dockerfile.frankenphp-base.staging) (`linux/amd64`). Coolify uses the **[Dockerfile build pack](https://coolify.io/docs/builds/packs/dockerfile)** with [`Dockerfile.production`](Dockerfile.production) or [`.staging`](Dockerfile.staging), build-arg **`FRANKENPHP_BASE_IMAGE`**, targets **`octane`** / **`queue-worker`** / **`scheduler`**, web port **8000**. Fourth app: **`Dockerfile.idx-images`** on **8080**. See **[docs/coolify-deployment.md](docs/coolify-deployment.md)**.

**Resources (starting points — tune in Coolify per service):** VPS **2 vCPU / 4 GB** minimum (tight); staging **4–8 GB RAM** if using Xdebug + Telescope heavily; production **4 vCPU / 8 GB+** for proxy + queue spikes. Per service: web **0.5–1.0 CPU**, **1024–1536 MB RAM**; each worker replica **0.25–0.5 CPU**, **512–1024 MB RAM**; scheduler **0.1–0.25 CPU**, **256–384 MB RAM**. Reserve **1–2 GB** on the host for PostgreSQL if it shares the VPS. Container memory limit should exceed PHP `memory_limit` + headroom (~300 MB on web for opcache).

**Route cache:** the production/staging Dockerfiles run `config:cache` and `view:cache` at build; **`route:cache` is not run in production** during build because multiple `IDX_PLATFORM_HOSTS` entries register duplicate route names. Run `php artisan route:cache` post-deploy only for single-host environments or after route names are unique per domain.

### Development

```bash
./scripts/docker-dev.sh up-watch   # idx-api-dev + idx-images-dev + cloudflared-dev
./scripts/docker-dev.sh down       # Stop all dev services
```

Dev compose (`docker-compose.dev.yml`) runs **Octane only** in `idx-api-dev` (no queue or scheduler containers). For local jobs and the scheduler, use **`composer dev`** on the host (server + queue + logs + Vite) or point **staging/production worker images** at the same database. Includes Xdebug support (`XDEBUG_MODE`, `client_host=host.docker.internal`), file watching via Compose `develop.watch`, and Cloudflare tunnel for public HTTPS access. Local Docker on macOS: **[OrbStack](https://docs.orbstack.dev/docker)** works well with the same Compose file; see comments in [`Dockerfile.dev`](Dockerfile.dev).

## Testing

- **16 test files** across Feature (10) and Unit (4) suites
- Tests run against **PostgreSQL** using `DB_*` from **`.env`** (loaded in `tests/bootstrap.php` before PHPUnit) with the **sync** queue driver
- External APIs faked via `Http::fake()` (Bridge, ArcGIS in GIS tests)
- `TestCase::setUp()` enforces ephemeral database safety guard
- Coverage includes Bridge proxy security, image proxy headers, GIS probe/proxy, dashboard and marketing home, domain auth

## Additional Resources

- @docs/INDEX.md — Documentation index
- @docs/database-migrations.md — Migration inventory, PostGIS, legacy drops
- @docs/coolify-deployment.md — Coolify production & staging (four apps, env, networking, resources)
- @docs/idx-api-bridge-proxy.md — Bridge proxy architecture, auth flow, cache strategy, image rewrite
- @docs/bridge-api-documentation.md — Bridge Data Output upstream API reference
- @docs/gis-api.md — GIS parcel/geometry proxy documentation
- @README.md — Project overview and Docker build instructions

## Skill Usage Guide

When working on tasks involving these technologies, invoke the corresponding skill from [`.cursor/skills/`](.cursor/skills/) (see the [skills index](.cursor/skills/README.md)):

| Skill | Invoke When |
|-------|-------------|
| livewire | Manages Livewire reactive components and Blade integration |
| postgresql | Handles PostgreSQL schema, migrations, and query patterns |
| frontend-design | Applies UI design with Livewire, Blade, Tailwind CSS 4, and Alpine.js |
| laravel | Manages Laravel 13 routing, ORM, queues, and service providers |
| docker | Configures Docker multi-stage builds, FrankenPHP, and Compose workflows |
| php | Enforces PHP 8.5 patterns, strict typing, and constructor promotion |
| tailwind | Applies Tailwind CSS 4 styling and utility patterns |
| vite | Configures Vite build pipeline and HMR |
| nginx | Configures Nginx reverse proxy for idx-images |
| crafting-empty-states | Creates empty states and onboarding affordances |
| designing-inapp-guidance | Builds tooltips, tours, and contextual guidance |
| inspecting-search-coverage | Audits Bridge MLS filters, GIS queries, and on-page search coverage |
| laravel-best-practices | Laravel PHP patterns, security, queues, validation (under `.agents/skills` or `.cursor/skills`) |
| fortify-development | Fortify authentication customization |
| pulse-development | Laravel Pulse dashboards and recorders |

## Laravel Boost Guidelines

### Foundation Rules

The Laravel Boost guidelines are specifically curated by Laravel maintainers for this application and should be followed closely.

#### Foundational Context

This application is a Laravel application and its main Laravel ecosystem package versions are:

- php - 8.5
- laravel/framework (LARAVEL) - v13
- laravel/octane (OCTANE) - v2
- laravel/prompts (PROMPTS) - v0
- laravel/sanctum (SANCTUM) - v4
- livewire/livewire (LIVEWIRE) - v4
- laravel/boost (BOOST) - v2
- laravel/mcp (MCP) - v0
- laravel/pail (PAIL) - v1
- laravel/pint (PINT) - v1
- phpunit/phpunit (PHPUNIT) - v12

#### Skills Activation

Activate the relevant domain skill whenever working in that domain:

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
