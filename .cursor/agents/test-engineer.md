---
  Feature and unit tests for Bridge, GHL, GIS, and billing flows using in-memory SQLite and Http::fake()
tools: Read, Edit, Write, Glob, Grep, Bash
skills: php, laravel, postgresql, livewire, tailwind, frontend-design, stripe, docker, scoping-feature-work, prioritizing-roadmap-bets, mapping-user-journeys, designing-onboarding-paths, improving-activation-flow, crafting-empty-states, orchestrating-feature-adoption, designing-inapp-guidance, instrumenting-product-metrics, running-product-experiments, triaging-user-feedback, writing-release-notes, clarifying-market-fit, structuring-offer-ladders, framing-release-stories, generating-growth-hypotheses, embedding-decision-cues, crafting-page-messaging, tightening-brand-voice, designing-lifecycle-messages, planning-editorial-arcs, orchestrating-social-rhythm, tuning-landing-journeys, streamlining-signup-steps, accelerating-first-run, reducing-form-falloff, refining-prompt-surfaces, strengthening-upgrade-moments, mapping-conversion-events, designing-variation-tests, calibrating-paid-campaigns, building-acquisition-tools, engineering-referral-loops, inspecting-search-coverage, scaling-template-pages, adding-structured-signals, building-compare-hubs
name: test-engineer
model: inherit
description: |
---

# Run all tests
composer test
# OR
php artisan test --compact

# Run specific test file
php artisan test tests/Feature/BridgeProxyTest.php

# Run with filter
php artisan test --filter=test_listings_endpoint
```

## Writing Feature Tests

**HTTP Testing Pattern:**
```php
<?php

namespace Tests\Feature;

use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class BridgeProxyTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();
        
        // Set required config for tests
        config(['bridge.host' => 'https://api.bridgedataoutput.com']);
        config(['bridge.dataset' => 'stellar']);
        
        // Fake external APIs
        Http::fake([
            'api.bridgedataoutput.com/*' => Http::response([
                'value' => [['ListingKey' => 'TEST123', 'Media' => []]]
            ], 200),
        ]);
    }

    public function test_listings_endpoint_returns_teaser_for_domain_auth(): void
    {
        // Arrange: Create domain
        $domain = \App\Models\Domain::factory()->create([
            'domain_slug' => 'test.com',
            'is_active' => true,
        ]);

        // Act: Make request with domain header
        $response = $this->withHeader('X-Domain-Slug', 'test.com')
            ->getJson('/api/v1/listings');

        // Assert
        $response->assertOk()
            ->assertJsonStructure(['value'])
            ->assertJsonCount(3, 'value'); // Teaser limit
    }

    public function test_listings_endpoint_requires_auth(): void
    {
        $response = $this->getJson('/api/v1/listings');

        $response->assertUnauthorized();
    }
}
```

## Writing Unit Tests

**Isolated Logic Pattern:**
```php
<?php

namespace Tests\Unit;

use App\Services\Bridge\BridgeImageUrlRewriter;
use Tests\TestCase;

class BridgeImageUrlRewriterTest extends TestCase
{
    public function test_rewrites_bridge_photo_urls_to_idx_images(): void
    {
        $service = new BridgeImageUrlRewriter();
        
        $json = json_encode([
            'Media' => [[
                'MediaURL' => 'https://api.bridgedataoutput.com/stellar/listings/L123/photos/P456'
            ]]
        ]);

        $result = $service->rewrite($json);

        $this->assertStringContainsString('idx-images.quantyralabs.cc/images/L123/P456', $result);
    }
}
```

## Database Testing

**Factory Usage:**
```php
// Create with factory
$user = \App\Models\User::factory()->create();
$domain = \App\Models\Domain::factory()->create(['is_active' => true]);

// Assert database state
$this->assertDatabaseHas('domains', [
    'domain_slug' => 'test.com',
    'is_active' => true,
]);

$this->assertDatabaseCount('bridge_proxy_audit_logs', 1);
```

**Seeding for Tests:**
```php
// In setUp or individual tests
$this->seed(\Database\Seeders\GhlConfigSeeder::class);
$this->seed(\Database\Seeders\DomainSeeder::class);
```

## Mocking External APIs

**Bridge Data Output:**
```php
Http::fake([
    // Specific endpoint
    'api.bridgedataoutput.com/stellar/listings' => Http::response([
        'value' => $listings,
    ], 200),
    
    // Fallback for any Bridge call
    'api.bridgedataoutput.com/*' => Http::response([], 200),
]);

