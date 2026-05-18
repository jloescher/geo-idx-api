<?php

namespace Tests\Unit\Services\Mls;

use App\Enums\MlsProvider;
use App\Services\Mls\MlsFeedResolver;
use Symfony\Component\HttpKernel\Exception\HttpException;
use Tests\TestCase;

class MlsFeedResolverSparkTest extends TestCase
{
    private MlsFeedResolver $resolver;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'bridge.datasets' => ['stellar'],
            'spark.datasets' => ['beaches'],
        ]);

        $this->resolver = app(MlsFeedResolver::class);
    }

    public function test_catalog_includes_spark_beaches_feed(): void
    {
        $codes = $this->resolver->catalogFeedCodes();

        $this->assertContains('spark_beaches', $codes);
        $this->assertContains('bridge_stellar', $codes);
    }

    public function test_normalize_beaches_wire_value_to_spark_catalog_key(): void
    {
        $this->assertSame('spark_beaches', $this->resolver->normalizeWireDatasetToCatalogKey('beaches'));
        $this->assertSame('spark_beaches', $this->resolver->normalizeWireDatasetToCatalogKey('spark_beaches'));
    }

    public function test_mirror_dataset_slug_for_spark_is_unprefixed(): void
    {
        $this->assertSame('beaches', $this->resolver->mirrorDatasetSlug('spark_beaches'));
    }

    public function test_provider_for_spark_feed(): void
    {
        $this->assertSame(MlsProvider::SPARK, $this->resolver->providerForFeedCode('spark_beaches'));
    }

    public function test_feed_label_for_dashboard(): void
    {
        $this->assertSame('Beaches MLS (Spark)', $this->resolver->feedLabel('spark_beaches'));
    }

    public function test_unknown_feed_throws(): void
    {
        $this->expectException(HttpException::class);

        $this->resolver->normalizeWireDatasetToCatalogKey('unknown_mls');
    }
}
