<?php

declare(strict_types=1);

namespace App\Services\Mls;

use App\Enums\ListingMirrorProvider;

/**
 * Revenue impact: one mapping path for flood zone and monthly fee fields across Bridge, Spark,
 * mirror persist, search results, and comps — avoids Beaches rows missing Stellar-prefixed columns.
 */
final class ListingResoFieldResolver
{
    public function __construct(
        private readonly AssociationFeeMonthlyNormalizer $associationFeeMonthlyNormalizer,
        private readonly MlsDatasetRegistry $datasetRegistry,
    ) {}

    public function providerForDatasetSlug(string $datasetSlug): ListingMirrorProvider
    {
        return match ($this->datasetRegistry->provider($datasetSlug)) {
            'spark' => ListingMirrorProvider::Spark,
            default => ListingMirrorProvider::Bridge,
        };
    }

    /**
     * @param  array<string, mixed>  $row
     */
    public function resolveFloodZoneCode(array $row, ListingMirrorProvider $provider, string $datasetUpper): ?string
    {
        $spark = $row['Location_sp_and_sp_Legal_co_Flood_sp_Zone2'] ?? null;
        $bridgeExt = $row[strtoupper($datasetUpper).'_FloodZoneCode'] ?? null;
        $generic = $row['FloodZoneCode'] ?? null;

        $value = match ($provider) {
            ListingMirrorProvider::Spark => $spark ?? $generic,
            ListingMirrorProvider::Bridge => $bridgeExt ?? $generic ?? $spark,
        };

        return is_string($value) && trim($value) !== '' ? trim($value) : null;
    }

    /**
     * @param  array<string, mixed>  $row
     */
    public function resolveEstimatedTotalMonthlyFees(array $row, ListingMirrorProvider $provider, string $datasetUpper): ?float
    {
        if ($provider === ListingMirrorProvider::Bridge) {
            $extFeesField = strtoupper($datasetUpper).'_TotalMonthlyFees';
            $extFees = $row[$extFeesField] ?? null;
            if (is_numeric($extFees) && (float) $extFees > 0) {
                return round((float) $extFees, 2);
            }
        }

        $fromAssociation = $this->associationFeeMonthlyNormalizer->fromResoRow(
            $row,
            $provider === ListingMirrorProvider::Spark,
        );

        return $fromAssociation !== null && $fromAssociation > 0 ? $fromAssociation : null;
    }

    /**
     * @param  array<string, mixed>  $row
     */
    public function resolveFloodZoneCodeForDataset(array $row, string $datasetSlug): ?string
    {
        return $this->resolveFloodZoneCode(
            $row,
            $this->providerForDatasetSlug($datasetSlug),
            strtoupper($datasetSlug),
        );
    }

    /**
     * @param  array<string, mixed>  $row
     */
    public function resolveEstimatedTotalMonthlyFeesForDataset(array $row, string $datasetSlug): ?float
    {
        return $this->resolveEstimatedTotalMonthlyFees(
            $row,
            $this->providerForDatasetSlug($datasetSlug),
            strtoupper($datasetSlug),
        );
    }
}
