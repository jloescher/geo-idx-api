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
}
