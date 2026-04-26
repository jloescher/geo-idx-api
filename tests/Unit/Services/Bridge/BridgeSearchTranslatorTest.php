<?php

namespace Tests\Unit\Services\Bridge;

use App\Http\Requests\Search\SearchRequest;
use App\Services\Bridge\BridgeSearchTranslator;
use Illuminate\Support\Facades\Validator;
use Tests\TestCase;

class BridgeSearchTranslatorTest extends TestCase
{
    private BridgeSearchTranslator $translator;

    protected function setUp(): void
    {
        parent::setUp();
        $this->translator = new BridgeSearchTranslator;
    }

    /**
     * Create a validated SearchRequest from raw input data.
     */
    private function makeRequest(array $input): SearchRequest
    {
        $rules = (new SearchRequest)->rules();
        $validator = Validator::make($input, $rules);
        $validated = $validator->validated();

        $request = new SearchRequest;
        $request->replace($input);
        $request->setContainer(app());
        $request->setRedirector(app('redirect'));
        $request->merge($validated);

        // Replace the validated() method behavior by setting the validator
        $request->setValidator($validator);

        return $request;
    }

    public function test_empty_params_produces_no_filter(): void
    {
        $request = $this->makeRequest([]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString("tolower(StandardStatus) eq 'active'", $result['filter']);
        $this->assertEmpty($result['orderby']);
        $this->assertSame(24, $result['top']);
        $this->assertSame(0, $result['skip']);
        $this->assertStringContainsString('ListingKey', $result['select']);
    }

    public function test_price_range_filter(): void
    {
        $request = $this->makeRequest(['min_price' => 200000, 'max_price' => 500000]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('ListPrice ge 200000', $result['filter']);
        $this->assertStringContainsString('ListPrice le 500000', $result['filter']);
    }

    public function test_beds_and_baths_filter(): void
    {
        $request = $this->makeRequest(['min_beds' => 3, 'max_beds' => 5, 'min_baths' => 2, 'max_baths' => 3.5]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('BedroomsTotal ge 3', $result['filter']);
        $this->assertStringContainsString('BedroomsTotal le 5', $result['filter']);
        $this->assertStringContainsString('BathroomsTotalDecimal ge 2', $result['filter']);
        $this->assertStringContainsString('BathroomsTotalDecimal le 3.5', $result['filter']);
    }

    public function test_sqft_and_lot_size_filter(): void
    {
        $request = $this->makeRequest(['min_sqft' => 1500, 'max_sqft' => 3000, 'min_lot_size_acres' => 0.5]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('LivingArea ge 1500', $result['filter']);
        $this->assertStringContainsString('LivingArea le 3000', $result['filter']);
        $this->assertStringContainsString('LotSizeAcres ge 0.5', $result['filter']);
    }

    public function test_year_built_filter(): void
    {
        $request = $this->makeRequest(['min_year_built' => 1990, 'max_year_built' => 2020]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('YearBuilt ge 1990', $result['filter']);
        $this->assertStringContainsString('YearBuilt le 2020', $result['filter']);
    }

    public function test_stories_filter(): void
    {
        $request = $this->makeRequest(['min_stories' => 2, 'max_stories' => 3]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('StoriesTotal ge 2', $result['filter']);
        $this->assertStringContainsString('StoriesTotal le 3', $result['filter']);
    }

    public function test_monthly_fees_filter_uses_dataset_prefix(): void
    {
        $request = $this->makeRequest(['min_monthly_fees' => 100, 'max_monthly_fees' => 500]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('STELLAR_TotalMonthlyFees ge 100', $result['filter']);
        $this->assertStringContainsString('STELLAR_TotalMonthlyFees le 500', $result['filter']);

        $result2 = $this->translator->translate($this->makeRequest(['min_monthly_fees' => 100, 'max_monthly_fees' => 500]), 'miami');
        $this->assertStringContainsString('MIAMI_TotalMonthlyFees ge 100', $result2['filter']);
    }

    public function test_boolean_filters(): void
    {
        $request = $this->makeRequest([
            'waterfront' => true,
            'pool_private' => true,
            'dock' => true,
            'new_construction' => true,
            'garage' => true,
            'association' => true,
            'spa' => true,
            'fireplace' => true,
            'active_adult_community' => true,
        ]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('WaterfrontYN eq true', $result['filter']);
        $this->assertStringContainsString('PoolPrivateYN eq true', $result['filter']);
        $this->assertStringContainsString('DockYN eq true', $result['filter']);
        $this->assertStringContainsString('NewConstructionYN eq true', $result['filter']);
        $this->assertStringContainsString('GarageYN eq true', $result['filter']);
        $this->assertStringContainsString('AssociationYN eq true', $result['filter']);
        $this->assertStringContainsString('SpaYN eq true', $result['filter']);
        $this->assertStringContainsString('FireplaceYN eq true', $result['filter']);
        $this->assertStringContainsString('SeniorCommunityYN eq true', $result['filter']);
    }

    public function test_for_lease_filter(): void
    {
        $request = $this->makeRequest(['for_lease' => true]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString("PropertyType eq 'Residential Lease'", $result['filter']);
    }

    public function test_low_risk_floodzone_filter(): void
    {
        $request = $this->makeRequest(['low_risk_floodzone' => true]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString("contains(tolower(STELLAR_FloodZoneCode), 'x')", $result['filter']);
        $this->assertTrue($result['needsFloodZonePostFilter']);
        $this->assertTrue($result['lowRiskFloodzone']);
    }

    public function test_location_string_filters(): void
    {
        $request = $this->makeRequest([
            'city' => 'Tampa',
            'state' => 'FL',
            'county' => 'Hillsborough',
            'postal_code' => '33602',
        ]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString("tolower(City) eq 'tampa'", $result['filter']);
        $this->assertStringContainsString("StateOrProvince eq 'FL'", $result['filter']);
        $this->assertStringContainsString("tolower(CountyOrParish) eq 'hillsborough'", $result['filter']);
        $this->assertStringContainsString("PostalCode eq '33602'", $result['filter']);
    }

    public function test_property_types_or_filter(): void
    {
        $request = $this->makeRequest(['property_types' => ['Residential', 'Land']]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString("(PropertyType eq 'Residential' or PropertyType eq 'Land')", $result['filter']);
    }

    public function test_property_sub_types_or_filter(): void
    {
        $request = $this->makeRequest(['property_sub_types' => ['Single Family Residence', 'Condo']]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString("(PropertySubType eq 'Single Family Residence' or PropertySubType eq 'Condo')", $result['filter']);
    }

    public function test_statuses_filter(): void
    {
        $request = $this->makeRequest(['statuses' => ['Active', 'Pending']]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString("(tolower(StandardStatus) eq 'active' or tolower(StandardStatus) eq 'pending')", $result['filter']);
    }

    public function test_special_listing_conditions_any_filter(): void
    {
        $request = $this->makeRequest(['special_listing_conditions' => ['Short Sale', 'REO']]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString("(SpecialListingConditions/any(c: c eq 'Short Sale') or SpecialListingConditions/any(c: c eq 'REO'))", $result['filter']);
    }

    public function test_focus_areas_filter(): void
    {
        $request = $this->makeRequest([
            'focus_areas' => [
                ['type' => 'city', 'name' => 'Tampa'],
                ['type' => 'state', 'name' => 'FL'],
            ],
        ]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString("tolower(City) eq 'tampa'", $result['filter']);
        $this->assertStringContainsString("StateOrProvince eq 'FL'", $result['filter']);
    }

    public function test_geo_distance_filter(): void
    {
        $request = $this->makeRequest([
            'geo' => [
                'distance' => ['lat' => 27.95, 'lng' => -82.45, 'radius_miles' => 5],
            ],
        ]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('geo.distance(Coordinates, POINT(-82.45 27.95)) lt 5', $result['filter']);
    }

    public function test_geo_bbox_filter(): void
    {
        $request = $this->makeRequest([
            'geo' => [
                'bbox' => ['west' => -82.8, 'south' => 27.9, 'east' => -82.4, 'north' => 28.2],
            ],
        ]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('Latitude ge 27.9 and Latitude le 28.2', $result['filter']);
        $this->assertStringContainsString('Longitude ge -82.8 and Longitude le -82.4', $result['filter']);
    }

    public function test_sort_translation(): void
    {
        $request = $this->makeRequest(['sort' => 'list_price', 'sort_dir' => 'asc']);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertSame('ListPrice asc', $result['orderby']);

        $request2 = $this->makeRequest(['sort' => 'year_built', 'sort_dir' => 'desc']);
        $result2 = $this->translator->translate($request2, 'stellar');
        $this->assertSame('YearBuilt desc', $result2['orderby']);
    }

    public function test_pagination(): void
    {
        $request = $this->makeRequest(['page' => ['limit' => 50, 'skip' => 100]]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertSame(50, $result['top']);
        $this->assertSame(100, $result['skip']);
    }

    public function test_select_includes_dataset_specific_fields(): void
    {
        $request = $this->makeRequest([]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('STELLAR_TotalMonthlyFees', $result['select']);
        $this->assertStringContainsString('STELLAR_FloodZoneCode', $result['select']);
    }

    public function test_unselect_excludes_heavy_fields(): void
    {
        $request = $this->makeRequest([]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString('PublicRemarks', $result['unselect']);
        $this->assertStringContainsString('PrivateRemarks', $result['unselect']);
    }

    public function test_combined_filters_with_and_operator(): void
    {
        $request = $this->makeRequest([
            'min_price' => 300000,
            'max_price' => 600000,
            'min_beds' => 3,
            'city' => 'Tampa',
            'waterfront' => true,
        ]);
        $result = $this->translator->translate($request, 'stellar');

        $this->assertStringContainsString(' and ', $result['filter']);
        $this->assertStringContainsString('ListPrice ge 300000', $result['filter']);
        $this->assertStringContainsString('ListPrice le 600000', $result['filter']);
        $this->assertStringContainsString('BedroomsTotal ge 3', $result['filter']);
        $this->assertStringContainsString("tolower(City) eq 'tampa'", $result['filter']);
        $this->assertStringContainsString('WaterfrontYN eq true', $result['filter']);
    }

    public function test_flood_zone_post_filter_excludes_high_risk(): void
    {
        $records = [
            ['STELLAR_FloodZoneCode' => 'X'],
            ['STELLAR_FloodZoneCode' => 'X, AE'],
            ['STELLAR_FloodZoneCode' => 'AE'],
            ['STELLAR_FloodZoneCode' => 'B'],
            ['STELLAR_FloodZoneCode' => 'V'],
            ['STELLAR_FloodZoneCode' => ''],
            ['STELLAR_FloodZoneCode' => 'X, A'],
        ];

        $filtered = $this->translator->filterLowRiskFloodZone($records, 'stellar');

        $this->assertCount(2, $filtered);
        $this->assertSame('X', $filtered[0]['STELLAR_FloodZoneCode']);
        $this->assertSame('B', $filtered[1]['STELLAR_FloodZoneCode']);
    }

    public function test_flood_zone_post_filter_empty_array(): void
    {
        $this->assertSame([], $this->translator->filterLowRiskFloodZone([], 'stellar'));
    }

    public function test_sort_distance_requires_geo_params(): void
    {
        $request = $this->makeRequest(['sort' => 'distance']);
        $result = $this->translator->translate($request, 'stellar');
        $this->assertEmpty($result['orderby']);
    }
}
