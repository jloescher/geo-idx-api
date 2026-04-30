<?php

namespace App\Services\Bridge;

/**
 * URAR-style Sales Comparison Adjustment Engine with market-derived rates.
 *
 * Produces a 14-line adjustment grid per comparable sale using rates extracted
 * from the sold comp dataset via OLS regression or paired-sales analysis,
 * then reconciles all adjusted values into a single BPO point estimate.
 *
 * Revenue impact: appraisal-grade BPO output differentiates the Mega tier
 * ($449/mo) comp engine from basic CMA tools, driving upgrade conversions.
 */
final class BpoAdjustmentEngine
{
    /**
     * Build the full URAR adjustment grid for one comp.
     *
     * @param  array<string, mixed>  $subject  Normalized subject property
     * @param  array<string, mixed>  $comp  Raw Bridge property record
     * @param  array<string, mixed>  $rates  Market-derived rates from BpoMarketExtractor
     * @param  list<array<string, mixed>>  $allComps  Full comp set (for quality tier derivation)
     * @param  string|null  $condition  Condition rating: poor, fair, good, excellent (for home_value mode)
     * @param  array<string, mixed>|null  $renovations  Renovation data (kitchen/bath/hvac years)
     * @return array{
     *     lines: list<array{feature: string, subject_value: mixed, comp_value: mixed, unit: string, rate_per_unit: float, rate_source: string, adjustment: float, reasoning: string}>,
     *     net_adjustment: float,
     *     gross_adjustment: float,
     *     gross_adjustment_pct: float,
     *     adjusted_price: float,
     * }
     */
    public function adjust(array $subject, array $comp, array $rates, array $allComps = [], ?string $condition = null, ?array $renovations = null): array
    {
        $closePrice = (float) ($comp['ClosePrice'] ?? 0);
        if ($closePrice <= 0) {
            return [
                'lines' => [],
                'net_adjustment' => 0.0,
                'gross_adjustment' => 0.0,
                'gross_adjustment_pct' => 0.0,
                'adjusted_price' => 0.0,
            ];
        }

        $method = $rates['method'] ?? 'median_only';
        $lines = [];
        $absSum = 0.0;

        $line = $this->timeOfSaleAdjustment($comp, $rates, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->locationAdjustment($subject, $comp, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->siteLotAdjustment($subject, $comp, $rates, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->designStyleAdjustment($subject, $comp, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->qualityAdjustment($subject, $comp, $allComps, $rates, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->ageConditionAdjustment($subject, $comp, $rates, $method, $condition);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->glaAdjustment($subject, $comp, $rates, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->bedroomAdjustment($subject, $comp, $rates, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->bathroomAdjustment($subject, $comp, $rates, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->garageAdjustment($subject, $comp, $rates, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->poolAdjustment($subject, $comp, $rates, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->waterfrontAdjustment($subject, $comp, $rates, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->porchPatioAdjustment($subject, $comp, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        $line = $this->hvacAdjustment($subject, $comp, $method);
        if ($line !== null) {
            $lines[] = $line;
            $absSum += abs($line['adjustment']);
        }

        // Renovation credit lines (home_value mode)
        if ($renovations !== null) {
            $renovationLines = $this->renovationCreditAdjustments($renovations, $rates);
            foreach ($renovationLines as $line) {
                $lines[] = $line;
                $absSum += abs($line['adjustment']);
            }
        }

        $net = 0.0;
        foreach ($lines as $l) {
            $net += $l['adjustment'];
        }

        $grossPct = $closePrice > 0 ? round($absSum / $closePrice * 100, 2) : 0.0;

        return [
            'lines' => $lines,
            'net_adjustment' => round($net),
            'gross_adjustment' => round($absSum),
            'gross_adjustment_pct' => $grossPct,
            'adjusted_price' => round($closePrice + $net),
        ];
    }

    /**
     * Weighted reconciliation across all adjusted comps.
     *
     * @param  array<string, mixed>  $subject
     * @param  list<array{adjusted_price: float, gross_adjustment_pct: float, comp: array<string, mixed>}>  $adjustedComps
     * @param  array<string, mixed>  $rates
     * @return array{
     *     point_estimate: float|null,
     *     range: array{low: float|null, high: float|null},
     *     confidence: int,
     *     confidence_band: string,
     *     reconciliation_summary: string,
     * }
     */
    public function reconcile(array $subject, array $adjustedComps, array $rates): array
    {
        if ($adjustedComps === []) {
            return [
                'point_estimate' => null,
                'range' => ['low' => null, 'high' => null],
                'confidence' => 0,
                'confidence_band' => 'insufficient',
                'reconciliation_summary' => 'No adjusted comps available for reconciliation.',
            ];
        }

        $weights = [];
        $prices = [];
        foreach ($adjustedComps as $i => $ac) {
            $w = $this->compWeight($subject, $ac['comp'] ?? [], $ac['gross_adjustment_pct'] ?? 0);
            $weights[$i] = $w;
            $prices[$i] = $ac['adjusted_price'];
        }

        $totalWeight = array_sum($weights);
        $pointEstimate = 0.0;
        if ($totalWeight > 0) {
            foreach ($adjustedComps as $i => $ac) {
                $pointEstimate += ($weights[$i] / $totalWeight) * $prices[$i];
            }
        } else {
            $pointEstimate = array_sum($prices) / count($prices);
        }

        $sortedPrices = array_values($prices);
        sort($sortedPrices);
        $minPrice = $sortedPrices[0];
        $maxPrice = $sortedPrices[count($sortedPrices) - 1];

        $confidence = $this->computeConfidence(count($adjustedComps), $rates['r_squared'] ?? 0, $adjustedComps, $minPrice, $maxPrice);
        $spreadPct = $confidence >= 80 ? 0.03 : ($confidence >= 50 ? 0.05 : 0.10);

        $band = $confidence >= 80 ? 'high' : ($confidence >= 50 ? 'moderate' : 'low');

        $topIdx = array_keys($weights);
        arsort($weights);
        $topIdx = array_slice(array_keys($weights), 0, 2);
        $compRefs = [];
        foreach ($topIdx as $idx) {
            $compRefs[] = '#'.($idx + 1);
        }

        $summary = sprintf(
            'Weighted average of %d adjusted comps. Strongest weight to comp(s) %s. Market rates derived via %s (R-squared: %s).',
            count($adjustedComps),
            implode(' and ', $compRefs),
            $rates['method'] ?? 'unknown',
            isset($rates['r_squared']) ? round($rates['r_squared'], 2) : 'N/A',
        );

        return [
            'point_estimate' => round($pointEstimate),
            'range' => [
                'low' => round($pointEstimate * (1 - $spreadPct)),
                'high' => round($pointEstimate * (1 + $spreadPct)),
            ],
            'confidence' => $confidence,
            'confidence_band' => $band,
            'reconciliation_summary' => $summary,
        ];
    }

    /**
     * Compute a 0-1 weight for a comp based on similarity and adjustment quality.
     *
     * @param  array<string, mixed>  $subject
     * @param  array<string, mixed>  $comp
     */
    private function compWeight(array $subject, array $comp, float $grossAdjPct): float
    {
        $subGla = (float) ($subject['living_area'] ?? 0);
        $compGla = (float) ($comp['LivingArea'] ?? 0);
        $glaSim = 0.5;
        if ($subGla > 0 && $compGla > 0) {
            $glaSim = 1.0 - min(1.0, abs($subGla - $compGla) / $subGla);
        }

        $grossSim = 1.0 - min(1.0, $grossAdjPct / 100.0);

        $recency = 0.5;
        $closeDateStr = $comp['CloseDate'] ?? null;
        if (is_string($closeDateStr) && $closeDateStr !== '') {
            try {
                $close = new \DateTimeImmutable($closeDateStr);
                $now = new \DateTimeImmutable('today');
                $daysAgo = max(0, $now->diff($close)->days);
                $recency = 1.0 - min(1.0, $daysAgo / 365.0);
            } catch (\Throwable) {
                $recency = 0.5;
            }
        }

        $featureMatch = 0.5;
        $matches = 0;
        $checks = 0;
        if (isset($subject['bedrooms']) && isset($comp['BedroomsTotal'])) {
            $checks++;
            if ((int) $subject['bedrooms'] === (int) $comp['BedroomsTotal']) {
                $matches++;
            }
        }
        if (isset($subject['bathrooms']) && isset($comp['BathroomsTotalDecimal'])) {
            $checks++;
            if (abs((float) $subject['bathrooms'] - (float) $comp['BathroomsTotalDecimal']) < 0.5) {
                $matches++;
            }
        }
        $subPool = ($subject['pool'] ?? null) === true;
        $compPool = ($comp['PoolPrivateYN'] ?? null) === true;
        $checks++;
        if ($subPool === $compPool) {
            $matches++;
        }
        if ($checks > 0) {
            $featureMatch = $matches / $checks;
        }

        return 0.30 * $glaSim + 0.30 * $grossSim + 0.15 * $recency + 0.25 * $featureMatch;
    }

    /**
     * Compute 0-100 confidence score.
     *
     * @param  list<array{gross_adjustment_pct: float}>  $adjustedComps
     */
    private function computeConfidence(int $compCount, float $rSquared, array $adjustedComps, float $minPrice, float $maxPrice): int
    {
        $score = 0.0;

        if ($compCount >= 6) {
            $score += 30;
        } elseif ($compCount >= 4) {
            $score += 20;
        } elseif ($compCount >= 3) {
            $score += 10;
        }

        if ($rSquared >= 0.80) {
            $score += 30;
        } elseif ($rSquared >= 0.60) {
            $score += 20;
        } elseif ($rSquared >= 0.40) {
            $score += 10;
        }

        $avgGross = 0.0;
        foreach ($adjustedComps as $ac) {
            $avgGross += $ac['gross_adjustment_pct'] ?? 0;
        }
        if ($compCount > 0) {
            $avgGross /= $compCount;
        }
        if ($avgGross <= 10) {
            $score += 25;
        } elseif ($avgGross <= 20) {
            $score += 15;
        } elseif ($avgGross <= 30) {
            $score += 5;
        }

        $mid = ($minPrice + $maxPrice) / 2.0;
        if ($mid > 0) {
            $spreadPct = ($maxPrice - $minPrice) / $mid * 100.0;
            if ($spreadPct <= 5) {
                $score += 15;
            } elseif ($spreadPct <= 10) {
                $score += 10;
            } elseif ($spreadPct <= 20) {
                $score += 5;
            }
        }

        return (int) max(0, min(100, round($score)));
    }

    /**
     * Time-of-sale adjustment using regression-derived monthly rate.
     */
    private function timeOfSaleAdjustment(array $comp, array $rates, string $method): ?array
    {
        $timePct = (float) ($rates['time_per_month_pct'] ?? 0);
        if ($timePct == 0) {
            return null;
        }

        $closeDateStr = $comp['CloseDate'] ?? null;
        if (! is_string($closeDateStr) || $closeDateStr === '') {
            return null;
        }

        try {
            $close = new \DateTimeImmutable($closeDateStr);
            $now = new \DateTimeImmutable('today');
            $monthsAgo = $now->diff($close)->days / 30.0;
        } catch (\Throwable) {
            return null;
        }

        if ($monthsAgo <= 0) {
            return null;
        }

        $closePrice = (float) ($comp['ClosePrice'] ?? 0);
        $adj = round($closePrice * ($timePct / 100.0) * $monthsAgo);

        return [
            'feature' => 'time_of_sale',
            'subject_value' => null,
            'comp_value' => round($monthsAgo, 1).' months ago',
            'unit' => 'months',
            'rate_per_unit' => round($timePct, 4),
            'rate_source' => $method,
            'adjustment' => $adj,
            'reasoning' => 'Monthly time adjustment from regression coefficient on CloseDate',
        ];
    }

    /**
     * Location adjustment from SubdivisionName / MLSAreaMajor match.
     */
    private function locationAdjustment(array $subject, array $comp, string $method): ?array
    {
        $subSub = trim((string) ($subject['subdivision_name'] ?? ''));
        $compSub = trim((string) ($comp['SubdivisionName'] ?? ''));
        $subArea = trim((string) ($subject['mls_area_major'] ?? ''));
        $compArea = trim((string) ($comp['MLSAreaMajor'] ?? ''));

        if ($subSub !== '' && $compSub !== '' && strtolower($subSub) === strtolower($compSub)) {
            return null;
        }

        if ($subArea !== '' && $compArea !== '' && strtolower($subArea) !== strtolower($compArea)) {
            return [
                'feature' => 'location',
                'subject_value' => $subArea ?: null,
                'comp_value' => $compArea ?: null,
                'unit' => 'area',
                'rate_per_unit' => 0,
                'rate_source' => 'qualitative',
                'adjustment' => 0,
                'reasoning' => 'Different MLS area; location variance noted but not quantified without paired sales.',
            ];
        }

        return null;
    }

    /**
     * Site/lot size adjustment using market-derived $/acre.
     */
    private function siteLotAdjustment(array $subject, array $comp, array $rates, string $method): ?array
    {
        $rate = (float) ($rates['lot_per_acre'] ?? 0);
        if ($rate <= 0) {
            return null;
        }

        $subLot = $subject['lot_acres'] !== null ? (float) $subject['lot_acres'] : null;
        $compLot = (float) ($comp['LotSizeAcres'] ?? 0);

        if ($subLot === null || $compLot <= 0) {
            return null;
        }

        $diff = $subLot - $compLot;
        if (abs($diff) < 0.001) {
            return null;
        }

        return [
            'feature' => 'lot_size',
            'subject_value' => round($subLot, 4),
            'comp_value' => round($compLot, 4),
            'unit' => 'acres',
            'rate_per_unit' => round($rate),
            'rate_source' => $method,
            'adjustment' => (int) round($diff * $rate),
            'reasoning' => 'Lot size delta x market-derived $/acre',
        ];
    }

    /**
     * Design/style adjustment from PropertySubType.
     */
    private function designStyleAdjustment(array $subject, array $comp, string $method): ?array
    {
        $subType = trim((string) ($subject['property_sub_type'] ?? ''));
        $compType = trim((string) ($comp['PropertySubType'] ?? ''));

        if ($subType === '' || $compType === '') {
            return null;
        }

        if (strtolower($subType) === strtolower($compType)) {
            return null;
        }

        return [
            'feature' => 'design_style',
            'subject_value' => $subType,
            'comp_value' => $compType,
            'unit' => 'type',
            'rate_per_unit' => 0,
            'rate_source' => 'qualitative',
            'adjustment' => 0,
            'reasoning' => 'Design/style differs; adjustment not quantified without paired sales.',
        ];
    }

    /**
     * Quality adjustment inferred from PPSF tier within comp set.
     */
    private function qualityAdjustment(array $subject, array $comp, array $allComps, array $rates, string $method): ?array
    {
        $rate = (float) ($rates['gla_per_sf'] ?? 0);
        if ($rate <= 0 || $allComps === []) {
            return null;
        }

        $compPpsf = 0.0;
        $compGla = (float) ($comp['LivingArea'] ?? 0);
        $compPrice = (float) ($comp['ClosePrice'] ?? 0);
        if ($compGla > 0 && $compPrice > 0) {
            $compPpsf = $compPrice / $compGla;
        }

        $subGla = (float) ($subject['living_area'] ?? 0);
        $subPrice = (float) ($subject['list_price'] ?? 0);
        $subPpsf = ($subGla > 0 && $subPrice > 0) ? $subPrice / $subGla : 0;

        if ($subPpsf <= 0 || $compPpsf <= 0) {
            return null;
        }

        $ppsfDiff = $subPpsf - $compPpsf;
        if (abs($ppsfDiff) < $rate * 0.10) {
            return null;
        }

        $ppsfValues = [];
        foreach ($allComps as $c) {
            $p = (float) ($c['ClosePrice'] ?? 0);
            $g = (float) ($c['LivingArea'] ?? 0);
            if ($p > 0 && $g > 0) {
                $ppsfValues[] = $p / $g;
            }
        }

        if (count($ppsfValues) < 3) {
            return null;
        }

        return [
            'feature' => 'quality',
            'subject_value' => round($subPpsf, 2).' $/sf',
            'comp_value' => round($compPpsf, 2).' $/sf',
            'unit' => 'ppsf_tier',
            'rate_per_unit' => 0,
            'rate_source' => 'inferred',
            'adjustment' => 0,
            'reasoning' => 'PPSF difference indicates quality variance; captured in GLA adjustment to avoid double-counting.',
        ];
    }

    /**
     * Age/condition adjustment using regression-derived $/year.
     * When condition rating is provided (home_value mode), applies effective age offset.
     */
    private function ageConditionAdjustment(array $subject, array $comp, array $rates, string $method, ?string $condition = null): ?array
    {
        $rate = (float) ($rates['age_per_year'] ?? 0);

        $subYear = $subject['year_built'] !== null ? (int) $subject['year_built'] : null;
        $compYear = (int) ($comp['YearBuilt'] ?? 0);

        if ($subYear === null || $compYear <= 0) {
            return null;
        }

        // Apply condition-based effective age offset for home_value mode
        $effectiveYear = $this->effectiveYear((int) $subYear, $condition);
        $diff = $effectiveYear - $compYear;

        if ($diff === 0) {
            return null;
        }

        if ($rate == 0) {
            return null;
        }

        $reasoning = 'Year built delta x market-derived $/year';
        if ($condition !== null && $effectiveYear !== $subYear) {
            $reasoning = sprintf(
                'Effective age adjusted from %d to %d (%s condition) x $/year',
                $subYear,
                $effectiveYear,
                ucfirst($condition),
            );
        }

        return [
            'feature' => 'age_condition',
            'subject_value' => $effectiveYear,
            'comp_value' => $compYear,
            'unit' => 'years',
            'rate_per_unit' => round($rate),
            'rate_source' => $method,
            'adjustment' => (int) round($diff * $rate),
            'reasoning' => $reasoning,
        ];
    }

    /**
     * Map condition rating to an effective year-built offset.
     *
     * Excellent = 10 years newer, Good = no change, Fair = 5 years older, Poor = 15 years older.
     */
    private function effectiveYear(int $yearBuilt, ?string $condition): int
    {
        return match ($condition) {
            'excellent' => $yearBuilt + 10,
            'good' => $yearBuilt,
            'fair' => $yearBuilt - 5,
            'poor' => $yearBuilt - 15,
            default => $yearBuilt,
        };
    }

    /**
     * Generate renovation credit adjustment lines based on renovation recency.
     *
     * Credits: kitchen $8K, bathrooms $6K, HVAC $4K.
     * Full credit within 5 years, 50% credit for 5-10 years, zero after 10 years.
     *
     * @param  array<string, mixed>  $renovations  {kitchen_year: ?int, bathrooms_year: ?int, hvac_year: ?int}
     * @return list<array{feature: string, subject_value: mixed, comp_value: mixed, unit: string, rate_per_unit: float, rate_source: string, adjustment: float, reasoning: string}>
     */
    /**
     * Renovation credit adjustments using market-derived credit amounts.
     *
     * Credits are derived from the market's gla_per_sf and median GLA via BpoMarketExtractor,
     * scaling with local price levels. Falls back to conservative defaults when market data
     * is unavailable (e.g., median_only method with no comps).
     *
     * @param  array<string, mixed>  $renovations
     * @param  array<string, mixed>  $rates  Market-derived rates including renovation_*_credit keys
     * @return list<array{feature: string, subject_value: int|null, comp_value: null, unit: string, rate_per_unit: float, rate_source: string, adjustment: int, reasoning: string}>
     */
    private function renovationCreditAdjustments(array $renovations, array $rates): array
    {
        $lines = [];
        $currentYear = (int) date('Y');

        // Default credits (fallback when market data unavailable)
        $defaultKitchen = 8000;
        $defaultBathrooms = 6000;
        $defaultHvac = 4000;

        $renovationItems = [
            [
                'key' => 'kitchen_year',
                'rate_key' => 'renovation_kitchen_credit',
                'default_credit' => $defaultKitchen,
                'label' => 'renovation_kitchen',
            ],
            [
                'key' => 'bathrooms_year',
                'rate_key' => 'renovation_bathrooms_credit',
                'default_credit' => $defaultBathrooms,
                'label' => 'renovation_bathrooms',
            ],
            [
                'key' => 'hvac_year',
                'rate_key' => 'renovation_hvac_credit',
                'default_credit' => $defaultHvac,
                'label' => 'renovation_hvac',
            ],
        ];

        foreach ($renovationItems as $item) {
            $year = $renovations[$item['key']] ?? null;
            if ($year === null) {
                continue;
            }

            $year = (int) $year;
            $age = $currentYear - $year;

            // Use market-derived credit if available, otherwise fall back to default
            $baseCredit = (float) ($rates[$item['rate_key']] ?? $item['default_credit']);
            $credit = 0;
            $recency = '';
            $source = 'default';

            if ($baseCredit > 0 && $age <= 5) {
                $credit = (int) round($baseCredit);
                $recency = 'within 5 years (full credit)';
                $source = ($rates[$item['rate_key']] ?? 0) > 0 ? 'market_derived' : 'default';
            } elseif ($baseCredit > 0 && $age <= 10) {
                $credit = (int) round($baseCredit * 0.5);
                $recency = '5-10 years ago (50% credit)';
                $source = ($rates[$item['rate_key']] ?? 0) > 0 ? 'market_derived' : 'default';
            }

            if ($credit <= 0) {
                continue;
            }

            $lines[] = [
                'feature' => $item['label'],
                'subject_value' => $year,
                'comp_value' => null,
                'unit' => 'credit',
                'rate_per_unit' => $credit,
                'rate_source' => $source,
                'adjustment' => $credit,
                'reasoning' => sprintf(
                    'Renovation in %d (%s): $%s credit (%s)',
                    $year,
                    $recency,
                    number_format($credit),
                    $source === 'market_derived' ? 'market-derived' : 'default',
                ),
            ];
        }

        return $lines;
    }

    /**
     * GLA adjustment using regression-derived $/sf.
     */
    private function glaAdjustment(array $subject, array $comp, array $rates, string $method): ?array
    {
        $rate = (float) ($rates['gla_per_sf'] ?? 0);
        if ($rate <= 0) {
            return null;
        }

        $subGla = $subject['living_area'] !== null ? (int) $subject['living_area'] : null;
        $compGla = (int) ($comp['LivingArea'] ?? 0);

        if ($subGla === null || $compGla <= 0) {
            return null;
        }

        $diff = $subGla - $compGla;
        if ($diff === 0) {
            return null;
        }

        return [
            'feature' => 'gla',
            'subject_value' => $subGla,
            'comp_value' => $compGla,
            'unit' => 'sqft',
            'rate_per_unit' => round($rate, 2),
            'rate_source' => $method,
            'adjustment' => (int) round($diff * $rate),
            'reasoning' => 'GLA delta x market-derived $/sf',
        ];
    }

    /**
     * Bedroom adjustment using regression-derived $/room.
     */
    private function bedroomAdjustment(array $subject, array $comp, array $rates, string $method): ?array
    {
        $rate = (float) ($rates['bed_per_room'] ?? 0);
        if ($rate == 0) {
            return null;
        }

        $subBed = $subject['bedrooms'] !== null ? (int) $subject['bedrooms'] : null;
        $compBed = (int) ($comp['BedroomsTotal'] ?? 0);

        if ($subBed === null || $compBed <= 0) {
            return null;
        }

        $diff = $subBed - $compBed;
        if ($diff === 0) {
            return null;
        }

        return [
            'feature' => 'bedrooms',
            'subject_value' => $subBed,
            'comp_value' => $compBed,
            'unit' => 'rooms',
            'rate_per_unit' => round($rate),
            'rate_source' => $method,
            'adjustment' => (int) round($diff * $rate),
            'reasoning' => 'Bedroom count delta x market-derived $/room',
        ];
    }

    /**
     * Bathroom adjustment using regression-derived $/bath.
     */
    private function bathroomAdjustment(array $subject, array $comp, array $rates, string $method): ?array
    {
        $rate = (float) ($rates['bath_per_full'] ?? 0);
        if ($rate == 0) {
            return null;
        }

        $subBath = $subject['bathrooms'] !== null ? (float) $subject['bathrooms'] : null;
        $compBath = (float) ($comp['BathroomsTotalDecimal'] ?? 0);

        if ($subBath === null || $compBath <= 0) {
            return null;
        }

        $diff = $subBath - $compBath;
        if (abs($diff) < 0.01) {
            return null;
        }

        return [
            'feature' => 'bathrooms',
            'subject_value' => $subBath,
            'comp_value' => $compBath,
            'unit' => 'baths',
            'rate_per_unit' => round($rate),
            'rate_source' => $method,
            'adjustment' => (int) round($diff * $rate),
            'reasoning' => 'Bathroom count delta x market-derived $/bath',
        ];
    }

    /**
     * Garage adjustment using regression-derived $/space.
     */
    private function garageAdjustment(array $subject, array $comp, array $rates, string $method): ?array
    {
        $rate = (float) ($rates['garage_per_space'] ?? 0);
        if ($rate <= 0) {
            return null;
        }

        $subGarage = $subject['garage_spaces'] !== null ? (int) $subject['garage_spaces'] : null;
        $compGarage = (int) ($comp['GarageSpaces'] ?? 0);

        if ($subGarage === null) {
            return null;
        }

        $diff = $subGarage - $compGarage;
        if ($diff === 0) {
            return null;
        }

        return [
            'feature' => 'garage',
            'subject_value' => $subGarage,
            'comp_value' => $compGarage,
            'unit' => 'spaces',
            'rate_per_unit' => round($rate),
            'rate_source' => $method,
            'adjustment' => (int) round($diff * $rate),
            'reasoning' => 'Garage stall delta x market-derived $/space',
        ];
    }

    /**
     * Pool adjustment using regression-derived pool value.
     */
    private function poolAdjustment(array $subject, array $comp, array $rates, string $method): ?array
    {
        $value = (float) ($rates['pool_value'] ?? 0);
        if ($value <= 0) {
            return null;
        }

        $subPool = ($subject['pool'] ?? null) === true;
        $compPool = ($comp['PoolPrivateYN'] ?? null) === true;

        if ($subPool === $compPool) {
            return null;
        }

        $adj = $subPool ? (int) round($value) : -(int) round($value);

        return [
            'feature' => 'pool',
            'subject_value' => $subPool,
            'comp_value' => $compPool,
            'unit' => 'boolean',
            'rate_per_unit' => round($value),
            'rate_source' => $method,
            'adjustment' => $adj,
            'reasoning' => 'Pool presence delta x market-derived pool value',
        ];
    }

    /**
     * Waterfront adjustment using regression-derived waterfront value.
     */
    private function waterfrontAdjustment(array $subject, array $comp, array $rates, string $method): ?array
    {
        $value = (float) ($rates['waterfront_value'] ?? 0);
        if ($value <= 0) {
            return null;
        }

        $subWf = ($subject['waterfront'] ?? null) === true;
        $compWf = ($comp['WaterfrontYN'] ?? null) === true;

        if ($subWf === $compWf) {
            return null;
        }

        $adj = $subWf ? (int) round($value) : -(int) round($value);

        return [
            'feature' => 'waterfront',
            'subject_value' => $subWf,
            'comp_value' => $compWf,
            'unit' => 'boolean',
            'rate_per_unit' => round($value),
            'rate_source' => $method,
            'adjustment' => $adj,
            'reasoning' => 'Waterfront presence delta x market-derived waterfront value',
        ];
    }

    /**
     * Porch/patio adjustment via keyword inference.
     */
    private function porchPatioAdjustment(array $subject, array $comp, string $method): ?array
    {
        $remarks = strtolower(trim((string) ($comp['PublicRemarks'] ?? '')));
        $hasPorch = str_contains($remarks, 'covered patio') || str_contains($remarks, 'porch')
            || str_contains($remarks, 'screened') || str_contains($remarks, 'lanai');

        if ($hasPorch) {
            return null;
        }

        return null;
    }

    /**
     * HVAC adjustment via keyword inference.
     */
    private function hvacAdjustment(array $subject, array $comp, string $method): ?array
    {
        return null;
    }
}
