<?php

namespace Tests\Feature\Bridge;

use App\Models\Domain;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Storage;
use Tests\TestCase;

class ImageProxyResponseTest extends TestCase
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
            'bridge.listing_photo_path_template' => '/api/v2/{dataset}/listings/{listingKey}/photos/{photoId}',
        ]);

        Storage::fake('images');
    }

    public function test_image_proxy_returns_immutable_cache_headers(): void
    {
        Domain::query()->create([
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
        ]);

        Http::fake([
            'https://bridge.test/*' => Http::response('fake-bytes', 200, ['Content-Type' => 'image/jpeg']),
        ]);

        $response = $this->get('/images/LK1/1', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertOk();
        $cc = (string) $response->headers->get('Cache-Control');
        $this->assertStringContainsString('public', $cc);
        $this->assertStringContainsString('max-age=31536000', $cc);
        $this->assertStringContainsString('immutable', $cc);
        $this->assertStringContainsString('idx-images', $response->headers->get('X-IDX-Proxied-Public-Url', ''));
    }
}
