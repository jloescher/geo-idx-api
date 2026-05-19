<?php

namespace Tests\Unit\Mls;

use App\Enums\ListingMirrorProvider;
use App\Services\Mls\AssociationFeeMonthlyNormalizer;
use App\Services\Mls\ListingResoFieldResolver;
use App\Services\Mls\MlsDatasetRegistry;
use Tests\TestCase;

class ListingResoFieldResolverTest extends TestCase
{
    private ListingResoFieldResolver $resolver;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'mls.datasets' => [
                'stellar' => ['provider' => 'bridge', 'enabled' => true],
                'beaches' => ['provider' => 'spark', 'enabled' => true],
            ],
        ]);

        $this->resolver = new ListingResoFieldResolver(
            new AssociationFeeMonthlyNormalizer,
            new MlsDatasetRegistry,
        );
    }

    public function test_spark_flood_zone_from_location_field(): void
    {
        $code = $this->resolver->resolveFloodZoneCode(
            ['Location_sp_and_sp_Legal_co_Flood_sp_Zone2' => 'X'],
            ListingMirrorProvider::Spark,
            'BEACHES',
        );

        $this->assertSame('X', $code);
    }

    public function test_bridge_flood_zone_prefers_dataset_extension(): void
    {
        $code = $this->resolver->resolveFloodZoneCode(
            [
                'STELLAR_FloodZoneCode' => 'AE',
                'FloodZoneCode' => 'X',
            ],
            ListingMirrorProvider::Bridge,
            'STELLAR',
        );

        $this->assertSame('AE', $code);
    }

    public function test_bridge_monthly_fees_from_total_monthly_fees_field(): void
    {
        $fees = $this->resolver->resolveEstimatedTotalMonthlyFees(
            ['STELLAR_TotalMonthlyFees' => 425.5],
            ListingMirrorProvider::Bridge,
            'STELLAR',
        );

        $this->assertSame(425.5, $fees);
    }

    public function test_spark_monthly_fees_from_association_pairs(): void
    {
        $fees = $this->resolver->resolveEstimatedTotalMonthlyFees(
            [
                'AssociationFee' => 500.22,
                'AssociationFeeFrequency' => 'Monthly',
                'AssociationFee2' => 312.54,
                'AssociationFee2Frequency' => null,
            ],
            ListingMirrorProvider::Spark,
            'BEACHES',
        );

        $this->assertSame(500.22, $fees);
    }

    public function test_resolve_for_dataset_slug_uses_registry_provider(): void
    {
        $fees = $this->resolver->resolveEstimatedTotalMonthlyFeesForDataset(
            [
                'AssociationFee' => 100.0,
                'AssociationFeeFrequency' => 'Monthly',
            ],
            'beaches',
        );

        $this->assertSame(100.0, $fees);
    }
}
