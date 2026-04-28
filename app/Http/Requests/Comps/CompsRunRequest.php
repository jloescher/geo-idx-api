<?php

namespace App\Http\Requests\Comps;

use Illuminate\Foundation\Http\FormRequest;

class CompsRunRequest extends FormRequest
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
            'subject' => ['required', 'array'],
            'subject.type' => ['required', 'string', 'in:mls,off_market'],
            'subject.listing_id' => ['required_if:subject.type,mls', 'nullable', 'string', 'max:255'],
            'subject.lat' => ['required_if:subject.type,off_market', 'nullable', 'numeric', 'between:-90,90'],
            'subject.lng' => ['required_if:subject.type,off_market', 'nullable', 'numeric', 'between:-180,180'],
            'subject.bedrooms' => ['nullable', 'integer', 'min:0', 'max:20'],
            'subject.bathrooms' => ['nullable', 'numeric', 'min:0', 'max:99'],
            'subject.living_area_sqft' => ['nullable', 'integer', 'min:0'],
            'subject.lot_size_sqft' => ['nullable', 'integer', 'min:0'],
            'subject.year_built' => ['nullable', 'integer', 'min:1800', 'max:2100'],
            'subject.pool' => ['nullable', 'boolean'],
            'subject.hoa' => ['nullable', 'boolean'],
            'subject.senior_community' => ['nullable', 'boolean'],
            'subject.waterfront' => ['nullable', 'boolean'],
            'subject.garage_spaces' => ['nullable', 'integer', 'min:0', 'max:20'],
            'subject.carport_spaces' => ['nullable', 'integer', 'min:0', 'max:20'],
            'subject.covered_spaces' => ['nullable', 'integer', 'min:0', 'max:20'],
            'subject.open_parking_spaces' => ['nullable', 'integer', 'min:0', 'max:20'],
            'subject.parking_stalls_total' => ['nullable', 'integer', 'min:0', 'max:20'],
            'subject.subdivision_name' => ['nullable', 'string', 'max:200'],
            'subject.mls_area_major' => ['nullable', 'string', 'max:400'],
            'subject.view_yn' => ['nullable', 'boolean'],
            'subject.monthly_fees' => ['nullable', 'numeric', 'min:0'],
            'subject.asking_price' => ['nullable', 'numeric', 'min:0'],
            'subject.flood_zone_code' => ['nullable', 'string', 'max:200'],

            'mode' => ['required', 'string', 'in:A,B,C,D,E,rent_hold_cashflow,flip_vs_hold,appraiser_simulation'],

            'scope' => ['required', 'array'],
            'scope.type' => ['required', 'string', 'in:radius,zip'],
            'scope.radius_miles' => ['required_if:scope.type,radius', 'nullable', 'numeric', 'min:0.1', 'max:50'],
            'scope.center_lat' => ['nullable', 'numeric', 'between:-90,90'],
            'scope.center_lng' => ['nullable', 'numeric', 'between:-180,180'],
            'scope.postal_codes' => ['required_if:scope.type,zip', 'nullable', 'array', 'min:1'],
            'scope.postal_codes.*' => ['string', 'max:20'],

            'filters' => ['nullable', 'array'],
            'filters.living_area_pct' => ['nullable', 'integer', 'min:1', 'max:100'],
            'filters.beds_tolerance' => ['nullable', 'integer', 'min:0', 'max:5'],
            'filters.baths_tolerance' => ['nullable', 'integer', 'min:0', 'max:5'],
            'filters.match_pool' => ['nullable', 'boolean'],
            'filters.match_hoa' => ['nullable', 'boolean'],
            'filters.match_senior_community' => ['nullable', 'boolean'],
            'filters.match_property_sub_type' => ['nullable', 'boolean'],
            'filters.match_waterfront' => ['nullable', 'boolean'],
            'filters.match_view' => ['nullable', 'boolean'],
            'filters.match_subdivision' => ['nullable', 'boolean'],
            'filters.match_mls_area_major' => ['nullable', 'boolean'],
            'filters.match_garage_spaces' => ['nullable', 'boolean'],
            'filters.min_garage_spaces' => ['nullable', 'integer', 'min:0', 'max:20'],
            'filters.max_garage_spaces' => ['nullable', 'integer', 'min:0', 'max:20'],
            'filters.year_built_tolerance' => ['nullable', 'integer', 'min:0', 'max:100'],
            'filters.lot_size_pct' => ['nullable', 'integer', 'min:1', 'max:200'],
            'filters.sold_months_back' => ['nullable', 'integer', 'min:1', 'max:36'],
            'filters.max_sold_comps' => ['nullable', 'integer', 'min:1', 'max:25'],
            'filters.max_competition_comps' => ['nullable', 'integer', 'min:0', 'max:50'],
            'filters.include_active_pending' => ['nullable', 'boolean'],
            'filters.include_overpriced_signals' => ['nullable', 'boolean'],
            'filters.include_failed_listings' => ['nullable', 'boolean'],
            'filters.adj_pool_value' => ['nullable', 'numeric', 'min:0'],
            'filters.adj_garage_per_space' => ['nullable', 'numeric', 'min:0'],
            'filters.adj_waterfront_value' => ['nullable', 'numeric', 'min:0'],
            'filters.adj_year_built_per_year' => ['nullable', 'numeric', 'min:0'],
            'filters.adj_lot_per_acre' => ['nullable', 'numeric', 'min:0'],

            'keywords' => ['nullable', 'array'],
            'keywords.*' => ['array', 'max:50'],
            'keywords.*.*.phrase' => ['required', 'string', 'max:500'],
            'keywords.*.*.weight' => ['required', 'numeric', 'min:0'],
            'keywords.*.*.is_regex' => ['nullable', 'boolean'],

            'aggregation_method' => ['nullable', 'string', 'in:median,average'],

            'rental_params' => ['nullable', 'array'],
            'rental_params.purchase_price' => ['required_if:mode,rent_hold_cashflow,flip_vs_hold', 'nullable', 'numeric', 'min:0'],
            'rental_params.down_payment_percent' => ['nullable', 'numeric', 'min:0', 'max:100'],
            'rental_params.down_payment_amount' => ['nullable', 'numeric', 'min:0'],
            'rental_params.interest_rate' => ['nullable', 'numeric', 'min:0', 'max:100'],
            'rental_params.loan_term_years' => ['nullable', 'integer', 'min:1', 'max:50'],
            'rental_params.annual_tax' => ['nullable', 'numeric', 'min:0'],
            'rental_params.annual_insurance' => ['nullable', 'numeric', 'min:0'],
            'rental_params.monthly_hoa' => ['nullable', 'numeric', 'min:0'],
            'rental_params.monthly_maintenance' => ['nullable', 'numeric', 'min:0'],
            'rental_params.monthly_management' => ['nullable', 'numeric', 'min:0'],
            'rental_params.vacancy_rate' => ['nullable', 'numeric', 'min:0', 'max:100'],

            'flip_params' => ['nullable', 'array'],
            'flip_params.purchase_price' => ['required_if:mode,flip_vs_hold', 'nullable', 'numeric', 'min:0'],
            'flip_params.rehab_budget' => ['nullable', 'numeric', 'min:0'],
            'flip_params.closing_costs_buy' => ['nullable', 'numeric', 'min:0'],
            'flip_params.closing_costs_sell_pct' => ['nullable', 'numeric', 'min:0', 'max:100'],
            'flip_params.holding_months' => ['nullable', 'numeric', 'min:0', 'max:36'],
            'flip_params.holding_cost_monthly' => ['nullable', 'numeric', 'min:0'],
            'flip_params.arv_override' => ['nullable', 'numeric', 'min:0'],

            'simulation_params' => ['nullable', 'array'],
            'simulation_params.supporting_comp_count' => ['nullable', 'integer', 'min:3', 'max:25'],
            'simulation_params.high_adjustment_threshold_pct' => ['nullable', 'numeric', 'min:5', 'max:60'],
        ];
    }
}
