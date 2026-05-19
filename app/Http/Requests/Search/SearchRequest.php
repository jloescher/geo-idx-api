<?php

namespace App\Http\Requests\Search;

use App\Services\Bridge\PostgisSearchService;
use Illuminate\Foundation\Http\FormRequest;

/**
 * PostGIS mirror leg: {@see PostgisSearchService} filters
 * {@code low_risk_floodzone} on {@code listings.flood_zone_code} and
 * {@code min_monthly_fees}/{@code max_monthly_fees} on {@code listings.estimated_total_monthly_fees}.
 */
class SearchRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    /**
     * @return array<string, mixed>
     */
    public function rules(): array
    {
        return [
            'min_price' => ['nullable', 'numeric', 'min:0'],
            'max_price' => ['nullable', 'numeric', 'min:0'],
            'min_beds' => ['nullable', 'integer', 'min:0'],
            'max_beds' => ['nullable', 'integer', 'min:0'],
            'min_baths' => ['nullable', 'numeric', 'min:0'],
            'max_baths' => ['nullable', 'numeric', 'min:0'],
            'min_sqft' => ['nullable', 'numeric', 'min:0'],
            'max_sqft' => ['nullable', 'numeric', 'min:0'],
            'min_lot_size_acres' => ['nullable', 'numeric', 'min:0'],
            'max_lot_size_acres' => ['nullable', 'numeric', 'min:0'],
            'min_year_built' => ['nullable', 'integer', 'min:1800'],
            'max_year_built' => ['nullable', 'integer', 'min:1800'],
            'min_days_on_market' => ['nullable', 'integer', 'min:0'],
            'max_days_on_market' => ['nullable', 'integer', 'min:0'],
            'active_only' => ['nullable', 'boolean'],
            'waterfront' => ['nullable', 'boolean'],
            'pool_private' => ['nullable', 'boolean'],
            'dock' => ['nullable', 'boolean'],
            'new_construction' => ['nullable', 'boolean'],
            'has_photos' => ['nullable', 'boolean'],
            'low_risk_floodzone' => ['nullable', 'boolean'],
            'city' => ['nullable', 'string', 'max:200'],
            'state' => ['nullable', 'string', 'max:30'],
            'county' => ['nullable', 'string', 'max:200'],
            'postal_code' => ['nullable', 'string', 'max:20'],
            'property_types' => ['nullable', 'array'],
            'property_types.*' => ['string', 'max:100'],
            'property_sub_types' => ['nullable', 'array'],
            'property_sub_types.*' => ['string', 'max:100'],
            'statuses' => ['nullable', 'array'],
            'statuses.*' => ['string', 'max:50'],
            'special_listing_conditions' => ['nullable', 'array'],
            'special_listing_conditions.*' => ['string', 'max:100'],
            'sort' => ['nullable', 'string', 'in:list_price,on_market_date,year_built,living_area,lot_size_acres,bedrooms_total,bathrooms_total,distance'],
            'sort_dir' => ['nullable', 'string', 'in:asc,desc'],
            'page.limit' => ['nullable', 'integer', 'min:1', 'max:200'],
            'page.skip' => ['nullable', 'integer', 'min:0'],
            'geo.distance.lat' => ['nullable', 'numeric', 'between:-90,90'],
            'geo.distance.lng' => ['nullable', 'numeric', 'between:-180,180'],
            'geo.distance.radius_miles' => ['nullable', 'numeric', 'min:0.01'],
            'geo.bbox.west' => ['nullable', 'numeric', 'between:-180,180'],
            'geo.bbox.south' => ['nullable', 'numeric', 'between:-90,90'],
            'geo.bbox.east' => ['nullable', 'numeric', 'between:-180,180'],
            'geo.bbox.north' => ['nullable', 'numeric', 'between:-90,90'],
            'focus_areas' => ['nullable', 'array'],
            'focus_areas.*.type' => ['required_with:focus_areas', 'string', 'in:city,county,state,postal_code,elementary_school,middle_school,high_school,subdivision'],
            'focus_areas.*.name' => ['required_with:focus_areas', 'string', 'max:500'],
            'min_monthly_fees' => ['nullable', 'numeric', 'min:0'],
            'max_monthly_fees' => ['nullable', 'numeric', 'min:0'],
            'min_stories' => ['nullable', 'integer', 'min:1'],
            'max_stories' => ['nullable', 'integer', 'min:1'],
            'garage' => ['nullable', 'boolean'],
            'association' => ['nullable', 'boolean'],
            'spa' => ['nullable', 'boolean'],
            'fireplace' => ['nullable', 'boolean'],
            'active_adult_community' => ['nullable', 'boolean'],
            'for_lease' => ['nullable', 'boolean'],
            'price_reduced_within_days' => ['nullable', 'integer', 'min:1'],
        ];
    }
}
