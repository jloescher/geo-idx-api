<?php

namespace Tests\Unit\Services\Mls;

use App\Services\Mls\MlsFeedResolver;
use Symfony\Component\HttpKernel\Exception\HttpException;
use Tests\TestCase;

class MlsFeedResolverNormalizationTest extends TestCase
{
    public function test_normalize_wire_slug_to_internal_bridge_key(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami']]);

        $feeds = app(MlsFeedResolver::class);

        $this->assertSame('bridge_stellar', $feeds->normalizeWireDatasetToCatalogKey('stellar'));
        $this->assertSame('bridge_miami', $feeds->normalizeWireDatasetToCatalogKey('miami'));
        $this->assertSame('bridge_stellar', $feeds->normalizeWireDatasetToCatalogKey('bridge_stellar'));
    }

    public function test_normalize_rejects_unknown_feed(): void
    {
        config(['bridge.datasets' => ['stellar']]);

        $feeds = app(MlsFeedResolver::class);

        $this->expectException(HttpException::class);
        $feeds->normalizeWireDatasetToCatalogKey('unknown_mls');
    }
}
