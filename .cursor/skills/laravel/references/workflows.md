# Laravel Workflows in IDX API

When to use: Setting up development environment, adding features, or debugging the Quantyra IDX API.

## Adding a Bridge/GIS API Endpoint

1. **Create controller method** in `app/Http/Controllers/Api/`
2. **Add route** in `routes/api.php` with `domain.token` middleware
3. **Add feature test** in `tests/Feature/` using `Http::fake()` and `RefreshDatabase`

```php
// Test pattern
public function test_endpoint_returns_teaser_for_domain(): void
{
    Http::fake(['*' => Http::response(['value' => []])]);
    
    $response = $this->withHeader('X-Domain-Slug', 'example.com')
        ->getJson('/api/v1/endpoint');
    
    $response->assertOk();
}
```

## Creating a Scheduled Job

1. **Create command** in `app/Console/Commands/` with `$signature`
2. **Add schedule** in `routes/console.php`
3. **Run queue worker** for async execution

```php
// routes/console.php
use App\Jobs\RefreshDomainListingsCacheJob;

Schedule::job(new RefreshDomainListingsCacheJob($domain))
    ->everyFifteenMinutes()
    ->withoutOverlapping();
```

## Local Development Setup

```bash
# Terminal 1: Full dev stack (server + queue + Vite + logs)
composer dev

# Terminal 2: Stripe webhook forwarding (optional)
./scripts/stripe-dev.sh listen

# Run tests
composer test

# Code formatting
vendor/bin/pint
```

## Pitfall: database safety in tests

PHPUnit forces a dedicated PostgreSQL database (see `phpunit.xml`, default `idx_api_testing`). `tests/TestCase.php` refuses other `DB_DATABASE` values when `APP_ENV=testing`, unless `ALLOW_DESTRUCTIVE_TEST_DB=true`. Never point tests at shared staging or production.