<?php

namespace App\Services\Bridge;

use App\Models\Listing;
use Carbon\CarbonImmutable;
use Illuminate\Database\Eloquent\Builder;

/**
 * Revenue impact: PostGIS-backed search serves map/radius-heavy Active+Pending workloads without
 * Bridge OData latency; mismatched filters bleed MLS budget on fallback — parity with
 * {@see BridgeSearchTranslator} clauses is intentional.
 *
 * Compliance: Stellar MLS IDX / MLS Grid Rule 11 — local mirror complements (not replaces) authoritative
 * MLS payloads; teaser + upstream Bridge gate anything outside this bounded replica window.
 */
final class PostgisSearchService
{
    private const SORT_COLUMN_MAP = [
        'list_price' => 'list_price',
        'on_market_date' => 'on_market_date',
        'year_built' => 'year_built',
        'living_area' => 'living_area',
        'lot_size_acres' => 'lot_size_acres',
        'bedrooms_total' => 'bedrooms_total',
        'bathrooms_total' => 'bathrooms_total_decimal',
    ];

    private const MILES_TO_METERS = 1609.344;

    /**
     * Executes local search mirroring validated SearchRequest + translator-derived pagination/sort hints.
     *
     * @param  array<string, mixed>  $validated
     * @param  array{filter: string, orderby: string, top: int, skip: int, select: string, unselect: string, needsFloodZonePostFilter: bool, lowRiskFloodzone: bool}  $translated
     * @return array{value: list<array<string, mixed>>, count: int, nextLink: ?string}
     */
    public function search(array $validated, string $dataset, array $translated): array
    {
        $datasetLower = strtolower($dataset);
        $query = Listing::query()->where('dataset_slug', $datasetLower);

        $months = max(1, min(48, (int) config('bridge.local_mirror_rolling_months', 12)));
        $rollingCutoff = CarbonImmutable::now('UTC')->subMonths($months);
        $query->whereRaw('LOWER(TRIM(COALESCE(standard_status, \'\'))) IN (\'active\', \'pending\')');
        $query->where(function (Builder $qRolling) use ($rollingCutoff): void {
            $qRolling->where('modification_timestamp', '>=', $rollingCutoff)
                ->orWhere(function (Builder $qRolling2) use ($rollingCutoff): void {
                    $qRolling2->whereNull('modification_timestamp')
                        ->whereNotNull('on_market_date')
                        ->whereDate('on_market_date', '>=', $rollingCutoff->toDateString());
                });
        });

        $hasStatuses = isset($validated['statuses']) && is_array($validated['statuses']) && $validated['statuses'] !== [];
        if (! $hasStatuses && ($validated['active_only'] ?? true)) {
            $query->whereRaw('LOWER(TRIM(COALESCE(standard_status, \'\'))) = ?', ['active']);
        }

        if (isset($validated['min_price'])) {
            $query->where('list_price', '>=', (float) $validated['min_price']);
        }
        if (isset($validated['max_price'])) {
            $query->where('list_price', '<=', (float) $validated['max_price']);
        }

        if (isset($validated['min_beds'])) {
            $query->where('bedrooms_total', '>=', (int) $validated['min_beds']);
        }
        if (isset($validated['max_beds'])) {
            $query->where('bedrooms_total', '<=', (int) $validated['max_beds']);
        }

        if (isset($validated['min_baths'])) {
            $query->where('bathrooms_total_decimal', '>=', (float) $validated['min_baths']);
        }
        if (isset($validated['max_baths'])) {
            $query->where('bathrooms_total_decimal', '<=', (float) $validated['max_baths']);
        }

        if (isset($validated['min_sqft'])) {
            $query->where('living_area', '>=', (int) $validated['min_sqft']);
        }
        if (isset($validated['max_sqft'])) {
            $query->where('living_area', '<=', (int) $validated['max_sqft']);
        }

        if (isset($validated['min_lot_size_acres'])) {
            $query->where('lot_size_acres', '>=', (float) $validated['min_lot_size_acres']);
        }
        if (isset($validated['max_lot_size_acres'])) {
            $query->where('lot_size_acres', '<=', (float) $validated['max_lot_size_acres']);
        }

        if (isset($validated['min_year_built'])) {
            $query->where('year_built', '>=', (int) $validated['min_year_built']);
        }
        if (isset($validated['max_year_built'])) {
            $query->where('year_built', '<=', (int) $validated['max_year_built']);
        }

        if (isset($validated['min_stories'])) {
            $query->where('stories_total', '>=', (int) $validated['min_stories']);
        }
        if (isset($validated['max_stories'])) {
            $query->where('stories_total', '<=', (int) $validated['max_stories']);
        }

        if (isset($validated['min_monthly_fees'])) {
            $query->whereRaw('COALESCE(estimated_total_monthly_fees, 0) >= ?', [(float) $validated['min_monthly_fees']]);
        }
        if (isset($validated['max_monthly_fees'])) {
            $query->whereRaw('COALESCE(estimated_total_monthly_fees, 0) <= ?', [(float) $validated['max_monthly_fees']]);
        }

        $boolColumns = [
            'waterfront' => 'waterfront_yn',
            'pool_private' => 'pool_private_yn',
            'dock' => 'dock_yn',
            'new_construction' => 'new_construction_yn',
            'garage' => 'garage_yn',
            'association' => 'association_yn',
            'spa' => 'spa_yn',
            'fireplace' => 'fireplace_yn',
            'active_adult_community' => 'senior_community_yn',
        ];
        foreach ($boolColumns as $paramKey => $col) {
            if (! empty($validated[$paramKey])) {
                $query->where($col, true);
            }
        }

        if (! empty($validated['for_lease'])) {
            $query->whereRaw('LOWER(COALESCE(property_type, \'\')) = ?', ['residential lease']);
        }

        if (! empty($validated['low_risk_floodzone'])) {
            $query->whereRaw('LOWER(COALESCE(flood_zone_code, \'\')) LIKE ?', ['%x%']);
        }

        if (! empty($validated['city'])) {
            $query->whereRaw('LOWER(TRIM(COALESCE(city, \'\'))) = ?', [strtolower((string) $validated['city'])]);
        }
        if (! empty($validated['state'])) {
            $query->whereRaw('COALESCE(state_or_province, \'\') = ?', [$validated['state']]);
        }
        if (! empty($validated['county'])) {
            $query->whereRaw('LOWER(TRIM(COALESCE(county_or_parish, \'\'))) = ?', [strtolower((string) $validated['county'])]);
        }
        if (! empty($validated['postal_code'])) {
            $query->whereRaw('COALESCE(postal_code, \'\') = ?', [$validated['postal_code']]);
        }

        if (! empty($validated['property_types'])) {
            $query->where(function (Builder $q) use ($validated): void {
                foreach ($validated['property_types'] as $t) {
                    $q->orWhereRaw('COALESCE(property_type, \'\') = ?', [(string) $t]);
                }
            });
        }

        if (! empty($validated['property_sub_types'])) {
            $query->where(function (Builder $q) use ($validated): void {
                foreach ($validated['property_sub_types'] as $t) {
                    $q->orWhereRaw('COALESCE(property_sub_type, \'\') = ?', [(string) $t]);
                }
            });
        }

        if (! empty($validated['statuses'])) {
            $query->where(function (Builder $q) use ($validated): void {
                foreach ($validated['statuses'] as $s) {
                    $q->orWhereRaw('LOWER(COALESCE(standard_status, \'\')) = ?', [strtolower((string) $s)]);
                }
            });
        }

        if (! empty($validated['special_listing_conditions'])) {
            foreach ($validated['special_listing_conditions'] as $cond) {
                $c = (string) $cond;
                $query->whereRaw(
                    'EXISTS (SELECT 1 FROM jsonb_array_elements_text(COALESCE(special_listing_conditions::jsonb, \'[]\'::jsonb)) AS t(v) WHERE t.v = ?)',
                    [$c]
                );
            }
        }

        if (! empty($validated['focus_areas'])) {
            $focusFieldMap = [
                'city' => 'city',
                'county' => 'county_or_parish',
                'state' => 'state_or_province',
                'postal_code' => 'postal_code',
                'elementary_school' => 'elementary_school',
                'middle_school' => 'middle_or_junior_school',
                'high_school' => 'high_school',
                'subdivision' => 'subdivision_name',
            ];
            $query->where(function (Builder $q) use ($validated, $focusFieldMap): void {
                foreach ($validated['focus_areas'] as $area) {
                    $type = $area['type'] ?? '';
                    $name = $area['name'] ?? '';
                    $col = $focusFieldMap[$type] ?? null;
                    if ($col === null || $name === '') {
                        continue;
                    }
                    if (in_array($type, ['city', 'county', 'elementary_school', 'middle_school', 'high_school', 'subdivision'], true)) {
                        $q->orWhereRaw("LOWER(TRIM(COALESCE({$col}, ''))) = ?", [strtolower((string) $name)]);
                    } else {
                        $q->orWhereRaw('COALESCE('.$col.", '') = ?", [(string) $name]);
                    }
                }
            });
        }

        if (isset($validated['geo']['distance']['lat'], $validated['geo']['distance']['lng'], $validated['geo']['distance']['radius_miles'])) {
            $lat = (float) $validated['geo']['distance']['lat'];
            $lng = (float) $validated['geo']['distance']['lng'];
            $miles = (float) $validated['geo']['distance']['radius_miles'];
            $meters = $miles * self::MILES_TO_METERS;
            $query->whereRaw(
                'coordinates IS NOT NULL AND ST_DWithin(coordinates::geography, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)',
                [$lng, $lat, $meters]
            );
        }

        if (isset($validated['geo']['bbox']['west'], $validated['geo']['bbox']['south'], $validated['geo']['bbox']['east'], $validated['geo']['bbox']['north'])) {
            $w = (float) $validated['geo']['bbox']['west'];
            $s = (float) $validated['geo']['bbox']['south'];
            $e = (float) $validated['geo']['bbox']['east'];
            $n = (float) $validated['geo']['bbox']['north'];
            $query->whereRaw(
                'coordinates IS NOT NULL AND ST_Intersects(coordinates, ST_MakeEnvelope(?, ?, ?, ?, 4326)::geography)',
                [$w, $s, $e, $n]
            );
        }

        if (isset($validated['min_days_on_market'])) {
            $cutoff = CarbonImmutable::now('America/New_York')->startOfDay()->subDays((int) $validated['min_days_on_market'])->format('Y-m-d');
            $query->whereDate('on_market_date', '<=', $cutoff);
        }
        if (isset($validated['max_days_on_market'])) {
            $cutoff = CarbonImmutable::now('America/New_York')->startOfDay()->subDays((int) $validated['max_days_on_market'])->format('Y-m-d');
            $query->whereDate('on_market_date', '>=', $cutoff);
        }

        $top = max(1, min(200, (int) $translated['top']));
        $skip = max(0, (int) $translated['skip']);

        $this->applySorting($query, $validated, $translated['orderby'] ?? '');

        $rows = $query->skip($skip)->take($top)->get(['raw_data']);

        $value = [];
        foreach ($rows as $listing) {
            $raw = $listing->raw_data;
            $value[] = is_array($raw) ? $raw : [];
        }

        return [
            'value' => $value,
            'count' => count($value),
            'nextLink' => null,
        ];
    }

    /**
     * @param  array<string, mixed>  $validated
     */
    private function applySorting(Builder $query, array $validated, string $odataOrderby): void
    {
        if (! empty($validated['sort'])) {
            $sortKey = $validated['sort'];
            if ($sortKey === 'distance' && isset($validated['geo']['distance']['lat'], $validated['geo']['distance']['lng'])) {
                $lat = (float) $validated['geo']['distance']['lat'];
                $lng = (float) $validated['geo']['distance']['lng'];
                $query->whereNotNull('coordinates');
                $query->orderByRaw(
                    'coordinates::geography <-> ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography ASC',
                    [$lng, $lat]
                );

                return;
            }
            $col = self::SORT_COLUMN_MAP[$sortKey] ?? 'modification_timestamp';
            $dir = strtolower((string) ($validated['sort_dir'] ?? 'desc')) === 'asc' ? 'asc' : 'desc';
            $query->orderBy($col, $dir);

            return;
        }

        if ($odataOrderby !== '' && preg_match('/^ModificationTimestamp|^BridgeModificationTimestamp/i', trim($odataOrderby))) {
            $query->orderByDesc('modification_timestamp');

            return;
        }

        $query->orderByDesc('modification_timestamp');
    }
}
