<?php

namespace Tests\Feature\Bridge;

use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
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
}
