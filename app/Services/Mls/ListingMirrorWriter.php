<?php

declare(strict_types=1);

namespace App\Services\Mls;

use App\Enums\ListingMirrorProvider;
use App\Models\Listing;
use App\Services\Bridge\BridgeReplicaPersistStats;
use Carbon\CarbonImmutable;
use Illuminate\Support\Facades\DB;

/**
 * Revenue impact: shared mirror persist keeps Bridge and Spark replication aligned on
 * Active/Pending scope and PostGIS indexing without duplicating batch upsert logic.
 */
final class ListingMirrorWriter
{
    public function __construct(
        private readonly ListingResoFieldResolver $resoFieldResolver,
    ) {}

    /**
     * @var list<string>
     */
    private const UPSERT_UPDATE_COLUMNS = [
        'mls_listing_id',
        'standard_status',
        'list_price',
        'bedrooms_total',
        'bathrooms_total_decimal',
        'living_area',
        'lot_size_acres',
        'year_built',
        'stories_total',
        'city',
        'county_or_parish',
        'postal_code',
        'state_or_province',
        'property_type',
        'property_sub_type',
        'on_market_date',
        'close_date',
        'modification_timestamp',
        'bridge_modification_timestamp',
        'price_change_timestamp',
        'previous_list_price',
        'flood_zone_code',
        'estimated_total_monthly_fees',
        'latitude',
        'longitude',
        'waterfront_yn',
        'pool_private_yn',
        'dock_yn',
        'new_construction_yn',
        'garage_yn',
        'association_yn',
        'spa_yn',
        'fireplace_yn',
        'senior_community_yn',
        'subdivision_name',
        'elementary_school',
        'middle_or_junior_school',
        'high_school',
        'special_listing_conditions',
        'raw_data',
        'custom_fields',
        'street_number',
        'street_name',
        'list_agent_mls_id',
        'list_office_mls_id',
        'updated_at',
    ];

    /**
     * @param  list<array<string, mixed>>  $rows
     */
    public function hydrateReplicaBatch(
        string $dataset,
        array $rows,
        ListingMirrorProvider $provider = ListingMirrorProvider::Bridge,
    ): BridgeReplicaPersistStats {
        $startedAt = hrtime(true);
        $chunkSize = $this->upsertChunkSize($provider);
        $pendingUpserts = [];
        $coords = [];
        /** @var array<string, array<string, bool>> $deletesGrouped */
        $deletesGrouped = [];
        /** @var array<string, array<string, bool>> $nullCoordKeys */
        $nullCoordKeys = [];
        $upserted = 0;
        $deleted = 0;
        $skipped = 0;

        foreach ($rows as $row) {
            if (! is_array($row)) {
                $skipped++;

                continue;
            }

            $listingKey = isset($row['ListingKey']) && is_string($row['ListingKey']) ? $row['ListingKey'] : null;
            if ($listingKey === null || $listingKey === '') {
                $skipped++;

                continue;
            }

            $datasetSlug = $this->datasetSlugFromListingKey($listingKey, $dataset);

            $statusNormalized = strtolower(trim((string) ($row['StandardStatus'] ?? '')));
            if (! in_array($statusNormalized, ['active', 'pending'], true)) {
                if (! isset($deletesGrouped[$datasetSlug])) {
                    $deletesGrouped[$datasetSlug] = [];
                }
                $deletesGrouped[$datasetSlug][$listingKey] = true;
                $deleted++;

                continue;
            }

            $coordsPair = $this->resolveLanLng($row);
            $lat = $coordsPair['lat'];
            $lng = $coordsPair['lng'];

            $payload = $this->buildUpsertPayload($dataset, $datasetSlug, $listingKey, $row, $lat, $lng, $provider);
            $pendingUpserts[] = $payload;
            $upserted++;

            if ($lng !== null && $lat !== null && is_finite($lat) && is_finite($lng)) {
                $coords[] = [$datasetSlug, $listingKey, $lng, $lat];
            } else {
                if (! isset($nullCoordKeys[$datasetSlug])) {
                    $nullCoordKeys[$datasetSlug] = [];
                }
                $nullCoordKeys[$datasetSlug][$listingKey] = true;
            }

            if (count($pendingUpserts) >= $chunkSize) {
                $this->flushReplicaSegment($pendingUpserts, $coords, $nullCoordKeys, $provider);
                $pendingUpserts = [];
                $coords = [];
                $nullCoordKeys = [];
            }
        }

        $this->flushReplicaSegment($pendingUpserts, $coords, $nullCoordKeys, $provider);
        $this->flushDeletes($deletesGrouped, $provider);

        $durationMs = (int) ((hrtime(true) - $startedAt) / 1_000_000);

        return new BridgeReplicaPersistStats(
            rowsReceived: count($rows),
            upserted: $upserted,
            deleted: $deleted,
            skipped: $skipped,
            durationMs: $durationMs,
        );
    }

