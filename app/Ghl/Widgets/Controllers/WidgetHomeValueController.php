<?php

namespace App\Ghl\Widgets\Controllers;

use App\Http\Requests\Widget\WidgetHomeValueRequest;
use App\Services\Bridge\BridgeCompsService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Revenue impact: public "What is my home worth" widget creates lead-gen pipeline
 * for subscribers and drives upsell to full BPO / Mega tier.
 */
class WidgetHomeValueController
{
    public function __construct(
        private readonly BridgeCompsService $compsService,
    ) {}

    public function __invoke(WidgetHomeValueRequest $request): JsonResponse
    {
        $validated = $request->validated();

        $subject = [
            'type' => 'off_market',
            'lat' => 0,
            'lng' => 0,
            'address' => $validated['address'] ?? null,
            'listing_id' => $validated['listing_id'] ?? null,
            'bedrooms' => $validated['bedrooms'] ?? null,
            'full_bathrooms' => $validated['full_bathrooms'] ?? null,
            'half_bathrooms' => $validated['half_bathrooms'] ?? 0,
            'living_area_sqft' => $validated['living_area_sqft'] ?? null,
            'year_built' => $validated['year_built'] ?? null,
            'condition' => $validated['condition'] ?? null,
            'property_type' => $validated['property_type'] ?? null,
            'garage_spaces' => $validated['garage_spaces'] ?? null,
            'pool' => $validated['pool'] ?? null,
            'waterfront' => $validated['waterfront'] ?? null,
            'lot_size_sqft' => $validated['lot_size_sqft'] ?? null,
            'hoa_monthly_fee' => $validated['hoa_monthly_fee'] ?? null,
            'stories' => $validated['stories'] ?? null,
            'renovated_kitchen_year' => $validated['renovated_kitchen_year'] ?? null,
            'renovated_bathrooms_year' => $validated['renovated_bathrooms_year'] ?? null,
            'renovated_hvac_year' => $validated['renovated_hvac_year'] ?? null,
            'enclosed_lanai_sqft' => $validated['enclosed_lanai_sqft'] ?? null,
            'screen_pool_enclosure' => $validated['screen_pool_enclosure'] ?? null,
        ];

        $compsPayload = [
            'subject' => $subject,
            'mode' => 'home_value',
            'scope' => [
                'type' => 'radius',
                'radius_miles' => 3.0,
            ],
            'filters' => [
                'sold_months_back' => 12,
                'max_sold_comps' => 10,
                'living_area_pct' => 20,
                'beds_tolerance' => 1,
                'baths_tolerance' => 1,
                'year_built_tolerance' => 15,
            ],
            'home_value_params' => [
                'sold_months_back' => 12,
                'max_comps' => 8,
            ],
        ];

        $internalRequest = Request::create('/api/v1/comps/run', 'POST', $compsPayload);
        $internalRequest->headers->set('Accept', 'application/json');

        // Forward full access flag from widget middleware
        $internalRequest->attributes->set('bridge.full_access', true);

        $result = $this->compsService->run($internalRequest, $compsPayload);

        if (($result['success'] ?? false) === false) {
            return response()->json([
                'success' => false,
                'error' => $result['error'] ?? 'estimate_failed',
            ], 422);
        }

        $homeValue = $result['home_value_result'] ?? [];

        // Simplified response for widget display
        return response()->json([
            'success' => true,
            'estimated_value' => $homeValue['point_estimate'] ?? null,
            'value_range' => $homeValue['range'] ?? ['low' => null, 'high' => null],
            'confidence' => $homeValue['confidence_band'] ?? 'low',
            'comparable_count' => count($homeValue['adjustment_grid'] ?? []),
            'comps_summary' => $this->summarizeComps($homeValue['adjustment_grid'] ?? []),
            'market_rates_summary' => [
                'median_price_per_sf' => ($homeValue['market_rates']['gla_per_sf'] ?? 0) > 0
                    ? round($homeValue['market_rates']['gla_per_sf'], 2)
                    : null,
                'method' => $homeValue['market_rates']['method'] ?? 'unknown',
            ],
        ]);
    }

    /**
     * @param  list<array<string, mixed>>  $grid
     * @return list<array<string, mixed>>
     */
    private function summarizeComps(array $grid): array
    {
        $top = array_slice($grid, 0, 3);

        return array_map(static function (array $entry): array {
            return [
                'address' => $entry['address'] ?? '',
                'sale_price' => $entry['sale_price'] ?? null,
                'adjusted_price' => $entry['adjusted_price'] ?? null,
                'distance_miles' => $entry['distance_miles'] ?? null,
            ];
        }, $top);
    }
}
