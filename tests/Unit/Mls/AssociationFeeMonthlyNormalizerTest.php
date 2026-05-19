<?php

namespace Tests\Unit\Mls;

use App\Services\Mls\AssociationFeeMonthlyNormalizer;
use PHPUnit\Framework\Attributes\DataProvider;
use Tests\TestCase;

class AssociationFeeMonthlyNormalizerTest extends TestCase
{
    private AssociationFeeMonthlyNormalizer $normalizer;

    protected function setUp(): void
    {
        parent::setUp();
        $this->normalizer = new AssociationFeeMonthlyNormalizer;
    }

    #[DataProvider('frequencyProvider')]
    public function test_converts_single_fee_by_frequency(string $frequency, float $fee, float $expectedMonthly): void
    {
        $result = $this->normalizer->fromResoRow([
            'AssociationFee' => $fee,
            'AssociationFeeFrequency' => $frequency,
        ]);

        $this->assertSame($expectedMonthly, $result);
    }

    /**
     * @return array<string, array{0: string, 1: float, 2: float}>
     */
    public static function frequencyProvider(): array
    {
        return [
            'monthly' => ['Monthly', 500.0, 500.0],
            'annually' => ['Annually', 1200.0, 100.0],
            'semi_annually' => ['Semi-Annually', 600.0, 100.0],
            'quarterly' => ['Quarterly', 300.0, 100.0],
            'weekly' => ['Weekly', 100.0, round(100.0 * 52 / 12, 2)],
            'daily' => ['Daily', 10.0, round(10.0 * 365 / 12, 2)],
        ];
    }

    public function test_one_time_fee_is_excluded(): void
    {
        $this->assertNull($this->normalizer->fromResoRow([
            'AssociationFee' => 999.0,
            'AssociationFeeFrequency' => 'One Time',
        ]));
    }

    public function test_null_frequency_with_numeric_fee_is_excluded(): void
    {
        $this->assertNull($this->normalizer->fromResoRow([
            'AssociationFee' => 312.54,
            'AssociationFeeFrequency' => null,
        ]));
    }

    public function test_dual_fee_sum_monthly_and_quarterly(): void
    {
        $result = $this->normalizer->fromResoRow([
            'AssociationFee' => 300.0,
            'AssociationFeeFrequency' => 'Monthly',
            'AssociationFee2' => 300.0,
            'AssociationFee2Frequency' => 'Quarterly',
        ]);

        $this->assertSame(400.0, $result);
    }

    public function test_beaches_sample_first_row_monthly_fee_only(): void
    {
        $result = $this->normalizer->fromResoRow([
            'AssociationFee' => 500.22,
            'AssociationFeeFrequency' => 'Monthly',
            'AssociationFee2' => 312.54,
            'AssociationFee2Frequency' => null,
        ]);

        $this->assertSame(500.22, $result);
    }

    public function test_spark_declared_fallback_when_association_sum_empty(): void
    {
        $result = $this->normalizer->fromResoRow([
            'Financial_sp_Information_co_Estimated_sp_Monthly_sp_Assoc_sp_Recurring_sp_Fee3' => 813.0,
        ], allowDeclaredFallback: true);

        $this->assertSame(813.0, $result);
    }

    public function test_spark_declared_fallback_not_used_without_flag(): void
    {
        $result = $this->normalizer->fromResoRow([
            'Financial_sp_Information_co_Estimated_sp_Monthly_sp_Assoc_sp_Recurring_sp_Fee3' => 813.0,
        ], allowDeclaredFallback: false);

        $this->assertNull($result);
    }

    public function test_all_null_returns_null(): void
    {
        $this->assertNull($this->normalizer->fromResoRow([]));
    }
}
