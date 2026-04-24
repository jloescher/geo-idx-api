# SQLite Workflows

When to use: Day-to-day database operations for local development, testing, and troubleshooting SQLite-specific issues.

## Workflow: Local File-Based SQLite Setup

Create persistent local database for development:

```bash
# 1. Create the database file
touch database/database.sqlite

# 2. Configure .env
echo "DB_CONNECTION=sqlite" >> .env
echo "DB_DATABASE=database/database.sqlite" >> .env

# 3. Run migrations
php artisan migrate

# 4. Optional: Seed with test data
php artisan db:seed
```

## Workflow: Testing with In-Memory SQLite

The test suite is pre-configured in `phpunit.xml` to use `:memory:`:

```xml
<env name="DB_CONNECTION" value="sqlite"/>
<env name="DB_DATABASE" value=":memory:"/>
<env name="QUEUE_CONNECTION" value="sync"/>
```

Run tests without touching disk:
```bash
# Full test suite
composer test

# Specific test file
php artisan test tests/Feature/Bridge/ProxyTest.php

# With coverage
php artisan test --coverage
```

## Workflow: Debugging Test Database Configuration

When tests fail with database errors, verify the configuration:

```bash
# Check phpunit.xml test environment
grep -A5 'DB_' phpunit.xml

# Verify TestCase guard is active
head -30 tests/TestCase.php

# Check current database connection
php artisan tinker --execute="echo config('database.default');"
```

## Workflow: Switching Database Connections

Temporarily switch to PostgreSQL for production-like testing:

```bash
# Backup current .env
cp .env .env.sqlite.backup

# Switch to PostgreSQL
sed -i '' 's/DB_CONNECTION=sqlite/DB_CONNECTION=pgsql/' .env

# Run migrations (will use PostgreSQL)
php artisan migrate

# Restore SQLite
mv .env.sqlite.backup .env
```

---

## ⚠️ Warning: Migration Fresh Destroys Data

`php artisan migrate:fresh` drops all tables. On SQLite file-based databases, this is destructive and irreversible. Always ensure:
- You're using `:memory:` for tests (automatic)
- You have backups of `database/database.sqlite` if it contains important local data
- You've committed seed data changes before running fresh migrations