    private function upsertChunkSize(ListingMirrorProvider $provider): int
    {
        return match ($provider) {
            ListingMirrorProvider::Spark => (int) config('spark.sync_upsert_chunk_size', 250),
            ListingMirrorProvider::Bridge => (int) config('bridge.sync_upsert_chunk_size', 250),
        };
    }

    /**
     * @param  list<array<string, mixed>>  $pendingUpserts
     * @param  list<array{0: string, 1: string, 2: float, 3: float}>  $coords
     * @param  array<string, array<string, bool>>  $nullCoordKeys
     */
    private function flushReplicaSegment(
        array $pendingUpserts,
        array $coords,
        array $nullCoordKeys,
        ListingMirrorProvider $provider,
    ): void {
        if ($pendingUpserts === [] && $coords === [] && $nullCoordKeys === []) {
            return;
        }

        DB::transaction(function () use ($pendingUpserts, $coords, $nullCoordKeys, $provider): void {
            $this->flushUpsertChunk($pendingUpserts);
            $this->flushCoordinates($coords, $provider);
            $this->flushNullCoordinates($nullCoordKeys, $provider);
        });
    }

    /**
     * @param  list<array<string, mixed>>  $records
     */
    private function flushUpsertChunk(array $records): void
    {
        if ($records === []) {
            return;
        }

        $now = now();
        foreach ($records as &$record) {
            $record['updated_at'] = $now;
            $record['created_at'] = $record['created_at'] ?? $now;
        }
        unset($record);

        DB::table('listings')->upsert(
            $this->encodeJsonColumnsForUpsert($records),
            ['dataset_slug', 'listing_key'],
            self::UPSERT_UPDATE_COLUMNS,
        );
    }

    /**
     * @param  list<array<string, mixed>>  $records
     * @return list<array<string, mixed>>
     */
    private function encodeJsonColumnsForUpsert(array $records): array
    {
        foreach ($records as &$record) {
            foreach (['raw_data', 'custom_fields', 'special_listing_conditions'] as $column) {
                if (isset($record[$column]) && is_array($record[$column])) {
                    $record[$column] = json_encode($record[$column], JSON_THROW_ON_ERROR);
                }
            }
        }
        unset($record);

        return $records;
    }

    /**
     * @param  list<array{0: string, 1: string, 2: float, 3: float}>  $pairs
     */
    private function flushCoordinates(array $pairs, ListingMirrorProvider $provider): void
    {
        if ($pairs === []) {
            return;
        }

        $chunkSize = $this->upsertChunkSize($provider);

        foreach (array_chunk($pairs, $chunkSize) as $segment) {
            $sqlParts = [];
            $bindings = [];
            foreach ($segment as [$ds, $key, $lng, $lat]) {
                $sqlParts[] = '(?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography)';
                array_push($bindings, $ds, $key, $lng, $lat);
            }
            $valuesSql = implode(',', $sqlParts);
            $updatedAt = now();
            DB::statement(
                "UPDATE listings AS l SET coordinates = v.geom, updated_at = ? FROM (VALUES {$valuesSql}) AS v(ds,k,geom) WHERE l.dataset_slug = v.ds AND l.listing_key = v.k",
                array_merge([$updatedAt], $bindings),
            );
        }
    }

