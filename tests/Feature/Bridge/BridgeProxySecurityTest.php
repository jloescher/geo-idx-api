<?php

namespace Tests\Feature\Bridge;

use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class BridgeProxySecurityTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.dataset' => 'stellar',
            'bridge.datasets' => ['stellar'],
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'bridge.images_public_base' => 'https://idx-images.test',
            'coingecko.cache_key' => 'coingecko.pricing.matrix',
            'coingecko.asset_ids' => ['btc', 'eth', 'sol', 'xrp', 'ada'],
            'coingecko.vs_currencies' => ['usd', 'cad', 'eur', 'gbp', 'mxn'],
        ]);
    }

    public function test_listings_requires_auth(): void
    {
        Http::fake();

        $this->getJson('/api/v1/listings')->assertStatus(401);
    }

    public function test_listings_teaser_for_registered_domain(): void
    {
        Domain::query()->create([
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
        ]);

        Http::fake([
            'https://bridge.test/stellar/listings*' => Http::response(
                json_encode(['value' => array_map(static fn (int $i): array => ['id' => $i], range(1, 10))], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $response = $this->getJson('/api/v1/listings', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertOk();
        $payload = $response->json();
        $this->assertIsArray($payload);
        $this->assertArrayHasKey('value', $payload);
        $this->assertCount(10, $payload['value']);
    }

    public function test_listings_teaser_for_idx_access_token(): void
    {
        Http::fake([
            'https://bridge.test/stellar/listings*' => Http::response(
                json_encode(['value' => array_map(static fn (int $i): array => ['id' => $i], range(1, 10))], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $user = User::factory()->create();
        $this->createVerifiedDomainForUser($user);
        $plain = $user->createToken('integration', ['idx:access'])->plainTextToken;

        $response = $this->getJson('/api/v1/listings', $this->tokenHeadersForDomain($plain));

        $response->assertOk();
        $this->assertCount(10, $response->json('value'));
    }

    public function test_listings_rejects_unknown_domain(): void
    {
        Http::fake();

        $this->getJson('/api/v1/listings', [
            'X-Domain-Slug' => 'not-registered.example',
        ])->assertStatus(403);
    }

    public function test_listings_accepts_domain_query_param(): void
    {
        Domain::query()->create([
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
        ]);

        Http::fake([
            'https://bridge.test/stellar/listings*' => Http::response(
                json_encode(['value' => [['id' => 1]]], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $this->getJson('/api/v1/listings?domain=searchtampabayhouses.com&limit=50')
            ->assertOk();
    }

    public function test_listings_rewrites_bridge_photo_urls_to_idx_images(): void
    {
        Domain::query()->create([
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
        ]);

        $payload = [
            'value' => [[
                'ListingKey' => 'LIST1',
                'Media' => [
                    [
                        'MediaURL' => 'https://api.bridgedataoutput.com/api/v2/stellar/listings/LIST1/photos/1',
                        'Order' => 1,
                    ],
                ],
            ]],
        ];

        Http::fake([
            'https://bridge.test/stellar/listings*' => Http::response(
                json_encode($payload, JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $response = $this->getJson('/api/v1/listings', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertOk();
        $mediaUrl = $response->json('value.0.Media.0.MediaURL');
        $this->assertIsString($mediaUrl);
        $this->assertStringStartsWith('https://idx-images.test/images/', $mediaUrl);
        $this->assertStringContainsString('LIST1', $mediaUrl);
    }

    public function test_listings_rewrites_cloudfront_media_urls_to_idx_images(): void
    {
        Domain::query()->create([
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
        ]);

        $payload = [
            'value' => [[
                'ListingKey' => 'LISTCF',
                'Media' => [
                    [
                        'MediaURL' => 'https://dvvjkgh94f2v6.cloudfront.net/735d922b/570651153/83dcefb7.jpeg',
                        'MediaKey' => 'LISTCF-m1',
                        'Order' => 1,
                    ],
                ],
            ]],
        ];

        Http::fake([
            'https://bridge.test/stellar/listings*' => Http::response(
                json_encode($payload, JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $response = $this->getJson('/api/v1/listings', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertOk();
        $mediaUrl = $response->json('value.0.Media.0.MediaURL');
        $this->assertIsString($mediaUrl);
        $this->assertStringStartsWith('https://idx-images.test/images/', $mediaUrl);
        $this->assertStringContainsString('LISTCF', $mediaUrl);
    }

    public function test_listings_full_for_idx_full_token(): void
    {
        $user = User::factory()->create();
        $this->createVerifiedDomainForUser($user);

        Http::fake([
            'https://bridge.test/stellar/listings*' => Http::response(
                json_encode(['value' => array_map(static fn (int $i): array => ['id' => $i], range(1, 10))], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $response = $this->getJson('/api/v1/listings', $this->tokenHeadersForDomain($plain));

        $response->assertOk();
        $payload = $response->json();
        $this->assertCount(10, $payload['value']);
    }

    public function test_listings_include_global_pricing_and_per_listing_conversions(): void
    {
        Cache::put('coingecko.pricing.matrix', [
            'quotes' => [
                'btc' => ['usd' => 100000.0, 'cad' => 136000.0, 'eur' => 92000.0, 'gbp' => 78000.0, 'mxn' => 1700000.0],
                'eth' => ['usd' => 4000.0, 'cad' => 5440.0, 'eur' => 3680.0, 'gbp' => 3120.0, 'mxn' => 68000.0],
                'sol' => ['usd' => 200.0, 'cad' => 272.0, 'eur' => 184.0, 'gbp' => 156.0, 'mxn' => 3400.0],
                'xrp' => ['usd' => 2.0, 'cad' => 2.72, 'eur' => 1.84, 'gbp' => 1.56, 'mxn' => 34.0],
                'ada' => ['usd' => 1.0, 'cad' => 1.36, 'eur' => 0.92, 'gbp' => 0.78, 'mxn' => 17.0],
            ],
            'as_of' => now()->toIso8601String(),
            'status' => 'ok',
        ], now()->addMinutes(20));

        Http::fake([
            'https://bridge.test/stellar/listings*' => Http::response(
                json_encode(['value' => [['ListingKey' => 'LIST1', 'ListPrice' => 500000]]], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $user = User::factory()->create();
        $this->createVerifiedDomainForUser($user);
        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $response = $this->getJson('/api/v1/listings', $this->tokenHeadersForDomain($plain));

        $response->assertOk();
        $payload = $response->json();
        $this->assertSame(100000, $payload['pricing']['quotes']['btc']['usd']);
        $this->assertEquals(500000.0, $payload['value'][0]['pricing_converted']['fiat']['usd']);
        $this->assertEquals(5.0, $payload['value'][0]['pricing_converted']['digital_assets']['btc']);
    }

    public function test_listings_pricing_enrichment_uses_cached_quotes_without_coingecko_http_calls(): void
    {
        Cache::put('coingecko.pricing.matrix', [
            'quotes' => [
                'btc' => ['usd' => 100000.0, 'cad' => 136000.0, 'eur' => 92000.0, 'gbp' => 78000.0, 'mxn' => 1700000.0],
            ],
            'as_of' => now()->toIso8601String(),
            'status' => 'ok',
        ], now()->addMinutes(20));

        Http::fake([
            'https://bridge.test/stellar/listings*' => Http::response(
                json_encode(['value' => [['ListingKey' => 'LIST1', 'ListPrice' => 500000]]], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
            'https://api.coingecko.com/*' => Http::response([], 500),
        ]);

        $user = User::factory()->create();
        $this->createVerifiedDomainForUser($user);
        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $this->getJson('/api/v1/listings', $this->tokenHeadersForDomain($plain))->assertOk();

        Http::assertSentCount(1);
    }

    public function test_properties_translates_city_and_limit_to_odata_params(): void
    {
        Http::fake([
            'https://bridge.test/stellar/Property*' => Http::response(
                json_encode(['value' => [['ListingKey' => 'stellar:1']]], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $user = User::factory()->create();
        $this->createVerifiedDomainForUser($user);
        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $this->getJson('/api/v1/properties?city=largo&limit=10', $this->tokenHeadersForDomain($plain))->assertOk();

        Http::assertSent(function (Request $request): bool {
            $query = $request->data();

            return str_contains($request->url(), '/stellar/Property')
                && isset($query['$filter'])
                && $query['$filter'] === "contains(tolower(City),'largo')"
                && isset($query['$top'])
                && (int) $query['$top'] === 10
                && ! isset($query['city'])
                && ! isset($query['limit']);
        });
    }

    public function test_properties_accepts_json_body_for_city_and_limit(): void
    {
        Http::fake([
            'https://bridge.test/stellar/Property*' => Http::response(
                json_encode(['value' => [['ListingKey' => 'stellar:1']]], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $user = User::factory()->create();
        $this->createVerifiedDomainForUser($user);
        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $this->postJson('/api/v1/properties', [
            'city' => 'largo',
            'limit' => 10,
        ], $this->tokenHeadersForDomain($plain))->assertOk();

        Http::assertSent(function (Request $request): bool {
            $query = $request->data();

            return str_contains($request->url(), '/stellar/Property')
                && isset($query['$filter'])
                && $query['$filter'] === "contains(tolower(City),'largo')"
                && isset($query['$top'])
                && (int) $query['$top'] === 10;
        });
    }

    public function test_properties_rewrites_odata_links_and_supports_cursor_pagination(): void
    {
        Http::fake([
            'https://bridge.test/stellar/Property*' => Http::response(
                json_encode([
                    'value' => [['ListingKey' => 'stellar:1']],
                    '@odata.id' => "https://api.bridgedataoutput.com/api/v2/OData/stellar/Property('stellar:1')",
                    '@odata.nextLink' => 'https://api.bridgedataoutput.com/api/v2/OData/stellar/Property?%24top=10&$next=TOKEN123',
                ], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $user = User::factory()->create();
        $this->createVerifiedDomainForUser($user);
        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $response = $this->getJson('/api/v1/properties?city=largo&limit=10', $this->tokenHeadersForDomain($plain))->assertOk();

        $payload = $response->json();
        $nextLink = $payload['@odata.nextLink'] ?? null;
        $this->assertIsString($nextLink);
        $this->assertStringContainsString('/api/v1/properties?cursor=TOKEN123', $nextLink);
        $this->assertSame(urlencode('stellar:1'), basename((string) ($payload['@odata.id'] ?? '')));

        Http::fake([
            'https://bridge.test/stellar/Property*' => Http::response(
                json_encode(['value' => [['ListingKey' => 'stellar:2']]], JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $this->getJson('/api/v1/properties?cursor=TOKEN123', $this->tokenHeadersForDomain($plain))->assertOk();

        Http::assertSent(function (Request $request): bool {
            $query = $request->data();

            return str_contains($request->url(), '/stellar/Property')
                && isset($query['$next'])
                && $query['$next'] === 'TOKEN123'
                && ! isset($query['cursor']);
        });
    }

    /**
     * @return array<string, string>
     */
    private function tokenHeadersForDomain(string $plainToken, string $slug = 'searchtampabayhouses.com'): array
    {
        return [
            'Authorization' => 'Bearer '.$plainToken,
            'X-Domain-Slug' => $slug,
        ];
    }

    private function createVerifiedDomainForUser(User $user, string $slug = 'searchtampabayhouses.com'): void
    {
        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => $slug,
            'is_active' => true,
            'verification_status' => 'verified',
            'mls_dataset' => 'stellar',
            'allowed_mls_datasets' => ['stellar'],
        ]);
    }
}
