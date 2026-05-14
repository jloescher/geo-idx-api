---
name: postgresql
description: Handles PostgreSQL schema, migrations, and query patterns for the Laravel idx-api service
allowed-tools: [Read, Edit, Write, Glob, Grep, Bash]
---

# Postgresql Skill

Manages PostgreSQL schema design, migrations, and query patterns for a Laravel 13 + Octane service handling MLS proxy data, GIS parcel caching, and subscriber dashboards. Development, staging, production, and automated tests all use PostgreSQL (`pgsql`).

## Quick Start

```bash
# Configure DB_* in .env, then apply migrations
php artisan migrate
```

```bash
# Example PostgreSQL connection (adjust per environment)
DB_CONNECTION=pgsql
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=idx_api
DB_USERNAME=postgres
DB_PASSWORD=
```

## Key Concepts

**Migration Structure**: Core migrations live in `database/migrations/`. Migration filenames follow Laravel's `YYYY_MM_DD_HHMMSS_description.php` format.

**PHP 8 Model Attributes**: Models use PHP 8 attributes instead of property arrays:
```php
#[Fillable(['domain_slug', 'is_active'])]
#[Hidden(['compressed_data'])]
class ListingsCache extends Model
{
    protected function casts(): array
    {
        return [
            'last_updated' => 'datetime',
            'compressed_data' => 'binary',
        ];
    }
}
```

**GIS Caching Strategy**: PostgreSQL stores GIS cache blobs with generation-based invalidation. The `gis_cache` table stores `query_hash`, `geojson_blob`, `source_generation`, and `expires_at`. Cache hits require matching `source_generation` against `gis_source_states.generation`.

**Bridge proxy audit**: MLS proxy requests can be logged to `bridge_proxy_audit_logs` where enabled.

**Domain-Scoped Caching**: `listings_cache` stores per-domain MLS listing caches with gzip-compressed `compressed_data` columns (see migrations for the current primary key shape).

**Ephemeral Test Guard**: PHPUnit uses a dedicated PostgreSQL database (see `phpunit.xml`, default `idx_api_testing`). `TestCase::setUp()` allows only `testing` or `idx_api_testing` unless `ALLOW_DESTRUCTIVE_TEST_DB=true`.

## Common Patterns

**Migration with JSON columns**:
```php
Schema::create('example_settings', function (Blueprint $table) {
    $table->id();
    $table->json('payload')->nullable();
    $table->timestamps();
});
```

**Encrypted columns**:
```php
protected function casts(): array
{
    return [
        'secret' => 'encrypted',
        'expires_at' => 'datetime',
    ];
}
```

**Soft Deletes with Unique Constraints**:
```php
$table->string('access_token_hash')->unique();
$table->softDeletes(); // Combined with unique hash for token lookup
```

**Queue Worker Configuration** (PostgreSQL production):
```php
// config/queue.php or env
QUEUE_CONNECTION=database  # Uses PostgreSQL jobs table
```

**Generation-Based Cache Invalidation**:
```sql
-- Check cache validity by comparing generations
SELECT * FROM gis_cache c
JOIN gis_source_states s ON c.source_used = s.source_name
WHERE c.query_hash = ?
AND c.source_generation = s.generation
AND c.expires_at > NOW();