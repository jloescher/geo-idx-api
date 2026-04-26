<?php

namespace Tests\Feature\Api;

use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Illuminate\Testing\TestResponse;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class SearchControllerTest extends TestCase
{
    use RefreshDatabase;

    private function actingAsWithToken(User $user, string $name, array $abilities): string
    {
        $token = $user->createToken($name, $abilities);

        return $token->plainTextToken;
    }

    private function createActiveSubscription(User $user, string $planKey): void
    {
        $priceMap = [
            'pro' => 'price_pro_monthly',
            'smart' => 'price_smart_monthly',
            'ultra' => 'price_ultra_monthly',
            'mega' => 'price_mega_monthly',
        ];
        $priceId = $priceMap[$planKey] ?? 'price_pro_monthly';

        config(["billing.plans.{$planKey}.stripe_price_monthly" => $priceId]);

        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_test_'.$user->id,
            'stripe_status' => 'active',
            'stripe_price' => $priceId,
            'quantity' => 1,
            'trial_ends_at' => now()->addDays(7),
            'ends_at' => null,
        ]);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_test_'.$user->id,
            'stripe_product' => 'prod_test',
            'stripe_price' => $priceId,
            'quantity' => 100,
        ]);
    }

    private function fakeBridgePropertyResponse(array $properties = []): void
    {
        Http::fake([
            '*/OData/stellar/Property*' => Http::response([
                'value' => $properties,
                '@odata.count' => count($properties),
            ], 200),
            '*/OData/*' => Http::response([
                'value' => $properties,
                '@odata.count' => count($properties),
            ], 200),
        ]);
    }

    private function sampleBridgeProperty(string $listingKey = 'stellar:12345'): array
    {
        return [
            'ListingKey' => $listingKey,
            'StandardStatus' => 'Active',
            'ListPrice' => 350000,
            'BedroomsTotal' => 3,
            'BathroomsTotalDecimal' => 2.0,
            'LivingArea' => 1800,
            'City' => 'Tampa',
            'StateOrProvince' => 'FL',
            'PostalCode' => '33602',
            'StreetNumber' => '123',
            'StreetName' => 'Main St',
            'PropertyType' => 'Residential',
            'OnMarketDate' => now()->subDays(5)->format('Y-m-d'),
            'Coordinates' => ['coordinates' => [-82.45, 27.95]],
        ];
    }

    private function requestSearch(string $token, array $body = []): TestResponse
    {
        return $this->withHeader('Authorization', 'Bearer '.$token)
            ->postJson('/api/v1/search', $body);
    }

    public function test_unauthenticated_request_is_rejected(): void
    {
        $this->fakeBridgePropertyResponse();

        $response = $this->postJson('/api/v1/search');
        $response->assertUnauthorized();
    }

    public function test_search_with_no_filters_returns_active_listings(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        $response = $this->requestSearch($token);

        $response->assertOk();
        $response->assertJsonStructure([
            'total_count',
            'results',
            'has_more',
            'next_skip',
            'stats',
        ]);
    }

    public function test_search_with_price_filter(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        $response = $this->requestSearch($token, [
            'min_price' => 200000,
            'max_price' => 500000,
        ]);

        $response->assertOk();
        $response->assertJsonCount(1, 'results');
    }

    public function test_search_with_location_filter(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        $response = $this->requestSearch($token, [
            'city' => 'Tampa',
            'state' => 'FL',
        ]);

        $response->assertOk();
    }

    public function test_search_with_geo_distance_filter(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        $response = $this->requestSearch($token, [
            'geo' => [
                'distance' => [
                    'lat' => 27.95,
                    'lng' => -82.45,
                    'radius_miles' => 5,
                ],
            ],
        ]);

        $response->assertOk();
    }

    public function test_search_with_focus_areas_filter(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        $response = $this->requestSearch($token, [
            'focus_areas' => [
                ['type' => 'city', 'name' => 'Tampa'],
                ['type' => 'state', 'name' => 'FL'],
            ],
        ]);

        $response->assertOk();
    }

    public function test_search_with_low_risk_floodzone_filter(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([
            array_merge($this->sampleBridgeProperty(), ['STELLAR_FloodZoneCode' => 'X']),
            array_merge($this->sampleBridgeProperty('stellar:99999'), ['STELLAR_FloodZoneCode' => 'X, AE']),
        ]);

        $response = $this->requestSearch($token, [
            'low_risk_floodzone' => true,
        ]);

        $response->assertOk();
        $response->assertJsonCount(1, 'results');
    }

    public function test_search_with_pagination(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        $response = $this->requestSearch($token, [
            'page' => ['limit' => 10, 'skip' => 20],
        ]);

        $response->assertOk();
    }

    public function test_search_with_sort(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        $response = $this->requestSearch($token, [
            'sort' => 'list_price',
            'sort_dir' => 'asc',
        ]);

        $response->assertOk();
    }

    public function test_search_with_dataset_query_param(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        config(['bridge.datasets' => ['stellar', 'miami']]);

        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        $response = $this->requestSearch($token, []);
        $response->assertOk();
    }

    public function test_search_rejects_invalid_dataset(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        config(['bridge.datasets' => ['stellar']]);

        $this->fakeBridgePropertyResponse();

        $response = $this->withHeader('Authorization', 'Bearer '.$token)
            ->postJson('/api/v1/search?dataset=invalid');

        $response->assertStatus(400);
    }

    public function test_search_respects_teaser_gating_for_ultra(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'ultra');
        $token = $this->actingAsWithToken($user, 'ultra-token', ['idx:access']);

        $this->fakeBridgePropertyResponse([
            $this->sampleBridgeProperty('stellar:1'),
            $this->sampleBridgeProperty('stellar:2'),
            $this->sampleBridgeProperty('stellar:3'),
            $this->sampleBridgeProperty('stellar:4'),
            $this->sampleBridgeProperty('stellar:5'),
        ]);

        $response = $this->requestSearch($token, [
            'page' => ['limit' => 20],
        ]);

        $response->assertOk();
        $response->assertJsonCount(3, 'results');
    }

    public function test_search_returns_full_results_for_mega(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([
            $this->sampleBridgeProperty('stellar:1'),
            $this->sampleBridgeProperty('stellar:2'),
            $this->sampleBridgeProperty('stellar:3'),
        ]);

        $response = $this->requestSearch($token);

        $response->assertOk();
        $response->assertJsonCount(3, 'results');
    }

    public function test_search_returns_correct_listing_result_fields(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        $response = $this->requestSearch($token);

        $response->assertOk();
        $response->assertJsonStructure([
            'results' => [
                '*' => [
                    'listingId',
                    'standardStatus',
                    'listPrice',
                    'bedroomsTotal',
                    'bathroomsTotal',
                    'livingArea',
                    'city',
                    'state',
                    'fullAddress',
                ],
            ],
        ]);
    }

    public function test_search_returns_stats(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([
            $this->sampleBridgeProperty('stellar:1'),
            $this->sampleBridgeProperty('stellar:2'),
        ]);

        $response = $this->requestSearch($token);

        $response->assertOk();
        $response->assertJsonStructure([
            'stats' => [
                'result_count',
                'avg_dom',
                'avg_price',
                'median_price',
            ],
        ]);
    }

    public function test_search_with_boolean_filters(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([
            array_merge($this->sampleBridgeProperty(), [
                'WaterfrontYN' => true,
                'PoolPrivateYN' => true,
                'GarageYN' => true,
            ]),
        ]);

        $response = $this->requestSearch($token, [
            'waterfront' => true,
            'pool_private' => true,
            'garage' => true,
        ]);

        $response->assertOk();
    }

    public function test_search_with_special_listing_conditions(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([
            array_merge($this->sampleBridgeProperty(), [
                'SpecialListingConditions' => ['Short Sale'],
            ]),
        ]);

        $response = $this->requestSearch($token, [
            'special_listing_conditions' => ['Short Sale'],
        ]);

        $response->assertOk();
    }

    public function test_search_with_price_reduced_within_days(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([
            array_merge($this->sampleBridgeProperty(), [
                'PriceChangeTimestamp' => now()->subDays(10)->format('Y-m-d\TH:i:sP'),
                'ListPrice' => 300000,
                'PreviousListPrice' => 350000,
            ]),
        ]);

        $response = $this->requestSearch($token, [
            'price_reduced_within_days' => 30,
        ]);

        $response->assertOk();
    }

    public function test_search_empty_results(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        $this->fakeBridgePropertyResponse([]);

        $response = $this->requestSearch($token);

        $response->assertOk();
        $response->assertJsonCount(0, 'results');
    }

    public function test_identical_search_requests_use_single_bridge_http_call(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->createActiveSubscription($user, 'mega');
        $token = $this->actingAsWithToken($user, 'mega-token', ['idx:full']);

        Http::fake([
            '*/OData/stellar/Property*' => Http::response([
                'value' => [$this->sampleBridgeProperty()],
            ], 200),
        ]);

        $body = ['min_price' => 250000];
        $this->requestSearch($token, $body)->assertOk();
        $this->requestSearch($token, $body)->assertOk();

        $this->assertCount(1, Http::recorded());
    }

    public function test_search_rejects_dataset_not_enabled_for_domain(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami']]);

        Domain::query()->create([
            'domain_slug' => 'dataset-gate.example.com',
            'is_active' => true,
            'allowed_mls_datasets' => ['stellar'],
            'mls_dataset' => null,
        ]);

        Http::fake([
            '*/OData/*' => Http::response(['value' => []], 200),
        ]);

        $this->withHeader('X-Domain-Slug', 'dataset-gate.example.com')
            ->postJson('/api/v1/search?dataset=miami', [])
            ->assertForbidden();
    }
}