    /**
     * @param  array<string, array<string, bool>>  $grouped
     */
    private function flushNullCoordinates(array $grouped, ListingMirrorProvider $provider): void
    {
        if ($grouped === []) {
            return;
        }

        $chunkSize = $this->upsertChunkSize($provider);

        foreach ($grouped as $datasetSlug => $keysMap) {
            $keys = array_keys($keysMap);
            foreach (array_chunk($keys, $chunkSize) as $segment) {
                Listing::query()
                    ->where('dataset_slug', $datasetSlug)
                    ->whereIn('listing_key', $segment, 'and', false)
                    ->update([
                        'coordinates' => null,
                        'updated_at' => now(),
                    ]);
            }
        }
    }

    /**
     * @param  array<string, array<string, bool>>  $groupedNormalized
     */
    private function flushDeletes(array $groupedNormalized, ListingMirrorProvider $provider): void
    {
        if ($groupedNormalized === []) {
            return;
        }

        $chunkSize = $this->upsertChunkSize($provider);

        foreach ($groupedNormalized as $datasetSlug => $keysMap) {
            $keys = array_keys($keysMap);
            foreach (array_chunk($keys, $chunkSize) as $segment) {
                if ($segment === []) {
                    continue;
                }

                $placeholders = implode(',', array_fill(0, count($segment), '(?,?)'));
                $bindings = [];
                foreach ($segment as $listingKey) {
                    $bindings[] = $datasetSlug;
                    $bindings[] = $listingKey;
                }

                DB::delete(
                    "DELETE FROM listings WHERE (dataset_slug, listing_key) IN ({$placeholders})",
                    $bindings,
                );
            }
        }
    }