// Assert request was made
Http::assertSent(function ($request) {
    return $request->url() === 'https://api.bridgedataoutput.com/stellar/listings';
});
```

**Stripe:**
```php
Http::fake([
    'api.stripe.com/v1/customers' => Http::response([
        'id' => 'cus_test123',
        'object' => 'customer',
    ], 200),
]);
```

## Testing GHL OAuth Flows

```php
public function test_oauth_callback_exchanges_code_and_stores_token(): void
{
    Http::fake([
        'services.leadconnectorhq.com/oauth/token' => Http::response([
            'access_token' => 'test_token',
            'refresh_token' => 'test_refresh',
            'expires_in' => 3600,
            'locationId' => 'loc123',
        ], 200),
    ]);

    $response = $this->withSession(['oauth_state' => 'test_state'])
        ->getJson('/oauth/leadconnector/callback?code=auth_code&state=test_state');

    $response->assertRedirect();
    
    $this->assertDatabaseHas('ghl_oauth_tokens', [
        'ghl_location_id' => 'loc123',
        'status' => 'active',
    ]);
}
```

## Testing Widget Endpoints

```php
public function test_widget_lead_ingestion_creates_quantyra_lead(): void
{
    $apiKey = 'qh_test123';
    
    // Setup registered URL with API key
    \App\Models\Ghl\RegisteredUrl::factory()->create([
        'widget_api_key' => $apiKey,
        'primary_url' => 'https://example.com',
    ]);

    $response = $this->withHeader('Origin', 'https://example.com')
        ->postJson('/widget/api/leads', [
            'api_key' => $apiKey,
            'lead_type' => 'showing_request',
            'first_name' => 'John',
            'email' => 'john@example.com',
        ]);

    $response->assertCreated();
    
    $this->assertDatabaseHas('quantyra_leads', [
        'lead_type' => 'showing_request',
    ]);
}
```

## Testing GIS Proxy

```php
public function test_gis_endpoint_returns_geojson_for_valid_bbox(): void
{
    Http::fake([
        'services.arcgis.com/*' => Http::response([
            'features' => [
                ['geometry' => ['rings' => []], 'attributes' => ['PARCELID' => '12345']],
            ],
        ], 200),
    ]);

    $domain = \App\Models\Domain::factory()->create(['is_active' => true]);

    $response = $this->withHeader('X-Domain-Slug', $domain->domain_slug)
        ->getJson('/api/v1/gis?bbox=-82.83,27.95,-82.79,27.98');

    $response->assertOk()
        ->assertJsonStructure([
            'type',
            'features',
            'meta' => ['source_used', 'county_hint'],
        ]);
}
```

## Testing Billing/Stripe

```php
public function test_checkout_session_creation_requires_auth(): void
{
    $user = \App\Models\User::factory()->create();

    $response = $this->actingAs($user)
        ->postJson('/billing/checkout', [
            'plan' => 'pro',
            'interval' => 'month',
        ]);

    $response->assertRedirectContains('stripe.com');
}
```

## CRITICAL for This Project

1. **NEVER run tests against production databases** - TestCase enforces SQLite `:memory:` or `ALLOW_DESTRUCTIVE_TEST_DB=true`

2. **Always use Http::fake() for external APIs** - Bridge, Stripe, GHL APIs must be mocked

3. **Domain authentication in tests** - Use `X-Domain-Slug` header or create Sanctum tokens with abilities

4. **Teaser vs full access** - Test both paths (domain auth = teaser, `idx:full` token = full)

5. **Assert audit logging** - Many endpoints write to `bridge_proxy_audit_logs` or `ghl_audit_logs`

6. **Test CORS behavior** - Widget endpoints require Origin header validation

7. **Queue assertions** - Use `Bus::fake()` or `Queue::fake()` for job dispatch testing

## File Paths to Know

- Base test class: `tests/TestCase.php`
- Feature tests: `tests/Feature/*.php`
- Unit tests: `tests/Unit/*.php`
- Factories: `database/factories/*.php`
- Seeders: `database/seeders/*.php`
- PHPUnit config: `phpunit.xml`

## Test Commands Reference

```bash
# Full suite
composer test

# With coverage (if configured)
php artisan test --coverage

# Specific file
php artisan test tests/Feature/Ghl/WebhookTest.php

# Filter by method name
php artisan test --filter=test_install_webhook

# Parallel (if configured)
php artisan test --parallel