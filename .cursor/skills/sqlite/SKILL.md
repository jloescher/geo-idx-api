---
name: sqlite
description: Handles SQLite local development databases and in-memory test databases for Laravel projects using ephemeral SQLite for testing and file-based SQLite for local development
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Sqlite Skill

Manages SQLite database workflows for Laravel applications that use SQLite for local development and in-memory databases for testing while running PostgreSQL in production.

## Quick Start

```bash
# Check current database configuration
grep DB_CONNECTION .env

# Run migrations on current database
php artisan migrate

# Run migrations with fresh seeders (destructive)
php artisan migrate:fresh --seed

# Run tests (uses in-memory SQLite via phpunit.xml)
composer test

# Run specific test file
php artisan test tests/Feature/Bridge/ProxyTest.php
```

## Key Concepts

**Dual Database Setup**: This codebase uses SQLite for local development and testing, PostgreSQL for production. Configuration flows through `DB_CONNECTION` in `.env`.

**In-Memory Testing**: `phpunit.xml` configures SQLite `:memory:` for tests. `TestCase.php` guards against running tests against non-ephemeral databases via `setUp()` checks that require SQLite `:memory:` or `ALLOW_DESTRUCTIVE_TEST_DB=true`.

**Migration Organization**: Core migrations live in `database/migrations/`. GHL-specific migrations load via `AppServiceProvider::loadMigrationsFrom('database/migrations/ghl')`.

**Seeder Safety**: `DatabaseSeeder` delegates to domain seeders (`DomainSeeder`, `GeoWebInternalTokenSeeder`, `GhlConfigSeeder`). Tokens and sensitive data seed only when missing.

## Common Patterns

**Switching to local file-based SQLite** (for persistent local data):
```bash
# .env
DB_CONNECTION=sqlite
DB_DATABASE=database/database.sqlite
```

**Creating the database file**:
```bash
touch database/database.sqlite
php artisan migrate
```

**Debugging test database issues**:
```bash
# Check if tests are hitting wrong database
grep -A5 '<env name="DB_' phpunit.xml

# Verify TestCase guard
head -30 tests/TestCase.php
```

**Running migrations for specific paths** (GHL tables):
```bash
php artisan migrate --path=database/migrations/ghl
```

**Safe test environment variables** (from `phpunit.xml`):
- `DB_CONNECTION=sqlite`
- `DB_DATABASE=:memory:`
- `QUEUE_CONNECTION=sync`