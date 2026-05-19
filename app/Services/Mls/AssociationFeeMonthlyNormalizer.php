<?php

declare(strict_types=1);

namespace App\Services\Mls;

/**
 * Revenue impact: Beaches (Spark) association fees arrive on mixed billing cycles;
 * normalizing to a monthly total powers search/comps filters on estimated_total_monthly_fees.
 */
final class AssociationFeeMonthlyNormalizer
{
    private const string SPARK_ESTIMATED_MONTHLY_FALLBACK = 'Financial_sp_Information_co_Estimated_sp_Monthly_sp_Assoc_sp_Recurring_sp_Fee3';

    /**
     * @var array<string, float>
     */
    private const array MONTHLY_MULTIPLIERS = [
        'Monthly' => 1.0,
        'Annually' => 1 / 12,
        'Semi-Annually' => 1 / 6,
        'Quarterly' => 1 / 3,
        'Weekly' => 52 / 12,
        'Daily' => 365 / 12,
        'One Time' => 0.0,
    ];

    /**
     * @param  array<string, mixed>  $row
     */
    public function fromResoRow(array $row, bool $allowDeclaredFallback = false): ?float
    {
        $sum = round(
            $this->componentMonthly($row['AssociationFee'] ?? null, $row['AssociationFeeFrequency'] ?? null)
            + $this->componentMonthly($row['AssociationFee2'] ?? null, $row['AssociationFee2Frequency'] ?? null),
            2,
        );

        if ($sum > 0) {
            return $sum;
        }

        if (! $allowDeclaredFallback) {
            return null;
        }

        $fallback = $row[self::SPARK_ESTIMATED_MONTHLY_FALLBACK] ?? null;

        return is_numeric($fallback) && (float) $fallback > 0
          ? round((float) $fallback, 2)
          : null;
    }

    private function componentMonthly(mixed $amount, mixed $frequency): float
    {
        if (! is_numeric($amount)) {
            return 0.0;
        }

        if (! is_string($frequency) || trim($frequency) === '') {
            return 0.0;
        }

        $multiplier = self::MONTHLY_MULTIPLIERS[trim($frequency)] ?? null;
        if ($multiplier === null) {
            return 0.0;
        }

        return round((float) $amount * $multiplier, 2);
    }
}
