# SQLite Database Patterns

When to use: Working with the dual-database Laravel setup where SQLite is used for local development and testing while PostgreSQL runs in production.

## Pattern: Dual Database Configuration

This codebase uses `DB_CONNECTION` to switch between SQLite (local/dev) and PostgreSQL (production).

**Local development (.env):**
```bash
DB_CONNECTION=sqlite
DB_DATABASE=database/database.sqlite
```

**Production environment:**
```bash
DB_CONNECTION=pgsql
DB_HOST=localhost
DB_DATABASE=quantyra_idx
```

Always verify connection before running destructive operations:
```bash
grep DB_CONNECTION .env
```

## Pattern: In-Memory Test Isolation

Tests use SQLite `:memory:` to ensure complete isolation and fast execution. The `TestCase.php` enforces this guard:

```php
// From tests/TestCase.php - prevents running tests against real databases
protected function setUp(): void
{
    parent::setUp();
    
    $connection = config('database.default');
    $database = config("database.connections.{$connection}.database");
    
    if ($connection !== 'sqlite' || $database !== ':memory:') {
        if (!env('ALLOW_DESTRUCTIVE_TEST_DB')) {
            throw new \RuntimeException('Tests must run against SQLite :memory:');
        }
    }
}
```

**Never** set `ALLOW_DESTRUCTIVE_TEST_DB=true` in production or against a PostgreSQL database.

## Pattern: Migration Path Organization

Migrations are split between core and GHL-specific paths:

| Path | Purpose |
|------|---------|
| `database/migrations/` | Core tables (users, domains, listings_cache, bridge_proxy_audit_logs) |
| `database/migrations/ghl/` | GHL Marketplace tables (tokens, webhooks, widgets, leads) |

GHL migrations load via `AppServiceProvider`:

```php
$this->loadMigrationsFrom(database_path('migrations/ghl'));
```

Run GHL migrations explicitly:
```bash
php artisan migrate --path=database/migrations/ghl
```

---

## ⚠️ Warning: Database Seeding Safety

`DatabaseSeeder` contains token generation logic that only runs when tokens are missing. Running `migrate:fresh --seed` on a production database will regenerate tokens and invalidate existing API keys. Always verify you're in a development environment before running fresh seeds.