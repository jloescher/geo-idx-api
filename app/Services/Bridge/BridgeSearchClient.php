<?php

namespace App\Services\Bridge;

use App\Http\Responses\Search\ListingResult;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

final readonly class BridgeSearchClient
{
    public function __construct(
        private BridgeHttpService $bridge,
    ) {}

    /**
     * Execute a search query against the Bridge RESO OData Property endpoint.
     *
     * @return array{value: list<array<string,mixed>>, count: int, nextLink: ?string}
     */
    public function search(
        string $dataset,
        string $filter,
        string $orderby,
        int $top,
        int $skip,
        string $select,
        string $unselect,
    ): array {
        $queryParams = [
            '$top' => $top,
            '$skip' => $skip,
            '$select' => $select,
            '$unselect' => $unselect,
        ];

        if ($filter !== '') {
            $queryParams['$filter'] = $filter;
        }
        if ($orderby !== '') {
            $queryParams['$orderby'] = $orderby;
        }

        $url = $this->buildResoUrl($dataset);
        $response = $this->bridge->serverJsonGet($url, $queryParams);

        if (! $response->successful()) {
            return [
                'value' => [],
                'count' => 0,
                'nextLink' => null,
            ];
        }

        $body = $response->json();
        $value = is_array($body['value'] ?? null) ? $body['value'] : [];
        $nextLink = is_string($body['@odata.nextLink'] ?? null) ? $body['@odata.nextLink'] : null;

        return [
            'value' => $value,
            'count' => count($value),
            'nextLink' => $nextLink,
        ];
    }

    /**
     * Fetch a single Property entity for comparables (subject resolution).
     * Tries OData entity URL first, then legacy {@see BridgeHttpService::resoEntityUrls} paths.
     *
     * @return array<string, mixed>|null
     */
    public function getPropertyForComps(Request $incoming, string $dataset, string $listingKey): ?array
    {
        $escaped = str_replace("'", "''", $listingKey);
        $select = $this->compsPropertySelectList($dataset);
        $urls = array_values(array_unique(array_filter([
            $this->buildResoUrl($dataset)."('{$escaped}')",
            ...$this->bridge->resoEntityUrls('Property', $listingKey, $dataset),
        ])));

        foreach ($urls as $url) {
            $response = $this->bridge->getJsonFromUrl($url, $incoming, ['$select' => $select]);
            if (! $response->successful()) {
                continue;
            }
            $data = $response->json();
            if (is_array($data) && isset($data['ListingKey']) && is_string($data['ListingKey'])) {
                return $data;
            }
        }

        return null;
    }

    public function compsPropertySelectList(string $dataset): string
    {
        $u = strtoupper($dataset);

        return implode(',', [
            'ListingKey', 'StandardStatus', 'ListPrice', 'ClosePrice', 'OriginalListPrice',
            'PreviousListPrice', 'CloseDate', 'OnMarketDate', 'BedroomsTotal', 'BathroomsTotalDecimal',
            'LivingArea', 'LotSizeAcres', 'YearBuilt', 'StoriesTotal', 'City', 'StateOrProvince',
            'PostalCode', 'CountyOrParish', 'PropertyType', 'PropertySubType', 'WaterfrontYN',
            'PoolPrivateYN', 'GarageYN', 'GarageSpaces', 'CarportSpaces', 'CoveredSpaces',
            'OpenParkingSpaces', 'AssociationYN', 'SeniorCommunityYN',
            'Coordinates', 'Latitude', 'Longitude', 'StreetNumber', 'StreetName', 'StreetDirPrefix', 'StreetSuffix',
            'StreetDirSuffix', 'UnitNumber', 'DaysOnMarket', 'CumulativeDaysOnMarket', 'PublicRemarks',
            'SubdivisionName', 'MLSAreaMajor', 'ViewYN', 'LeaseAmountFrequency', 'LeaseTerm',
            'Furnished', 'PetsAllowed', 'OwnerPays', 'TenantPays',
            "{$u}_FloodZoneCode", "{$u}_TotalMonthlyFees",
        ]);
    }

    /**
     * Build the RESO OData Property URL for a given dataset.
     */
    private function buildResoUrl(string $dataset): string
    {
        $host = rtrim((string) config('bridge.host'), '/');
        $prefix = trim((string) config('bridge.path_prefix'), '/');
        $resoRoot = trim((string) config('bridge.reso_root'), '/');

        $basePath = $prefix !== ''
            ? "{$prefix}/OData/{$dataset}/Property"
            : ($resoRoot !== ''
                ? "{$resoRoot}/OData/{$dataset}/Property"
                : "OData/{$dataset}/Property"
            );

        return "{$host}/{$basePath}";
    }

    /**
     * Map a Bridge Property OData record to a ListingResult.
     */
    public function mapToListingResult(array $record, string $dataset): ListingResult
    {
        $datasetUpper = strtoupper($dataset);

        $listingKey = (string) ($record['ListingKey'] ?? '');
        $mediaSources = $this->extractPrimaryImageSources($record, $listingKey);

        // Compute full address
        $addressParts = array_filter([
            (string) ($record['StreetNumber'] ?? ''),
            (string) ($record['StreetName'] ?? ''),
        ]);
        $fullAddress = implode(' ', $addressParts);
        $cityState = array_filter([
            (string) ($record['City'] ?? ''),
            (string) ($record['StateOrProvince'] ?? ''),
            (string) ($record['PostalCode'] ?? ''),
        ]);
        if ($fullAddress !== '' && $cityState !== []) {
            $fullAddress .= ', '.implode(' ', $cityState);
        } elseif ($cityState !== []) {
            $fullAddress = implode(' ', $cityState);
        }

        // Compute days on market
        $daysOnMarket = null;
        $onMarketDate = $record['OnMarketDate'] ?? null;
        if (is_string($onMarketDate) && $onMarketDate !== '') {
            try {
                $omd = new \DateTimeImmutable($onMarketDate);
                $now = new \DateTimeImmutable('now', new \DateTimeZone('America/New_York'));
                $today = new \DateTimeImmutable($now->format('Y-m-d'), new \DateTimeZone('America/New_York'));
                $omdDay = new \DateTimeImmutable($omd->format('Y-m-d'), new \DateTimeZone('America/New_York'));
                $dom = (int) $today->diff($omdDay)->days;
                $daysOnMarket = max(0, $dom);
            } catch (\Throwable) {
                // Leave as null
            }
        }

        // Extract special listing conditions from array field
        $specialConditions = null;
        if (is_array($record['SpecialListingConditions'] ?? null)) {
            $specialConditions = array_values(array_filter(
                array_map('trim', $record['SpecialListingConditions']),
                static fn (string $v) => $v !== '',
            ));
        }

        // Coordinates
        $latitude = null;
        $longitude = null;
        if (isset($record['Coordinates'])) {
            $coords = $record['Coordinates'];
            if (is_array($coords) && isset($coords['coordinates'])) {
                $coordPair = $coords['coordinates'];
                if (is_array($coordPair) && count($coordPair) >= 2) {
                    $longitude = is_numeric($coordPair[0]) ? (float) $coordPair[0] : null;
                    $latitude = is_numeric($coordPair[1]) ? (float) $coordPair[1] : null;
                }
            }
        }

        // Flood zone code (dataset-specific field)
        $floodZoneField = "{$datasetUpper}_FloodZoneCode";
        $floodZoneCode = is_string($record[$floodZoneField] ?? null) && $record[$floodZoneField] !== ''
            ? $record[$floodZoneField]
            : null;

        // Monthly fees (dataset-specific field)
        $monthlyFeesField = "{$datasetUpper}_TotalMonthlyFees";
        $monthlyFees = is_numeric($record[$monthlyFeesField] ?? null)
            ? (float) $record[$monthlyFeesField]
            : null;

        return new ListingResult(
            listingId: Str::afterLast($listingKey, ':'),
            standardStatus: $this->nullableString($record['StandardStatus'] ?? null),
            listPrice: $this->nullableFloat($record['ListPrice'] ?? null),
            closePrice: $this->nullableFloat($record['ClosePrice'] ?? null),
            originalListPrice: $this->nullableFloat($record['OriginalListPrice'] ?? null),
            previousListPrice: $this->nullableFloat($record['PreviousListPrice'] ?? null),
            bedroomsTotal: $this->nullableInt($record['BedroomsTotal'] ?? null),
            bathroomsTotal: $this->nullableFloat($record['BathroomsTotalDecimal'] ?? null),
            livingArea: $this->nullableInt($record['LivingArea'] ?? null),
            lotSizeAcres: $this->nullableFloat($record['LotSizeAcres'] ?? null),
            yearBuilt: $this->nullableInt($record['YearBuilt'] ?? null),
            streetNumber: $this->nullableString($record['StreetNumber'] ?? null),
            streetName: $this->nullableString($record['StreetName'] ?? null),
            city: $this->nullableString($record['City'] ?? null),
            state: $this->nullableString($record['StateOrProvince'] ?? null),
            postalCode: $this->nullableString($record['PostalCode'] ?? null),
            county: $this->nullableString($record['CountyOrParish'] ?? null),
            fullAddress: $fullAddress !== '' ? $fullAddress : null,
            propertyType: $this->nullableString($record['PropertyType'] ?? null),
            propertySubType: $this->nullableString($record['PropertySubType'] ?? null),
            specialListingConditions: $specialConditions,
            waterfrontYn: $this->nullableBool($record['WaterfrontYN'] ?? null),
            poolPrivateYn: $this->nullableBool($record['PoolPrivateYN'] ?? null),
            dockYn: $this->nullableBool($record['DockYN'] ?? null),
            newConstructionYn: $this->nullableBool($record['NewConstructionYN'] ?? null),
            latitude: $latitude,
            longitude: $longitude,
            onMarketDate: $this->nullableString($record['OnMarketDate'] ?? null),
            modificationTimestamp: $this->nullableString($record['ModificationTimestamp'] ?? null),
            daysOnMarket: $daysOnMarket,
            floodZoneCode: $floodZoneCode,
            primaryImage: $mediaSources,
            listAgentMlsId: $this->nullableString($record['ListAgentMlsId'] ?? null),
            listOfficeMlsId: $this->nullableString($record['ListOfficeMlsId'] ?? null),
            distanceMiles: null,
            storiesTotal: $this->nullableInt($record['StoriesTotal'] ?? null),
            monthlyFees: $monthlyFees,
            garageYn: $this->nullableBool($record['GarageYN'] ?? null),
            associationYn: $this->nullableBool($record['AssociationYN'] ?? null),
            spaYn: $this->nullableBool($record['SpaYN'] ?? null),
            fireplaceYn: $this->nullableBool($record['FireplaceYN'] ?? null),
            seniorCommunityYn: $this->nullableBool($record['SeniorCommunityYN'] ?? null),
            priceChangeTimestamp: $this->nullableString($record['PriceChangeTimestamp'] ?? null),
        );
    }

    private function nullableString(mixed $value): ?string
    {
        return is_string($value) && trim($value) !== '' ? trim($value) : null;
    }

    private function nullableFloat(mixed $value): ?float
    {
        return is_numeric($value) ? (float) $value : null;
    }

    private function nullableInt(mixed $value): ?int
    {
        return is_numeric($value) ? (int) $value : null;
    }

    private function nullableBool(mixed $value): ?bool
    {
        return is_bool($value) ? $value : null;
    }

    /**
     * Extract primary image sources from inline Media array.
     *
     * @return array<string,string>|null
     */
    private function extractPrimaryImageSources(array $record, string $listingKey): ?array
    {
        $media = $record['Media'] ?? null;
        if (! is_array($media) || $media === []) {
            return null;
        }

        // Find primary media (order 0 or first photo)
        $primaryMedia = null;
        foreach ($media as $item) {
            if (! is_array($item)) {
                continue;
            }
            $order = $item['Order'] ?? null;
            $category = $item['MediaCategory'] ?? null;
            if (($order === 0 || $order === '0') && $category === 'Photo') {
                $primaryMedia = $item;
                break;
            }
            if ($primaryMedia === null && $category === 'Photo') {
                $primaryMedia = $item;
            }
        }

        if ($primaryMedia === null) {
            return null;
        }

        $mediaKey = (string) ($primaryMedia['MediaKey'] ?? '');
        if ($mediaKey === '') {
            return null;
        }

        $cdnHost = config('idx_urls.images_public_url', rtrim((string) config('app.url'), '/'));
        // Build idx-images URL pattern
        $listingSlug = Str::afterLast($listingKey, ':');
        $keySlug = basename($mediaKey);
        $keySlug = preg_replace('/\.(jpg|jpeg|png|gif|webp|avif)$/i', '', $keySlug);

        return [
            'avif' => "{$cdnHost}/{$listingSlug}/{$keySlug}.avif",
            'webp' => "{$cdnHost}/{$listingSlug}/{$keySlug}.webp",
        ];
    }
}
