<?php

namespace App\Http\Requests\Widget;

use Illuminate\Foundation\Http\FormRequest;

class WidgetHomeValueRequest extends FormRequest
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
            // Either address or listing_id is required
            'address' => ['required_without:listing_id', 'nullable', 'string', 'max:500'],
            'listing_id' => ['required_without:address', 'nullable', 'string', 'max:255'],
            'api_key' => ['required', 'string'],

            // Required fields (used when address is provided; auto-populated from listing when listing_id given)
            'property_type' => ['required_without:listing_id', 'nullable', 'string', 'in:sfr,townhouse,condo,manufactured,duplex,triplex,quadplex,modular,cabin'],
            'bedrooms' => ['required_without:listing_id', 'nullable', 'integer', 'min:1', 'max:20'],
            'full_bathrooms' => ['required_without:listing_id', 'nullable', 'integer', 'min:0', 'max:20'],
            'half_bathrooms' => ['required', 'integer', 'min:0', 'max:10'],
            'living_area_sqft' => ['required_without:listing_id', 'nullable', 'integer', 'min:100', 'max:50000'],
            'condition' => ['nullable', 'string', 'in:poor,fair,good,excellent'],
            'year_built' => ['required_without:listing_id', 'nullable', 'integer', 'min:1800', 'max:2100'],

            // Optional fields ("Tell us more" expandable section)
            'garage_spaces' => ['nullable', 'integer', 'min:0', 'max:20'],
            'pool' => ['nullable', 'boolean'],
            'waterfront' => ['nullable', 'boolean'],
            'lot_size_sqft' => ['nullable', 'integer', 'min:0'],
            'hoa_monthly_fee' => ['nullable', 'numeric', 'min:0'],
            'stories' => ['nullable', 'integer', 'min:1', 'max:5'],
            'roof_year_replaced' => ['nullable', 'integer', 'min:1800', 'max:2100'],
            'renovated_kitchen_year' => ['nullable', 'integer', 'min:1980', 'max:2100'],
            'renovated_bathrooms_year' => ['nullable', 'integer', 'min:1980', 'max:2100'],
            'renovated_hvac_year' => ['nullable', 'integer', 'min:1980', 'max:2100'],
            'enclosed_lanai_sqft' => ['nullable', 'integer', 'min:0'],
            'screen_pool_enclosure' => ['nullable', 'boolean'],
        ];
    }
}
