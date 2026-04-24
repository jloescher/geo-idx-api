# PostgreSQL Workflows

## When to use
Use these workflows for local development setup, production deployment, and maintenance operations.

## Local Development Workflow

```bash
# Local uses SQLite by default; ensure migrations exist
php artisan migrate

# Run GHL-specific migrations (loaded via AppServiceProvider)
php artisan migrate --path=database/migrations/ghl

# Seed GHL configuration data
php artisan db:seed --class=GhlConfigSeeder
```

**Test Safety Guard**
Tests enforce ephemeral databases. `TestCase::setUp()` blocks against accidental PostgreSQL truncation unless `ALLOW_DESTRUCTIVE_TEST_DB=true`.

## Production Setup Workflow

```bash
# Configure .env for PostgreSQL
DB_CONNECTION=pgsql
DB_HOST=localhost
DB_DATABASE=idx_api
DB_USERNAME=quantyra
DB_PASSWORD=secret

# Queue uses database connection
QUEUE_CONNECTION=database

# Run all migrations (core + GHL auto-loaded)
php artisan migrate

# Verify GIS source states table exists for cache invalidation
php artisan gis:probe-sources
```

## GIS Cache Maintenance Workflow

```bash
# Weekly: Refresh source metadata to detect ArcGIS layer changes
php artisan gis:probe-sources --queued

# Clear cache for a specific source (invalidates by bumping generation)
php artisan gis:clear-cache --source=pinellas_enterprise_parcels

# Emergency: Clear all GIS cache
php artisan gis:clear-cache --all
```

## Token Rotation Workflow

```bash
# Rotate the internal geo-web Sanctum token
php artisan idx-api:issue-geo-web-token --force

# Hourly scheduled: Refresh expiring GHL OAuth tokens
php artisan ghl:refresh-tokens
```

## Warning
GIS cache relies on `gis_source_states.generation` counters—never manually truncate `gis_cache` without bumping the corresponding source generation, or you'll serve stale data from PostgreSQL while the Laravel Cache edge layer may still hold valid-looking entries.