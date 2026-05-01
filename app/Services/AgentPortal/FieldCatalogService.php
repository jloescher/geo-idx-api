<?php

namespace App\Services\AgentPortal;

use App\Models\FieldMappingAdapter;
use App\Models\MlsFieldCatalog;
use App\Services\AgentPortal\Contracts\LookupProviderInterface;

final class FieldCatalogService
{
    private const RESO_CANONICAL_MAP = [
        'ListPrice' => 'property.list_price',
        'BedroomsTotal' => 'property.bedrooms_total',
        'BathroomsTotalDecimal' => 'property.bathrooms_total',
        'City' => 'location.city',
        'StateOrProvince' => 'location.state',
        'PostalCode' => 'location.postal_code',
        'StreetNumber' => 'location.street_number',
        'StreetName' => 'location.street_name',
        'StandardStatus' => 'listing.status',
        'PropertyType' => 'property.type',
        'PropertySubType' => 'property.subtype',
        'LivingArea' => 'property.living_area',
        'LotSizeArea' => 'property.lot_size_area',
        'YearBuilt' => 'property.year_built',
        'OnMarketDate' => 'dates.on_market',
        'CloseDate' => 'dates.close_date',
        'StatusChangeTimestamp' => 'dates.status_change',
        'Latitude' => 'property.latitude',
        'Longitude' => 'property.longitude',
        'PublicRemarks' => 'listing.public_remarks',
        'ListingKey' => 'listing.key',
        'AssociationFee' => 'association.hoa_fee',
        'GarageSpaces' => 'features.garage_spaces',
        'PoolFeatures' => 'features.pool',
        'FireplacesTotal' => 'features.fireplaces',
        'Stories' => 'features.stories',
        'RoofLines' => 'features.roof',
        'ConstructionMaterials' => 'features.construction',
        'Heating' => 'features.heating',
        'Cooling' => 'features.cooling',
        ' Flooring' => 'features.flooring',
        'Appliances' => 'features.appliances',
        'InteriorFeatures' => 'features.interior',
        'ExteriorFeatures' => 'features.exterior',
        'CommunityFeatures' => 'amenities.community',
        'ElementarySchool' => 'schools.elementary',
        'MiddleOrJuniorSchool' => 'schools.middle',
        'HighSchool' => 'schools.high',
        'SchoolDistrict' => 'schools.district',
        'OpenHouseStartTime' => 'open_house.start_time',
        'OpenHouseEndTime' => 'open_house.end_time',
        'Media' => 'photos.media',
        'PhotosCount' => 'photos.count',
    ];

    private const FIELD_CATEGORY_RULES = [
        'property.list_price' => 'general',
        'property.bedrooms_total' => 'general',
        'property.bathrooms_total' => 'general',
        'property.type' => 'general',
        'property.subtype' => 'general',
        'property.living_area' => 'general',
        'property.lot_size_area' => 'general',
        'property.year_built' => 'general',
        'listing.status' => 'general',
        'listing.key' => 'general',
        'listing.public_remarks' => 'general',
        'location.city' => 'locations',
        'location.state' => 'locations',
        'location.postal_code' => 'locations',
        'location.street_number' => 'locations',
        'location.street_name' => 'locations',
        'property.latitude' => 'locations',
        'property.longitude' => 'locations',
        'association.hoa_fee' => 'general',
        'features.garage_spaces' => 'features',
        'features.pool' => 'features',
        'features.fireplaces' => 'features',
        'features.stories' => 'features',
        'features.roof' => 'features',
        'features.construction' => 'features',
        'features.heating' => 'features',
        'features.cooling' => 'features',
        'features.flooring' => 'features',
        'features.appliances' => 'features',
        'features.interior' => 'features',
        'features.exterior' => 'features',
        'amenities.community' => 'amenities',
        'dates.on_market' => 'dates',
        'dates.close_date' => 'dates',
        'dates.status_change' => 'dates',
        'schools.elementary' => 'schools',
        'schools.middle' => 'schools',
        'schools.high' => 'schools',
        'schools.district' => 'schools',
        'open_house.start_time' => 'open_house_photos',
        'open_house.end_time' => 'open_house_photos',
        'photos.media' => 'open_house_photos',
        'photos.count' => 'open_house_photos',
    ];

    private const FIELD_TYPE_RULES = [
        'ListPrice' => 'number',
        'BedroomsTotal' => 'number',
        'BathroomsTotalDecimal' => 'number',
        'LivingArea' => 'number',
        'LotSizeArea' => 'number',
        'YearBuilt' => 'number',
        'Latitude' => 'number',
        'Longitude' => 'number',
        'AssociationFee' => 'number',
        'GarageSpaces' => 'number',
        'FireplacesTotal' => 'number',
        'Stories' => 'number',
        'PhotosCount' => 'number',
        'OnMarketDate' => 'date',
        'CloseDate' => 'date',
        'StatusChangeTimestamp' => 'date',
        'OpenHouseStartTime' => 'date',
        'OpenHouseEndTime' => 'date',
        'PublicRemarks' => 'string',
        'ListingKey' => 'string',
        'StreetNumber' => 'string',
        'StreetName' => 'string',
    ];

