<?php

namespace Tests\Feature\Widget;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class WidgetHomeValueTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.dataset' => 'stellar',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'geocoding.google_api_key' => 'test-google-geocoding-key',
        ]);
    }

    private function createRegisteredUrl(): GhlRegisteredUrl
    {
        $access = 'access-whv-'.uniqid('', true);
        $refresh = 'refresh-whv-'.uniqid('', true);

        $oauth = new GhlOAuthToken([
            'ghl_company_id' => 'co-whv',
            'ghl_location_id' => 'loc-whv',
            'ghl_user_id' => 'u-whv',
            'user_type' => 'Location',
            'scopes' => 'contacts.write',
            'is_bulk_install' => false,
            'expires_at' => now()->addDay(),
            'status' => 'active',
            'access_token_hash' => hash('sha256', $access),
        ]);
        $oauth->access_token = $access;
        $oauth->refresh_token = $refresh;
        $oauth->save();

        return GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $oauth->id,
            'ghl_location_id' => 'loc-whv',
            'ghl_company_id' => 'co-whv',
            'primary_url' => 'https://example.com',
            'widget_api_key' => 'wak_test_home_value_123',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);
    }

    private function validHomeValuePayload(): array
    {
        return [
            'address' => '100 Main St, Tampa, FL 33602',
            'property_type' => 'sfr',
            'bedrooms' => 3,
            'full_bathrooms' => 2,
            'half_bathrooms' => 1,
            'living_area_sqft' => 1800,
            'condition' => 'good',
            'year_built' => 2000,
            'api_key' => 'wak_test_home_value_123',
        ];
    }

    private function fakeComps(): array
    {
        $comps = [];
        for ($i = 0; $i < 6; $i++) {
            $comps[] = [
                'ListingKey' => 'stellar:whv'.$i,
                'StandardStatus' => 'Closed',
                'ClosePrice' => 380000 + ($i * 15000),
                'CloseDate' => now()->subMonths(2 + $i)->format('Y-m-d'),
                'LivingArea' => 1700 + ($i * 80),
                'BedroomsTotal' => 3,
                'BathroomsTotalDecimal' => 2,
                'YearBuilt' => 2000 + $i,
                'LotSizeAcres' => 0.22,
                'PoolPrivateYN' => $i % 2 === 0,
                'GarageSpaces' => 2,
                'WaterfrontYN' => false,
                'PropertySubType' => 'Single Family Residence',
                'Coordinates' => ['coordinates' => [-82.45, 27.95 + ($i * 0.002)]],
                'StreetNumber' => (string) (200 + $i),
                'StreetName' => 'Oak',
                'City' => 'Tampa',
                'StateOrProvince' => 'FL',
                'PostalCode' => '33602',
                'DaysOnMarket' => 20 + $i,
                'CumulativeDaysOnMarket' => 20 + $i,
                'PublicRemarks' => 'Nice home.',
                'STELLAR_FloodZoneCode' => 'X',
                'STELLAR_TotalMonthlyFees' => 0,
            ];
        }

        return $comps;
    }

    public function test_widget_home_value_returns_estimate_for_valid_address(): void
    {
        $this->createRegisteredUrl();

        Http::fake([
            'maps.googleapis.com/*' => Http::response([
                'status' => 'OK',
                'results' => [[
                    'geometry' => ['location' => ['lat' => 27.95, 'lng' => -82.45]],
                    'formatted_address' => '100 Main St, Tampa, FL 33602, USA',
                    'place_id' => 'test_place_id',
                ]],
            ], 200),
            'bridge.test/*' => Http::response(['value' => $this->fakeComps()], 200),
        ]);

        $response = $this->postJson('/widget/api/home-value', $this->validHomeValuePayload(), [
            'Origin' => 'https://example.com',
            'X-Quantyra-Widget-Key' => 'wak_test_home_value_123',
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $this->assertNotNull($response->json('estimated_value'));
        $this->assertNotNull($response->json('value_range.low'));
        $this->assertNotNull($response->json('value_range.high'));
        $this->assertContains($response->json('confidence'), ['high', 'moderate', 'low']);
        $this->assertIsInt($response->json('comparable_count'));
        $this->assertIsArray($response->json('comps_summary'));
    }

    public function test_widget_home_value_requires_widget_api_key(): void
    {
        $response = $this->postJson('/widget/api/home-value', [
            'address' => '100 Main St, Tampa, FL 33602',
            'property_type' => 'sfr',
            'bedrooms' => 3,
            'full_bathrooms' => 2,
            'living_area_sqft' => 1800,
            'condition' => 'good',
            'year_built' => 2000,
        ], [
            'Origin' => 'https://example.com',
        ]);

        $response->assertStatus(401);
    }

    public function test_widget_home_value_validates_required_fields(): void
    {
        $this->createRegisteredUrl();

        $response = $this->postJson('/widget/api/home-value', [
            'api_key' => 'wak_test_home_value_123',
        ], [
            'Origin' => 'https://example.com',
            'X-Quantyra-Widget-Key' => 'wak_test_home_value_123',
        ]);

        $response->assertStatus(422);
        $response->assertJsonValidationErrors([
            'address',
            'property_type',
            'bedrooms',
            'full_bathrooms',
            'living_area_sqft',
            'condition',
            'year_built',
        ]);
    }

    public function test_widget_home_value_optional_fields_accepted(): void
    {
        $this->createRegisteredUrl();

        Http::fake([
            'maps.googleapis.com/*' => Http::response([
                'status' => 'OK',
                'results' => [[
                    'geometry' => ['location' => ['lat' => 27.95, 'lng' => -82.45]],
                    'formatted_address' => '100 Main St, Tampa, FL 33602, USA',
                    'place_id' => 'test_place_id',
                ]],
            ], 200),
            'bridge.test/*' => Http::response(['value' => $this->fakeComps()], 200),
        ]);

        $payload = array_merge($this->validHomeValuePayload(), [
            'garage_spaces' => 2,
            'pool' => true,
            'waterfront' => false,
            'lot_size_sqft' => 8000,
            'hoa_monthly_fee' => 150,
            'stories' => 1,
            'renovated_kitchen_year' => 2023,
            'renovated_bathrooms_year' => 2023,
            'renovated_hvac_year' => 2022,
            'enclosed_lanai_sqft' => 200,
            'screen_pool_enclosure' => true,
        ]);

        $response = $this->postJson('/widget/api/home-value', $payload, [
            'Origin' => 'https://example.com',
            'X-Quantyra-Widget-Key' => 'wak_test_home_value_123',
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
    }
}
