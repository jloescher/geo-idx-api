---
name: postgresql
description: Handles PostgreSQL schema, migrations, and query patterns for the Laravel idx-api service
allowed-tools: [Read, Edit, Write, Glob, Grep, Bash]
---

# Postgresql Skill

Manages PostgreSQL schema design, migrations, and query patterns for a Laravel 13 + Octane service handling MLS proxy data, GHL marketplace integration, and GIS parcel caching. The codebase uses SQLite for local development and testing, PostgreSQL for production.

## Quick Start

```bash
# Local development uses SQLite; ensure migrations run
php artisan migrate

# Production PostgreSQL connection (configured via env)
DB_CONNECTION=pgsql
DB_HOST=localhost
DB_DATABASE=idx_api
DB_USERNAME=quantyra
DB_PASSWORD=secret

# Run GHL-specific migrations (loaded via AppServiceProvider)
php artisan migrate --path=database/migrations/ghl

# Seed GHL configuration data
php artisan db:seed --class=GhlConfigSeeder
```

## Key Concepts

**Migration Structure**: Core migrations live in `database/migrations/`; GHL marketplace migrations are isolated in `database/migrations/ghl/` and loaded via `AppServiceProvider::loadMigrationsFrom()`. Migration filenames follow Laravel's `YYYY_MM_DD_HHMMSS_description.php` format.

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

**Dual-Channel Audit**: GHL operations write to both `ghl_audit_logs` (PostgreSQL) and optionally `storage/logs/ghl_audit.log` (filesystem) via `GhlAuditService`.

**Domain-Scoped Caching**: `listings_cache` uses `domain_slug` as primary key for per-domain MLS listing caches with gzip-compressed `compressed_data` columns.

**Ephemeral Test Guard**: Tests enforce SQLite `:memory:` or require `ALLOW_DESTRUCTIVE_TEST_DB=true` to prevent accidental PostgreSQL truncation in `TestCase::setUp()`.

## Common Patterns

**Migration with JSON Columns** (GHL tables):
```php
Schema::create('ghl_registered_urls', function (Blueprint $table) {
    $table->id();
    $table->string('primary_url');
    $table->json('additional_urls')->nullable();
    $table->string('widget_api_key')->unique();
    $table->enum('integration_type', ['ghl_website', 'external_website', 'both']);
    $table->timestamps();
});
```

**Encrypted Token Storage**:
```php
// In GhlOAuthToken model
protected function casts(): array
{
    return [
        'access_token' => 'encrypted',
        'refresh_token' => 'encrypted',
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