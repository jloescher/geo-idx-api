---
name: laravel
description: Manages Laravel 13 routing, ORM, queues, and service providers
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Laravel Skill

Laravel 13 + Octane service with FrankenPHP, Eloquent ORM, and PostgreSQL. Uses PHP 8 attributes for model definitions, constructor property promotion, and import-style routing.

## Quick Start

```bash
cp .env.example .env
composer install
php artisan key:generate
php artisan migrate
composer dev  # server + queue + logs + Vite
```

Run tests: `composer test` (PostgreSQL test database from `phpunit.xml`, sync queue)
Code format: `vendor/bin/pint`

## Key Concepts

**Routing** — Import-style with `use` statements in `routes/`:
- `api.php` — Sanctum-auth API routes (`/api` prefix)
- `web.php` — Web routes (marketing, dashboard, agent JSON endpoints)
- `console.php` — Scheduled tasks

**Models** — PHP 8 attributes for fillable/hidden, `casts()` method:
```php
#[Fillable(['name', 'email'])]
#[Hidden(['password'])]
class User extends Model { }
```

**Middleware** — Registered in `bootstrap/app.php`:
- `domain.token` — `DomainOrTokenAuth` for Bridge/GIS proxy
- `mls.access` — feed access for MLS clients

**Queues** — Database queue with jobs in `app/Jobs/`:
- `RefreshDomainListingsCacheJob`
- `RefreshGisSourceMetadataJob`

**Service Providers** — `app/Providers/`:
- `AppServiceProvider` — migration loading, model bindings
- `FortifyServiceProvider` — auth scaffolding
- `TelescopeServiceProvider` — local debugging

## Common Patterns

**Constructor Injection** — Readonly properties:
```php
public function __construct(
    private readonly Service $svc
) {}
```

**HTTP Client** — `Http::fake()` in tests, `Http::withHeaders()` for external APIs

**Artisan Commands** — `app/Console/Commands/` with `handle()` method and `$signature`

**Migrations** — `database/migrations/` with dated migration classes

**Config** — Use `env()` directly in config files (not `config()` to avoid cache-breaking)