    public function __construct(
        private readonly LookupProviderInterface $lookups,
    ) {}

    /**
     * Sync field catalog entries from the Bridge lookup API into the database.
     *
     * @return int Number of entries upserted
     */
    public function syncFieldCatalog(string $mlsCode, string $datasetCode): int
    {
        $rawFields = $this->lookups->fetchFieldCatalog($mlsCode, $datasetCode);
        $version = date('Y-m-d');
        $count = 0;

        $seenKeys = [];
        foreach ($rawFields as $entry) {
            $sourceKey = (string) ($entry['LookupName'] ?? '');
            if ($sourceKey === '' || isset($seenKeys[$sourceKey])) {
                continue;
            }
            $seenKeys[$sourceKey] = true;

            $canonical = self::RESO_CANONICAL_MAP[$sourceKey] ?? null;
            $category = $canonical !== null
                ? (self::FIELD_CATEGORY_RULES[$canonical] ?? 'additional_fields')
                : 'additional_fields';
            $fieldType = self::FIELD_TYPE_RULES[$sourceKey] ?? $this->inferFieldType($entry);
            $isReso = $canonical !== null;
            $label = (string) ($entry['LookupValue'] ?? $sourceKey);

            $enumValues = null;
            if (isset($entry['LookupValue']) && is_array($entry['LookupValue'])) {
                $enumValues = array_map(fn ($v) => is_array($v) ? ($v['LookupValue'] ?? $v['Value'] ?? null) : $v, $entry['LookupValue']);
                $enumValues = array_filter($enumValues, fn ($v) => $v !== null);
            }

            MlsFieldCatalog::query()->updateOrCreate(
                [
                    'mls_code' => $mlsCode,
                    'dataset_code' => $datasetCode,
                    'source_field_key' => $sourceKey,
                ],
                [
                    'canonical_field_key' => $canonical,
                    'display_label' => $label,
                    'field_type' => $fieldType,
                    'category' => $category,
                    'operators_json' => $this->defaultOperatorsForType($fieldType),
                    'enum_values_json' => $enumValues,
                    'is_reso_standard' => $isReso,
                    'is_custom_mls_field' => ! $isReso,
                    'lookup_version' => $version,
                ],
            );
            $count++;
        }

        $this->syncMappingAdapters($mlsCode, $datasetCode);

        return $count;
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function getFieldsForScope(array $mlsScope, ?string $category = null): array
    {
        $query = MlsFieldCatalog::query();

        $query->where(function ($q) use ($mlsScope): void {
            foreach ($mlsScope as $scope) {
                $mls = (string) ($scope['mls_code'] ?? '');
                $dataset = (string) ($scope['dataset_code'] ?? '');
                if ($mls !== '' && $dataset !== '') {
                    $q->orWhere(fn ($inner) => $inner->where('mls_code', $mls)->where('dataset_code', $dataset));
                }
            }
        });

        if ($category !== null && $category !== '') {
            $query->where('category', $category);
        }

        return $query
            ->orderByRaw('CASE WHEN canonical_field_key IS NOT NULL THEN 0 ELSE 1 END', [])
            ->orderBy('category')
            ->orderBy('display_label')
            ->get()
            ->toArray();
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function getEnumValues(string $mlsCode, string $datasetCode, string $sourceFieldKey): array
    {
        $field = MlsFieldCatalog::query()
            ->where('mls_code', $mlsCode)
            ->where('dataset_code', $datasetCode)
            ->where('source_field_key', $sourceFieldKey)
            ->first();

        return is_array($field?->enum_values_json) ? $field->enum_values_json : [];
    }

    /**
     * @return list<string>
     */
    public function getCategories(): array
    {
        return [
            'general',
            'locations',
            'school_boundaries',
            'excluded_boundaries',
            'schools',
            'features',
            'amenities',
            'dates',
            'open_house_photos',
            'additional_fields',
        ];
    }

    private function syncMappingAdapters(string $mlsCode, string $datasetCode): void
    {
        foreach (self::RESO_CANONICAL_MAP as $source => $canonical) {
            FieldMappingAdapter::query()->updateOrCreate(
                [
                    'mls_code' => $mlsCode,
                    'dataset_code' => $datasetCode,
                    'canonical_field_key' => $canonical,
                ],
                [
                    'source_field_key' => $source,
                ],
            );
        }
    }

    private function inferFieldType(array $entry): string
    {
        $name = (string) ($entry['LookupName'] ?? '');
        if (preg_match('/(Price|Area|Total|Count|Number|Square|Fee|Space|Year)/i', $name)) {
            return 'number';
        }
        if (preg_match('/(Date|Timestamp|Time)/i', $name)) {
            return 'date';
        }

        return 'enum';
    }

    /**
     * @return list<string>
     */
    private function defaultOperatorsForType(string $fieldType): array
    {
        return match ($fieldType) {
            'number' => ['eq', 'gte', 'lte', 'between'],
            'date' => ['eq', 'gte', 'lte', 'between'],
            'bool' => ['eq'],
            'string' => ['eq', 'contains', 'starts_with'],
            'geo' => ['geo_within', 'geo_not_within'],
            default => ['eq', 'contains'],
        };
    }
}
