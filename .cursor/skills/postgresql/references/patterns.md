# PostgreSQL Patterns

## When to use
Use these patterns when designing schema, writing migrations, or querying PostgreSQL in the idx-api Laravel service.

## Migration Patterns

**GHL Isolated Migrations**
```php
// Store GHL marketplace migrations separately
// database/migrations/ghl/2026_04_22_120000_create_ghl_oauth_tokens_table.php

// Load via AppServiceProvider::boot()
public function boot(): void
{
    $this->loadMigrationsFrom(__DIR__.'/../../database/migrations/ghl');
}
```

**JSON Columns for Flexible Data**
```php
Schema::create('ghl_registered_urls', function (Blueprint $table) {
    $table->id();
    $table->string('primary_url');
    $table->json('additional_urls')->nullable();  // Array of extra origins
    $table->json('default_tags')->nullable();     // Tag configuration
    $table->timestamps();
});
```

**Soft Deletes + Unique Constraints**
```php
// For token lookup with revocation support
$table->string('access_token_hash')->unique();
$table->softDeletes(); // Unique constraint ignores soft-deleted rows
```

## Model Patterns

**PHP 8 Attributes (Preferred)**
```php
use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Attributes\Hidden;

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

**Encrypted At-Rest Storage**
```php
protected function casts(): array
{
    return [
        'access_token' => 'encrypted',
        'refresh_token' => 'encrypted',
    ];
}
```

**Generation-Based Cache Invalidation**
```sql
-- Check GIS cache validity by comparing generations
SELECT * FROM gis_cache c
JOIN gis_source_states s ON c.source_used = s.source_name
WHERE c.query_hash = ?
AND c.source_generation = s.generation
AND c.expires_at > NOW();
```

## Warning
Never rely on `$casts` property arrays—this codebase uses PHP 8 attributes and the `casts()` method exclusively. For JSON columns, remember `->nullable()` when the column is optional so factories and partial updates behave consistently.