    /**
     * @param  array<string, mixed>  $row
     * @return array<string, mixed>
     */
    private function buildUpsertPayload(
        string $datasetUpper,
        string $datasetSlug,
        string $listingKey,
        array $row,
        ?float $lat,
        ?float $lng,
        ListingMirrorProvider $provider,
    ): array {
        $custom = match ($provider) {
            ListingMirrorProvider::Spark => $this->extractSparkExtensionFields($row),
            ListingMirrorProvider::Bridge => $this->extractBridgeExtensionFields($row, strtoupper($datasetUpper)),
        };

        $datasetUpperNorm = strtoupper($datasetUpper);
        $floodZoneCode = $this->resoFieldResolver->resolveFloodZoneCode($row, $provider, $datasetUpperNorm);
        $estimatedTotalMonthlyFees = $this->resoFieldResolver->resolveEstimatedTotalMonthlyFees($row, $provider, $datasetUpperNorm);

        $conditions = null;
        if (isset($row['SpecialListingConditions']) && is_array($row['SpecialListingConditions'])) {
            $conditions = array_values(array_filter(
                array_map(static fn ($c) => is_string($c) ? trim($c) : null, $row['SpecialListingConditions']),
                static fn (?string $x) => $x !== null && $x !== '',
            ));
        }

        $now = now();

        return [
            'dataset_slug' => $datasetSlug,
            'listing_key' => $listingKey,
            'mls_listing_id' => is_string($row['ListingId'] ?? null) ? $row['ListingId'] : (is_numeric($row['ListingId'] ?? null) ? (string) $row['ListingId'] : null),
            'standard_status' => is_string($row['StandardStatus'] ?? null) ? $row['StandardStatus'] : null,
            'list_price' => $this->num($row['ListPrice'] ?? null),
            'bedrooms_total' => $this->intOrNull($row['BedroomsTotal'] ?? null),
            'bathrooms_total_decimal' => $this->bathroomsTotal($row),
            'living_area' => $this->livingArea($row),
            'lot_size_acres' => $this->num($row['LotSizeAcres'] ?? null),
            'year_built' => $this->intOrNull($row['YearBuilt'] ?? null),
            'stories_total' => $this->intOrNull($row['StoriesTotal'] ?? null),
            'city' => $this->str($row['City'] ?? null),
            'county_or_parish' => $this->str($row['CountyOrParish'] ?? null),
            'postal_code' => $this->str($row['PostalCode'] ?? null),
            'state_or_province' => $this->str($row['StateOrProvince'] ?? null),
            'property_type' => $this->str($row['PropertyType'] ?? null),
            'property_sub_type' => $this->str($row['PropertySubType'] ?? null),
            'on_market_date' => $this->dateStr($row['OnMarketDate'] ?? null),
            'close_date' => $this->dateStr($row['CloseDate'] ?? null),
            'modification_timestamp' => $this->toSqlTimestamp($row['ModificationTimestamp'] ?? null),
            'bridge_modification_timestamp' => $provider === ListingMirrorProvider::Bridge
                ? $this->toSqlTimestamp($row['BridgeModificationTimestamp'] ?? null)
                : null,
            'price_change_timestamp' => $this->toSqlTimestamp($row['PriceChangeTimestamp'] ?? null),
            'previous_list_price' => $this->num($row['PreviousListPrice'] ?? null),
            'flood_zone_code' => $floodZoneCode,
            'estimated_total_monthly_fees' => $estimatedTotalMonthlyFees,
            'latitude' => $lat,
            'longitude' => $lng,
            'waterfront_yn' => $this->boolOrNull($row['WaterfrontYN'] ?? null),
            'pool_private_yn' => $this->boolOrNull($row['PoolPrivateYN'] ?? null),
            'dock_yn' => $this->boolOrNull($row['DockYN'] ?? null),
            'new_construction_yn' => $this->boolOrNull($row['NewConstructionYN'] ?? null),
            'garage_yn' => $this->boolOrNull($row['GarageYN'] ?? null),
            'association_yn' => $this->boolOrNull($row['AssociationYN'] ?? null),
            'spa_yn' => $this->boolOrNull($row['SpaYN'] ?? null),
            'fireplace_yn' => $this->boolOrNull($row['FireplaceYN'] ?? null),
            'senior_community_yn' => $this->boolOrNull($row['SeniorCommunityYN'] ?? null),
            'subdivision_name' => $this->str($row['SubdivisionName'] ?? null),
            'elementary_school' => $this->str($row['ElementarySchool'] ?? null),
            'middle_or_junior_school' => $this->str($row['MiddleOrJuniorSchool'] ?? null),
            'high_school' => $this->str($row['HighSchool'] ?? null),
            'special_listing_conditions' => $conditions ?? [],
            'raw_data' => $row,
            'custom_fields' => $custom,
            'street_number' => $this->str($row['StreetNumber'] ?? null),
            'street_name' => $this->str($row['StreetName'] ?? null),
            'list_agent_mls_id' => $this->str($row['ListAgentMlsId'] ?? null),
            'list_office_mls_id' => $this->str($row['ListOfficeMlsId'] ?? null),
            'created_at' => $now,
            'updated_at' => $now,
        ];
    }

    /**
     * @param  array<string, mixed>  $row
     */
    private function bathroomsTotal(array $row): ?float
    {
        $decimal = $this->num($row['BathroomsTotalDecimal'] ?? null);
        if ($decimal !== null) {
            return $decimal;
        }

        return $this->num($row['BathroomsTotalInteger'] ?? null);
    }

    /**
     * @param  array<string, mixed>  $row
     */
    private function livingArea(array $row): ?int
    {
        $area = $row['LivingArea'] ?? null;
        if (! is_numeric($area)) {
            $area = $row['BuildingAreaTotal'] ?? null;
        }
        if (! is_numeric($area)) {
            return null;
        }

        return (int) round((float) $area);
    }

    private function toSqlTimestamp(mixed $v): ?string
    {
        if (! is_string($v) || trim($v) === '') {
            return null;
        }
        try {
            return CarbonImmutable::parse($v)->utc()->format('Y-m-d H:i:s.u');
        } catch (\Throwable) {
            return null;
        }
    }

