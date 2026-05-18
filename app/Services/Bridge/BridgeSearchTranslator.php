<?php

namespace App\Services\Bridge;

use App\Http\Requests\Search\SearchRequest;

final readonly class BridgeSearchTranslator
{
    private const HIGH_RISK_FLOOD_ZONES = ['a', 'ae', 'ah', 'ao', 'ar', 'a99', 'v', 've'];

    private const SORT_FIELD_MAP = [
        'list_price' => 'ListPrice',
        'on_market_date' => 'OnMarketDate',
        'year_built' => 'YearBuilt',
        'living_area' => 'LivingArea',
        'lot_size_acres' => 'LotSizeAcres',
        'bedrooms_total' => 'BedroomsTotal',
        'bathrooms_total' => 'BathroomsTotalDecimal',
    ];

    /**
     * @param  list<string>|null  $statusOverride  Lowercase RESO statuses (e.g. active, closed) for split/hybrid legs.
     * @param  array{top?: int, skip?: int}|null  $pageOverride
     * @return array{filter: string, orderby: string, top: int, skip: int, select: string, unselect: string, needsFloodZonePostFilter: bool, lowRiskFloodzone: bool}
     */
    public function translate(
        SearchRequest $request,
        string $dataset,
        ?array $statusOverride = null,
        ?array $pageOverride = null,
    ): array {
        $clauses = [];
        $params = $request->validated();

        if ($statusOverride !== null) {
            if ($statusOverride !== []) {
                $statusClauses = array_map(
                    static fn (string $s): string => "tolower(StandardStatus) eq '".strtolower($s)."'",
                    $statusOverride,
                );
                $clauses[] = '('.implode(' or ', $statusClauses).')';
            }
        } else {
            $activeOnly = $params['active_only'] ?? true;
            if ($activeOnly) {
                $clauses[] = "tolower(StandardStatus) eq 'active'";
            }
        }

        if (isset($params['min_price'])) {
            $clauses[] = "ListPrice ge {$params['min_price']}";
        }
        if (isset($params['max_price'])) {
            $clauses[] = "ListPrice le {$params['max_price']}";
        }

        if (isset($params['min_beds'])) {
            $clauses[] = "BedroomsTotal ge {$params['min_beds']}";
        }
        if (isset($params['max_beds'])) {
            $clauses[] = "BedroomsTotal le {$params['max_beds']}";
        }

        if (isset($params['min_baths'])) {
            $clauses[] = "BathroomsTotalDecimal ge {$params['min_baths']}";
        }
        if (isset($params['max_baths'])) {
            $clauses[] = "BathroomsTotalDecimal le {$params['max_baths']}";
        }

        if (isset($params['min_sqft'])) {
            $clauses[] = "LivingArea ge {$params['min_sqft']}";
        }
        if (isset($params['max_sqft'])) {
            $clauses[] = "LivingArea le {$params['max_sqft']}";
        }

        if (isset($params['min_lot_size_acres'])) {
            $clauses[] = "LotSizeAcres ge {$params['min_lot_size_acres']}";
        }
        if (isset($params['max_lot_size_acres'])) {
            $clauses[] = "LotSizeAcres le {$params['max_lot_size_acres']}";
        }

        if (isset($params['min_year_built'])) {
            $clauses[] = "YearBuilt ge {$params['min_year_built']}";
        }
        if (isset($params['max_year_built'])) {
            $clauses[] = "YearBuilt le {$params['max_year_built']}";
        }

        if (isset($params['min_stories'])) {
            $clauses[] = "StoriesTotal ge {$params['min_stories']}";
        }
        if (isset($params['max_stories'])) {
            $clauses[] = "StoriesTotal le {$params['max_stories']}";
        }

        if (isset($params['min_monthly_fees'])) {
            $datasetUpper = strtoupper($dataset);
            $clauses[] = "{$datasetUpper}_TotalMonthlyFees ge {$params['min_monthly_fees']}";
        }
        if (isset($params['max_monthly_fees'])) {
            $datasetUpper = strtoupper($dataset);
            $clauses[] = "{$datasetUpper}_TotalMonthlyFees le {$params['max_monthly_fees']}";
        }

        $boolMap = [
            'waterfront' => 'WaterfrontYN',
            'pool_private' => 'PoolPrivateYN',
            'dock' => 'DockYN',
            'new_construction' => 'NewConstructionYN',
            'garage' => 'GarageYN',
            'association' => 'AssociationYN',
            'spa' => 'SpaYN',
            'fireplace' => 'FireplaceYN',
            'active_adult_community' => 'SeniorCommunityYN',
        ];
        foreach ($boolMap as $paramKey => $field) {
            if (! empty($params[$paramKey])) {
                $clauses[] = "{$field} eq true";
            }
        }

        if (! empty($params['for_lease'])) {
            $clauses[] = "PropertyType eq 'Residential Lease'";
        }

        $lowRiskFloodzone = ! empty($params['low_risk_floodzone']);
        $needsFloodZonePostFilter = false;
        if ($lowRiskFloodzone) {
            $datasetUpper = strtoupper($dataset);
            $clauses[] = "contains(tolower({$datasetUpper}_FloodZoneCode), 'x')";
            $needsFloodZonePostFilter = true;
        }

        if (! empty($params['city'])) {
            $clauses[] = "tolower(City) eq '".strtolower($params['city'])."'";
        }
        if (! empty($params['state'])) {
            $clauses[] = "StateOrProvince eq '{$params['state']}'";
        }
        if (! empty($params['county'])) {
            $clauses[] = "tolower(CountyOrParish) eq '".strtolower($params['county'])."'";
        }
        if (! empty($params['postal_code'])) {
            $clauses[] = "PostalCode eq '{$params['postal_code']}'";
        }

        if (! empty($params['property_types'])) {
            $typeClauses = array_map(
                static fn ($t) => "PropertyType eq '{$t}'",
                $params['property_types'],
            );
            $clauses[] = '('.implode(' or ', $typeClauses).')';
        }

        if (! empty($params['property_sub_types'])) {
            $subTypeClauses = array_map(
                static fn ($t) => "PropertySubType eq '{$t}'",
                $params['property_sub_types'],
            );
            $clauses[] = '('.implode(' or ', $subTypeClauses).')';
        }

        if ($statusOverride === null && ! empty($params['statuses'])) {
            $statusClauses = array_map(
                static fn ($s) => "tolower(StandardStatus) eq '".strtolower($s)."'",
                $params['statuses'],
            );
            $clauses[] = '('.implode(' or ', $statusClauses).')';
        }

        if (! empty($params['special_listing_conditions'])) {
            $condClauses = array_map(
                static fn ($c) => "SpecialListingConditions/any(c: c eq '{$c}')",
                $params['special_listing_conditions'],
            );
            $clauses[] = '('.implode(' or ', $condClauses).')';
        }

        if (! empty($params['focus_areas'])) {
            $focusClauses = [];
            $focusFieldMap = [
                'city' => 'City',
                'county' => 'CountyOrParish',
                'state' => 'StateOrProvince',
                'postal_code' => 'PostalCode',
                'elementary_school' => 'ElementarySchool',
                'middle_school' => 'MiddleOrJuniorSchool',
                'high_school' => 'HighSchool',
                'subdivision' => 'SubdivisionName',
            ];
            foreach ($params['focus_areas'] as $area) {
                $type = $area['type'] ?? '';
                $name = $area['name'] ?? '';
                $field = $focusFieldMap[$type] ?? null;
                if ($field !== null && $name !== '') {
                    if (in_array($type, ['city', 'county', 'elementary_school', 'middle_school', 'high_school', 'subdivision'], true)) {
                        $focusClauses[] = "tolower({$field}) eq '".strtolower($name)."'";
                    } else {
                        $focusClauses[] = "{$field} eq '{$name}'";
                    }
                }
            }
            if ($focusClauses !== []) {
                $clauses[] = '('.implode(' or ', $focusClauses).')';
            }
        }

        if (isset($params['geo']['distance']['lat']) && isset($params['geo']['distance']['lng']) && isset($params['geo']['distance']['radius_miles'])) {
            $lat = $params['geo']['distance']['lat'];
            $lng = $params['geo']['distance']['lng'];
            $radius = $params['geo']['distance']['radius_miles'];
            $clauses[] = "geo.distance(Coordinates, POINT({$lng} {$lat})) lt {$radius}";
        }

        if (isset($params['geo']['bbox']['west'], $params['geo']['bbox']['south'], $params['geo']['bbox']['east'], $params['geo']['bbox']['north'])) {
            $w = $params['geo']['bbox']['west'];
            $s = $params['geo']['bbox']['south'];
            $e = $params['geo']['bbox']['east'];
            $n = $params['geo']['bbox']['north'];
            $clauses[] = "Latitude ge {$s} and Latitude le {$n} and Longitude ge {$w} and Longitude le {$e}";
        }

        if (isset($params['price_reduced_within_days'])) {
            $threshold = now()->subDays($params['price_reduced_within_days'])->format('Y-m-d');
            $clauses[] = "PriceChangeTimestamp ge {$threshold} and ListPrice lt PreviousListPrice";
        }

        if (isset($params['min_days_on_market'])) {
            $cutoff = now()->subDays($params['min_days_on_market'])->format('Y-m-d');
            $clauses[] = "OnMarketDate le {$cutoff}";
        }
        if (isset($params['max_days_on_market'])) {
            $cutoff = now()->subDays($params['max_days_on_market'])->format('Y-m-d');
            $clauses[] = "OnMarketDate ge {$cutoff}";
        }

        $filter = implode(' and ', $clauses);

        $orderby = '';
        if (! empty($params['sort'])) {
            $sortKey = $params['sort'];
            if ($sortKey === 'distance') {
                if (isset($params['geo']['distance']['lat']) && isset($params['geo']['distance']['lng'])) {
                    $lat = $params['geo']['distance']['lat'];
                    $lng = $params['geo']['distance']['lng'];
                    $orderby = "geo.distance(Coordinates, POINT({$lng} {$lat})) asc";
                }
            } else {
                $field = self::SORT_FIELD_MAP[$sortKey] ?? 'OnMarketDate';
                $dir = $params['sort_dir'] ?? 'desc';
                $orderby = "{$field} {$dir}";
            }
        }

        $top = (int) ($pageOverride['top'] ?? $params['page']['limit'] ?? 24);
        $skip = (int) ($pageOverride['skip'] ?? $params['page']['skip'] ?? 0);

        $datasetUpper = strtoupper($dataset);
        $selectFields = [
            'ListingKey', 'StandardStatus', 'ListPrice', 'ClosePrice', 'OriginalListPrice',
            'PreviousListPrice', 'BedroomsTotal', 'BathroomsTotalDecimal', 'LivingArea',
            'LotSizeAcres', 'YearBuilt', 'StoriesTotal', 'City', 'StateOrProvince',
            'PostalCode', 'CountyOrParish', 'PropertyType', 'PropertySubType',
            'WaterfrontYN', 'PoolPrivateYN', 'DockYN', 'NewConstructionYN',
            'GarageYN', 'AssociationYN', 'SpaYN', 'FireplaceYN', 'SeniorCommunityYN',
            'OnMarketDate', 'ModificationTimestamp', 'PriceChangeTimestamp',
            'Coordinates', 'Media', 'StreetNumber', 'StreetName',
            'ListAgentMlsId', 'ListOfficeMlsId', 'SpecialListingConditions',
            "{$datasetUpper}_TotalMonthlyFees", "{$datasetUpper}_FloodZoneCode",
        ];
        $select = implode(',', $selectFields);

        $unselect = implode(',', [
            'PublicRemarks', 'PrivateRemarks', 'InteriorFeatures', 'ExteriorFeatures',
            'Utilities', 'Appliances', 'ConstructionMaterials', 'FoundationDetails',
            'Roof', 'View', 'WindowFeatures', 'Cooling', 'Heating',
        ]);

        return [
            'filter' => $filter,
            'orderby' => $orderby,
            'top' => $top,
            'skip' => $skip,
            'select' => $select,
            'unselect' => $unselect,
            'needsFloodZonePostFilter' => $needsFloodZonePostFilter,
            'lowRiskFloodzone' => $lowRiskFloodzone,
        ];
    }

    /**
     * @param  list<array<string,mixed>>  $results
     * @return list<array<string,mixed>>
     */
    public function filterLowRiskFloodZone(array $results, string $dataset): array
    {
        $field = strtoupper($dataset).'_FloodZoneCode';

        return array_values(array_filter($results, static function (array $row) use ($field): bool {
            $zoneValue = $row[$field] ?? null;
            if (! is_string($zoneValue) || $zoneValue === '') {
                return false;
            }

            $zones = array_map('trim', explode(',', $zoneValue));
            foreach ($zones as $zone) {
                $zoneLower = strtolower($zone);
                if ($zoneLower === '') {
                    continue;
                }
                foreach (self::HIGH_RISK_FLOOD_ZONES as $highRisk) {
                    if ($zoneLower === $highRisk || str_starts_with($zoneLower, $highRisk)) {
                        return false;
                    }
                }
            }

            return true;
        }));
    }
}
