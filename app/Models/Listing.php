<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;

/**
 * Revenue impact: local mirror rows fuel PostGIS-backed search Latency SLA for map-heavy
 * traffic without extra Bridge OData round trips until filters require Bridge fallback.
 */
#[Fillable([
    'dataset_slug',
    'listing_key',
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
    'stellar_flood_zone_code',
    'stellar_total_monthly_fees',
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
])]
class Listing extends Model
{
    protected $table = 'listings';

    /**
     * @return array<string, string>
     */
    public function casts(): array
    {
        return [
            'list_price' => 'decimal:2',
            'bathrooms_total_decimal' => 'decimal:2',
            'lot_size_acres' => 'decimal:4',
            'living_area' => 'integer',
            'bedrooms_total' => 'integer',
            'year_built' => 'integer',
            'stories_total' => 'integer',
            'on_market_date' => 'date',
            'close_date' => 'date',
            'modification_timestamp' => 'immutable_datetime',
            'bridge_modification_timestamp' => 'immutable_datetime',
            'price_change_timestamp' => 'immutable_datetime',
            'previous_list_price' => 'decimal:2',
            'stellar_total_monthly_fees' => 'decimal:2',
            'latitude' => 'double',
            'longitude' => 'double',
            'waterfront_yn' => 'boolean',
            'pool_private_yn' => 'boolean',
            'dock_yn' => 'boolean',
            'new_construction_yn' => 'boolean',
            'garage_yn' => 'boolean',
            'association_yn' => 'boolean',
            'spa_yn' => 'boolean',
            'fireplace_yn' => 'boolean',
            'senior_community_yn' => 'boolean',
            'special_listing_conditions' => 'array',
            'raw_data' => 'array',
            'custom_fields' => 'array',
        ];
    }
}
