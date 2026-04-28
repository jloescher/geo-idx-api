<?php

namespace Tests\Feature\Api;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class BridgeCompsTest extends TestCase
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
        ]);
    }

    private function actingAsWithToken(User $user, string $name, array $abilities): string
    {
        return $user->createToken($name, $abilities)->plainTextToken;
    }

    private function subscribeMega(User $user): void
    {
        $priceId = 'price_mega_monthly';
        config(['billing.plans.mega.stripe_price_monthly' => $priceId]);

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

    /**
     * @return array<string, mixed>
     */
    private function subjectProperty(string $listingKey = 'stellar:12345'): array
    {
        return [
            'ListingKey' => $listingKey,
            'StandardStatus' => 'Active',
            'ListPrice' => 450000,
            'ClosePrice' => null,
            'OriginalListPrice' => 460000,
            'PreviousListPrice' => null,
            'CloseDate' => null,
            'OnMarketDate' => now()->subDays(10)->format('Y-m-d'),
            'BedroomsTotal' => 3,
            'BathroomsTotalDecimal' => 2,
            'LivingArea' => 1800,
            'LotSizeAcres' => 0.25,
            'YearBuilt' => 2010,
            'StoriesTotal' => 1,
            'City' => 'Tampa',
            'StateOrProvince' => 'FL',
            'PostalCode' => '33602',
            'CountyOrParish' => 'Hillsborough',
            'PropertyType' => 'Residential',
            'PropertySubType' => 'Single Family Residence',
            'WaterfrontYN' => false,
            'ViewYN' => false,
            'PoolPrivateYN' => true,
            'GarageYN' => true,
            'GarageSpaces' => 2,
            'CarportSpaces' => 0,
            'CoveredSpaces' => 0,
            'OpenParkingSpaces' => null,
            'AssociationYN' => false,
            'SeniorCommunityYN' => false,
            'StreetNumber' => '100',
            'StreetName' => 'Main',
            'SubdivisionName' => 'Comps Test Subdivision',
            'MLSAreaMajor' => '33602 - Tampa',
            'Coordinates' => ['coordinates' => [-82.45, 27.95]],
            'DaysOnMarket' => 12,
            'CumulativeDaysOnMarket' => 12,
            'PublicRemarks' => 'Lovely updated kitchen.',
            'STELLAR_FloodZoneCode' => 'X',
            'STELLAR_TotalMonthlyFees' => 0,
        ];
    }

    /**
     * @return array<string, mixed>
     */
    private function closedComp(string $key, float $distLatOffset = 0.01): array
    {
        return [
            'ListingKey' => $key,
            'StandardStatus' => 'Closed',
            'ListPrice' => 400000,
            'ClosePrice' => 395000,
            'OriginalListPrice' => 410000,
            'PreviousListPrice' => 405000,
            'CloseDate' => now()->subMonths(2)->format('Y-m-d'),
            'OnMarketDate' => now()->subMonths(3)->format('Y-m-d'),
            'BedroomsTotal' => 3,
            'BathroomsTotalDecimal' => 2,
            'LivingArea' => 1750,
            'LotSizeAcres' => 0.22,
            'YearBuilt' => 2008,
            'StoriesTotal' => 1,
            'City' => 'Tampa',
            'StateOrProvince' => 'FL',
            'PostalCode' => '33602',
            'CountyOrParish' => 'Hillsborough',
            'PropertyType' => 'Residential',
            'PropertySubType' => 'Single Family Residence',
            'WaterfrontYN' => false,
            'ViewYN' => false,
            'PoolPrivateYN' => true,
            'GarageYN' => true,
            'GarageSpaces' => 1,
            'CarportSpaces' => 0,
            'CoveredSpaces' => 0,
            'OpenParkingSpaces' => null,
            'AssociationYN' => false,
            'SeniorCommunityYN' => false,
            'StreetNumber' => '200',
            'StreetName' => 'Oak',
            'SubdivisionName' => 'Comps Test Subdivision',
            'MLSAreaMajor' => '33602 - Tampa',
            'Coordinates' => ['coordinates' => [-82.45, 27.95 + $distLatOffset]],
            'DaysOnMarket' => 30,
            'CumulativeDaysOnMarket' => 30,
            'PublicRemarks' => 'Move-in ready.',
            'STELLAR_FloodZoneCode' => 'X',
            'STELLAR_TotalMonthlyFees' => 0,
        ];
    }

    /**
     * @return array<string, mixed>
     */
    private function leaseComp(string $key, string $status = 'Closed', float $rent = 2200): array
    {
        return [
            'ListingKey' => $key,
            'StandardStatus' => $status,
            'PropertyType' => 'Residential Lease',
            'PropertySubType' => 'Single Family Residence',
            'ListPrice' => $rent,
            'ClosePrice' => $status === 'Closed' ? $rent : null,
            'CloseDate' => now()->subDays(20)->format('Y-m-d'),
            'OnMarketDate' => now()->subDays(45)->format('Y-m-d'),
            'LeaseAmountFrequency' => 'Monthly',
            'BedroomsTotal' => 3,
            'BathroomsTotalDecimal' => 2,
            'LivingArea' => 1700,
            'YearBuilt' => 2009,
            'LotSizeAcres' => 0.2,
            'Coordinates' => [-82.44, 27.951],
            'StreetNumber' => '300',
            'StreetName' => 'Palm',
            'City' => 'Tampa',
            'StateOrProvince' => 'FL',
            'PostalCode' => '33602',
            'DaysOnMarket' => 25,
            'CumulativeDaysOnMarket' => 25,
            'STELLAR_FloodZoneCode' => 'X',
            'STELLAR_TotalMonthlyFees' => 0,
        ];
    }

    public function test_comps_requires_authentication(): void
    {
        Http::fake();

        $this->postJson('/api/v1/comps/run', [])->assertUnauthorized();
    }

    public function test_comps_validation_errors(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $this->postJson('/api/v1/comps/run', [], [
            'Authorization' => 'Bearer '.$token,
        ])->assertStatus(422);
    }

    public function test_comps_subject_not_found_returns_payload(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        Http::fake(function (Request $request) {
            if (preg_match('#/Property\\([^?]+\\)#', $request->url()) === 1) {
                return Http::response([], 404);
            }

            return Http::response(['value' => []], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'missing-key'],
            'mode' => 'A',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
            'filters' => ['sold_months_back' => 6, 'max_sold_comps' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJson([
            'success' => false,
            'error' => 'Subject listing not found',
        ]);
    }

    public function test_comps_returns_sold_comps_with_full_token(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subject = $this->subjectProperty();
        $compA = $this->closedComp('stellar:90001', 0.005);
        $compB = $this->closedComp('stellar:90002', 0.008);

        Http::fake(function (Request $request) use ($subject, $compA, $compB) {
            if (preg_match('#/Property\\([^?]+\\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }

            return Http::response([
                'value' => [$compA, $compB],
            ], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'stellar:12345'],
            'mode' => 'A',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
            'filters' => [
                'sold_months_back' => 12,
                'max_sold_comps' => 10,
                'include_active_pending' => false,
            ],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $response->assertJsonPath('subject.listing_id', '12345');
        $sold = $response->json('sold_comps');
        $this->assertIsArray($sold);
        $this->assertGreaterThanOrEqual(1, count($sold));
        $this->assertArrayHasKey('similarity_score', $sold[0]);
        $this->assertArrayHasKey('adjustments', $sold[0]);
        $this->assertNotNull($response->json('market_conditions.median_sold_ppsf'));
        $this->assertGreaterThan(0, (float) $response->json('market_conditions.median_sold_ppsf'));
        $response->assertJsonPath('metadata.aggregation_method', 'median');
    }

    public function test_comps_teaser_cap_for_idx_access_token(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:access']);

        $subject = $this->subjectProperty();
        $many = [];
        for ($i = 0; $i < 15; $i++) {
            $many[] = $this->closedComp('stellar:9'.str_pad((string) $i, 4, '0', STR_PAD_LEFT), $i * 0.001);
        }

        Http::fake(function (Request $request) use ($subject, $many) {
            if (preg_match('#/Property\\([^?]+\\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }

            return Http::response(['value' => $many], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => '12345'],
            'mode' => 'B',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
            'filters' => ['max_sold_comps' => 12, 'include_active_pending' => true],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $this->assertLessThanOrEqual(3, count($response->json('sold_comps')));
        $this->assertSame([], $response->json('competition_comps'));
    }

    public function test_comps_applies_garage_stall_adjustment_when_counts_differ(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subject = $this->subjectProperty();
        $comp = $this->closedComp('stellar:90001', 0.005);

        Http::fake(function (Request $request) use ($subject, $comp) {
            if (preg_match('#/Property\\([^?]+\\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }

            return Http::response(['value' => [$comp]], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'stellar:12345'],
            'mode' => 'A',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
            'filters' => [
                'sold_months_back' => 12,
                'max_sold_comps' => 5,
                'include_active_pending' => false,
                'adj_garage_per_space' => 25_000,
            ],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $sold = $response->json('sold_comps');
        $this->assertNotEmpty($sold);
        $lines = $sold[0]['adjustments']['lines'] ?? [];
        $garageLine = collect($lines)->firstWhere('feature', 'garage');
        $this->assertNotNull($garageLine);
        $this->assertSame(25_000, $garageLine['adjustment']);
    }

    public function test_comps_returns_flood_zone_codes_on_sold_comp(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subject = $this->subjectProperty();
        $comp = $this->closedComp('stellar:90001', 0.005);
        $comp['STELLAR_FloodZoneCode'] = 'X, B';

        Http::fake(function (Request $request) use ($subject, $comp) {
            if (preg_match('#/Property\\([^?]+\\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }

            return Http::response(['value' => [$comp]], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'stellar:12345'],
            'mode' => 'A',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
            'filters' => ['sold_months_back' => 12, 'max_sold_comps' => 5, 'include_active_pending' => false],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $this->assertSame(['X', 'B'], $response->json('sold_comps.0.flood_zone_codes'));
    }

    public function test_comps_rent_hold_cashflow_returns_rental_result(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subject = $this->subjectProperty();
        $leaseClosed = $this->leaseComp('stellar:lease1', 'Closed', 2400);
        $leaseActive = $this->leaseComp('stellar:lease2', 'Active', 2500);

        Http::fake(function (Request $request) use ($subject, $leaseClosed, $leaseActive) {
            if (preg_match('#/Property\\([^?]+\\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }
            $filter = (string) ($request->data()['$filter'] ?? '');
            if (str_contains($filter, "PropertyType eq 'Residential Lease'")) {
                if (str_contains($filter, "tolower(StandardStatus) eq 'closed'")) {
                    return Http::response(['value' => [$leaseClosed]], 200);
                }

                return Http::response(['value' => [$leaseActive]], 200);
            }

            return Http::response(['value' => []], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'stellar:12345'],
            'mode' => 'rent_hold_cashflow',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
            'rental_params' => [
                'purchase_price' => 450000,
                'down_payment_percent' => 20,
                'interest_rate' => 7,
                'loan_term_years' => 30,
                'annual_tax' => 4800,
                'annual_insurance' => 2200,
            ],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $this->assertIsArray($response->json('rental_result'));
        $this->assertSame(1, count($response->json('rental_comps.closed')));
        $this->assertSame(1, count($response->json('rental_comps.active')));
    }

    public function test_comps_flip_vs_hold_returns_result_payload(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subject = $this->subjectProperty();
        $sold = $this->closedComp('stellar:90001');
        $leaseClosed = $this->leaseComp('stellar:lease1', 'Closed', 2400);
        $leaseActive = $this->leaseComp('stellar:lease2', 'Active', 2500);

        Http::fake(function (Request $request) use ($subject, $sold, $leaseClosed, $leaseActive) {
            if (preg_match('#/Property\\([^?]+\\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }
            $filter = (string) ($request->data()['$filter'] ?? '');
            if (str_contains($filter, "PropertyType eq 'Residential Lease'")) {
                if (str_contains($filter, "tolower(StandardStatus) eq 'closed'")) {
                    return Http::response(['value' => [$leaseClosed]], 200);
                }

                return Http::response(['value' => [$leaseActive]], 200);
            }

            return Http::response(['value' => [$sold]], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'stellar:12345'],
            'mode' => 'flip_vs_hold',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
            'rental_params' => ['purchase_price' => 450000],
            'flip_params' => ['purchase_price' => 450000, 'rehab_budget' => 30000, 'holding_months' => 6],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $this->assertIsArray($response->json('flip_vs_hold_result'));
        $this->assertNotNull($response->json('flip_vs_hold_result.recommendation'));
    }

    public function test_comps_appraiser_simulation_returns_result_payload(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subject = $this->subjectProperty();
        $compA = $this->closedComp('stellar:90001');
        $compB = $this->closedComp('stellar:90002', 0.002);
        $compC = $this->closedComp('stellar:90003', 0.004);

        Http::fake(function (Request $request) use ($subject, $compA, $compB, $compC) {
            if (preg_match('#/Property\\([^?]+\\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }

            return Http::response(['value' => [$compA, $compB, $compC]], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'stellar:12345'],
            'mode' => 'appraiser_simulation',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
            'simulation_params' => ['supporting_comp_count' => 3],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $this->assertIsArray($response->json('simulation_result'));
        $this->assertNotNull($response->json('simulation_result.risk_band'));
    }
}