    /**
     * @param  array<string, mixed>  $row
     * @return array{lat: ?float, lng: ?float}
     */
    private function resolveLanLng(array $row): array
    {
        $lat = null;
        $lng = null;
        if (isset($row['Coordinates']) && is_array($row['Coordinates'])) {
            $pair = $row['Coordinates']['coordinates'] ?? null;
            if (is_array($pair) && count($pair) >= 2) {
                $lng = is_numeric($pair[0]) ? (float) $pair[0] : null;
                $lat = is_numeric($pair[1]) ? (float) $pair[1] : null;
            }
        }
        if (($lat === null || $lng === null) && isset($row['Latitude'], $row['Longitude'])) {
            $lat = is_numeric($row['Latitude']) ? (float) $row['Latitude'] : $lat;
            $lng = is_numeric($row['Longitude']) ? (float) $row['Longitude'] : $lng;
        }

        return ['lat' => $lat, 'lng' => $lng];
    }

    private function datasetSlugFromListingKey(string $listingKey, string $fallbackDataset): string
    {
        $idx = strpos($listingKey, ':');
        if ($idx !== false && $idx > 0) {
            return strtolower(substr($listingKey, 0, $idx));
        }

        return strtolower($fallbackDataset);
    }

    /**
     * @return array<string, mixed>
     */
    private function extractBridgeExtensionFields(array $record, string $datasetUpper): array
    {
        $prefix = $datasetUpper.'_';
        $out = [];
        foreach ($record as $k => $v) {
            if (is_string($k) && str_starts_with(strtoupper($k), $prefix)) {
                $out[$k] = $v;
            }
        }

        return $out;
    }

    /**
     * Spark / Beaches encoded MLS fields on the Property root (e.g. Room_sp_Types_co_*).
     *
     * @return array<string, mixed>
     */
    private function extractSparkExtensionFields(array $record): array
    {
        $standard = $this->standardResoFieldNames();
        $out = [];
        foreach ($record as $k => $v) {
            if (! is_string($k)) {
                continue;
            }
            if (in_array($k, $standard, true)) {
                continue;
            }
            if (in_array($k, ['Media', 'Room', 'Unit', 'OpenHouse', '@odata.id'], true)) {
                continue;
            }
            if (str_contains($k, '_sp_') || str_contains($k, '_co_')) {
                $out[$k] = $v;
            }
        }

        return $out;
    }

    /**
     * @return list<string>
     */
    private function standardResoFieldNames(): array
    {
        return [
            'ListingKey', 'ListingId', 'StandardStatus', 'ListPrice', 'ClosePrice', 'PreviousListPrice',
            'PriceChangeTimestamp', 'ModificationTimestamp', 'BridgeModificationTimestamp',
            'BedroomsTotal', 'BathroomsTotalDecimal', 'BathroomsTotalInteger', 'LivingArea', 'BuildingAreaTotal', 'LotSizeAcres',
            'AssociationFee', 'AssociationFeeFrequency', 'AssociationFee2', 'AssociationFee2Frequency',
            'YearBuilt', 'StoriesTotal', 'City', 'CountyOrParish', 'PostalCode', 'StateOrProvince',
            'PropertyType', 'PropertySubType', 'OnMarketDate', 'CloseDate', 'Latitude', 'Longitude',
            'Coordinates', 'WaterfrontYN', 'PoolPrivateYN', 'DockYN', 'NewConstructionYN', 'GarageYN',
            'AssociationYN', 'SpaYN', 'FireplaceYN', 'SeniorCommunityYN', 'SpecialListingConditions',
            'SubdivisionName', 'ElementarySchool', 'MiddleOrJuniorSchool', 'HighSchool',
            'StreetNumber', 'StreetName', 'ListAgentMlsId', 'ListOfficeMlsId', 'MlsStatus',
        ];
    }

    private function str(mixed $v): ?string
    {
        if (! is_string($v)) {
            return null;
        }
        $t = trim($v);

        return $t === '' ? null : $t;
    }

    private function num(mixed $v): ?float
    {
        return is_numeric($v) ? (float) $v : null;
    }

    private function intOrNull(mixed $v): ?int
    {
        return is_numeric($v) ? (int) $v : null;
    }

    private function boolOrNull(mixed $v): ?bool
    {
        return is_bool($v) ? $v : null;
    }

    private function dateStr(mixed $v): ?string
    {
        if (! is_string($v) || trim($v) === '') {
            return null;
        }
        try {
            return CarbonImmutable::parse($v)->format('Y-m-d');
        } catch (\Throwable) {
            return null;
        }
    }
}
