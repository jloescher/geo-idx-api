<?php

namespace Tests\Unit\Bridge;

use App\Services\Bridge\BridgeImageUrlRewriter;
use Tests\TestCase;

class BridgeImageUrlRewriterTest extends TestCase
{
    public function test_rewrites_listings_photos_url(): void
    {
        config([
            'bridge.images_public_base' => 'https://idx-images.test',
            'bridge.host' => 'https://api.bridgedataoutput.com',
        ]);

        $json = json_encode([
            'value' => [[
                'ListingKey' => 'K1',
                'Media' => [
                    [
                        'MediaURL' => 'https://api.bridgedataoutput.com/api/v2/stellar/listings/K1/photos/0',
                        'Order' => 0,
                    ],
                ],
            ]],
        ], JSON_THROW_ON_ERROR);

        $out = (new BridgeImageUrlRewriter)->rewriteJson($json);
        $this->assertStringContainsString('https://idx-images.test/images/', $out);
        $this->assertStringContainsString('listings', $json);
        $this->assertStringNotContainsString('api.bridgedataoutput.com/api/v2', $out);
    }

    public function test_normalizes_idx_images_host_to_current_environment(): void
    {
        config([
            'bridge.images_public_base' => 'https://dev-idx-images.quantyralabs.cc',
            'bridge.host' => 'https://api.bridgedataoutput.com',
        ]);

        $json = json_encode([
            'value' => [[
                'ListingKey' => 'c8b74c4ee86a9de4f7c845fa28ecb18b',
                'Media' => [[
                    'MediaURL' => 'https://idx-images.quantyralabs.cc/images/c8b74c4ee86a9de4f7c845fa28ecb18b/1',
                    'Order' => 1,
                ]],
            ]],
        ], JSON_THROW_ON_ERROR);

        $out = (new BridgeImageUrlRewriter)->rewriteJson($json);
        $this->assertStringContainsString(
            'https://dev-idx-images.quantyralabs.cc/images/c8b74c4ee86a9de4f7c845fa28ecb18b/1',
            $out
        );
        $this->assertStringNotContainsString('https://idx-images.quantyralabs.cc/images/', $out);
    }
}
