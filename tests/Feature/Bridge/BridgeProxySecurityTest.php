<?php

namespace Tests\Feature\Bridge;

use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Client\Request;
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
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'bridge.images_public_base' => 'https://idx-images.test',
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
        $this->assertCount(3, $payload['value']);
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
        $plain = $user->createToken('integration', ['idx:access'])->plainTextToken;

        $response = $this->getJson('/api/v1/listings', [
            'Authorization' => 'Bearer '.$plain,
        ]);

        $response->assertOk();
        $this->assertCount(3, $response->json('value'));
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

        $user = User::factory()->create();
        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $response = $this->getJson('/api/v1/listings', [
            'Authorization' => 'Bearer '.$plain,
        ]);

        $response->assertOk();
        $payload = $response->json();
        $this->assertCount(10, $payload['value']);
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
        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $this->getJson('/api/v1/properties?city=largo&limit=10', [
            'Authorization' => 'Bearer '.$plain,
        ])->assertOk();

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
        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $this->postJson('/api/v1/properties', [
            'city' => 'largo',
            'limit' => 10,
        ], [
            'Authorization' => 'Bearer '.$plain,
        ])->assertOk();

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
        $plain = $user->createToken('integration', ['idx:full'])->plainTextToken;

        $response = $this->getJson('/api/v1/properties?city=largo&limit=10', [
            'Authorization' => 'Bearer '.$plain,
        ])->assertOk();

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

        $this->getJson('/api/v1/properties?cursor=TOKEN123', [
            'Authorization' => 'Bearer '.$plain,
        ])->assertOk();

        Http::assertSent(function (Request $request): bool {
            $query = $request->data();

            return str_contains($request->url(), '/stellar/Property')
                && isset($query['$next'])
                && $query['$next'] === 'TOKEN123'
                && ! isset($query['cursor']);
        });
    }
}
