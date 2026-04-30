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
            'geocoding.google_api_key' => 'test-google-geocoding-key',
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

    public function test_comps_bpo_requires_idx_full_access(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:access']);

        Http::fake();

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'off_market', 'lat' => 27.95, 'lng' => -82.45],
            'mode' => 'bpo',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
            'bpo_params' => ['sold_months_back' => 6, 'max_comps' => 3],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', false);
        $response->assertJsonPath('error', 'Investor modes require idx:full access.');
    }

    public function test_comps_bpo_returns_full_urar_grid_with_rate_sources(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subject = $this->subjectProperty();

        $comps = [];
        for ($i = 0; $i < 8; $i++) {
            $comp = $this->closedComp('stellar:90'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 1600 + ($i * 100);
            $comp['ClosePrice'] = 320000 + ($i * 25000);
            $comp['PoolPrivateYN'] = $i % 2 === 0;
            $comp['GarageSpaces'] = 1 + ($i % 3);
            $comps[] = $comp;
        }

        Http::fake(function (Request $request) use ($subject, $comps) {
            if (preg_match('#/Property\([^?]+\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }

            return Http::response(['value' => $comps], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'stellar:12345'],
            'mode' => 'bpo',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);

        $this->assertNotNull($response->json('bpo_result'));
        $this->assertNotNull($response->json('bpo_result.point_estimate'));
        $this->assertNotNull($response->json('bpo_result.range.low'));
        $this->assertNotNull($response->json('bpo_result.range.high'));
        $this->assertNotNull($response->json('bpo_result.confidence'));
        $this->assertNotNull($response->json('bpo_result.confidence_band'));
        $this->assertIsString($response->json('bpo_result.reconciliation_summary'));

        $marketRates = $response->json('bpo_result.market_rates');
        $this->assertIsArray($marketRates);
        $this->assertArrayHasKey('gla_per_sf', $marketRates);
        $this->assertArrayHasKey('method', $marketRates);
        $this->assertContains($marketRates['method'], ['ols', 'paired', 'median_only']);
        $this->assertGreaterThan(0, (float) $marketRates['gla_per_sf']);

        $grid = $response->json('bpo_result.adjustment_grid');
        $this->assertIsArray($grid);
        $this->assertNotEmpty($grid);
        $this->assertArrayHasKey('lines', $grid[0]);
        $this->assertArrayHasKey('net_adjustment', $grid[0]);
        $this->assertArrayHasKey('adjusted_price', $grid[0]);

        $hasGlaLine = false;
        foreach ($grid as $entry) {
            foreach ($entry['lines'] ?? [] as $line) {
                if (($line['feature'] ?? '') === 'gla') {
                    $hasGlaLine = true;
                    $this->assertArrayHasKey('rate_source', $line);
                    $this->assertArrayHasKey('rate_per_unit', $line);
                    $this->assertArrayHasKey('adjustment', $line);
                    break 2;
                }
            }
        }
        $this->assertTrue($hasGlaLine, 'At least one comp should have a GLA adjustment line');
    }

    public function test_comps_bpo_derives_pool_value_from_market_not_static(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subject = $this->subjectProperty();

        $comps = [];
        for ($i = 0; $i < 8; $i++) {
            $comp = $this->closedComp('stellar:91'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.002);
            $comp['LivingArea'] = 1800;
            $comp['BedroomsTotal'] = 3;
            $comp['BathroomsTotalDecimal'] = 2;
            $comp['YearBuilt'] = 2008;
            $comp['LotSizeAcres'] = 0.22;
            $comp['GarageSpaces'] = 2;
            $hasPool = $i < 4;
            $comp['PoolPrivateYN'] = $hasPool;
            $comp['WaterfrontYN'] = false;
            $comp['ClosePrice'] = $hasPool ? 420000 : 395000;
            $comps[] = $comp;
        }

        Http::fake(function (Request $request) use ($subject, $comps) {
            if (preg_match('#/Property\([^?]+\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }

            return Http::response(['value' => $comps], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'stellar:12345'],
            'mode' => 'bpo',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);

        $poolValue = (float) $response->json('bpo_result.market_rates.pool_value');
        $this->assertGreaterThan(0, $poolValue, 'Pool value should be derived from market data');

        $method = $response->json('bpo_result.market_rates.method');
        $this->assertContains($method, ['ols', 'paired'], 'Market rates should be derived from comps');
    }

    public function test_comps_bpo_falls_back_when_insufficient_comps(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subject = $this->subjectProperty();
        $compA = $this->closedComp('stellar:90001', 0.005);

        Http::fake(function (Request $request) use ($subject, $compA) {
            if (preg_match('#/Property\([^?]+\)#', $request->url()) === 1) {
                return Http::response($subject, 200);
            }

            return Http::response(['value' => [$compA]], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => ['type' => 'mls', 'listing_id' => 'stellar:12345'],
            'mode' => 'bpo',
            'scope' => ['type' => 'radius', 'radius_miles' => 10],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $response->assertJsonPath('bpo_result.market_rates.method', 'median_only');
        $warnings = $response->json('warnings');
        $this->assertNotEmpty($warnings);
    }

    public function test_comps_home_value_requires_idx_full_access(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:access']);

        Http::fake();

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'type' => 'off_market',
                'lat' => 27.95,
                'lng' => -82.45,
                'address' => '100 Main St, Tampa, FL 33602',
                'condition' => 'good',
                'property_type' => 'sfr',
                'half_bathrooms' => 0,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', false);
        $response->assertJsonPath('error', 'Investor modes require idx:full access.');
    }

    public function test_comps_home_value_returns_error_for_unresolvable_address(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        // Geocoding returns no results
        Http::fake([
            'maps.googleapis.com/*' => Http::response([
                'status' => 'ZERO_RESULTS',
                'results' => [],
            ], 200),
        ]);

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'type' => 'off_market',
                'lat' => 0,
                'lng' => 0,
                'address' => 'Invalid Address That Cannot Be Geocoded XYZ123',
                'condition' => 'good',
                'property_type' => 'sfr',
                'bedrooms' => 3,
                'full_bathrooms' => 2,
                'half_bathrooms' => 0,
                'living_area_sqft' => 1800,
                'year_built' => 2000,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', false);
        $response->assertJsonPath('error', 'Subject listing not found');
    }

    public function test_comps_home_value_geocodes_address_and_returns_estimate(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $comps = [];
        for ($i = 0; $i < 6; $i++) {
            $comp = $this->closedComp('stellar:hv'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 1600 + ($i * 100);
            $comp['ClosePrice'] = 320000 + ($i * 25000);
            $comp['PoolPrivateYN'] = $i % 2 === 0;
            $comp['GarageSpaces'] = 2;
            $comps[] = $comp;
        }

        Http::fake([
            'maps.googleapis.com/*' => Http::response([
                'status' => 'OK',
                'results' => [[
                    'geometry' => ['location' => ['lat' => 27.95, 'lng' => -82.45]],
                    'formatted_address' => '100 Main St, Tampa, FL 33602, USA',
                    'place_id' => 'test_place_id',
                ]],
            ], 200),
            'bridge.test/*' => Http::response(['value' => $comps], 200),
        ]);

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'type' => 'off_market',
                'lat' => 0,
                'lng' => 0,
                'address' => '100 Main St, Tampa, FL 33602',
                'condition' => 'good',
                'property_type' => 'sfr',
                'bedrooms' => 3,
                'full_bathrooms' => 2,
                'half_bathrooms' => 0,
                'living_area_sqft' => 1800,
                'year_built' => 2010,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
            'home_value_params' => ['sold_months_back' => 12, 'max_comps' => 6],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $this->assertNotNull($response->json('home_value_result'));
        $this->assertNotNull($response->json('home_value_result.point_estimate'));
        $this->assertNotNull($response->json('home_value_result.range.low'));
        $this->assertNotNull($response->json('home_value_result.range.high'));
        $this->assertNotNull($response->json('home_value_result.confidence'));
        $this->assertSame('good', $response->json('home_value_result.condition'));
    }

    public function test_comps_home_value_condition_poor_reduces_estimate(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $comps = [];
        for ($i = 0; $i < 8; $i++) {
            $comp = $this->closedComp('stellar:cp'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 1700 + ($i * 120);
            $comp['ClosePrice'] = 340000 + ($i * 20000);
            $comp['YearBuilt'] = 1998 + ($i * 3);
            $comp['LotSizeAcres'] = 0.18 + ($i * 0.02);
            $comp['GarageSpaces'] = 1 + ($i % 3);
            $comp['PoolPrivateYN'] = $i % 2 === 0;
            $comps[] = $comp;
        }

        Http::fake(function ($request) use ($comps) {
            $url = $request->url();
            if (str_contains($url, 'maps.googleapis.com')) {
                return Http::response([
                    'status' => 'OK',
                    'results' => [[
                        'geometry' => ['location' => ['lat' => 27.95, 'lng' => -82.45]],
                        'formatted_address' => '100 Main St, Tampa, FL 33602, USA',
                        'place_id' => 'test_place_id',
                    ]],
                ], 200);
            }

            return Http::response(['value' => $comps], 200);
        });

        // Get "poor" condition estimate
        $responsePoor = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'type' => 'off_market',
                'lat' => 0,
                'lng' => 0,
                'address' => '100 Main St, Tampa, FL 33602',
                'condition' => 'poor',
                'property_type' => 'sfr',
                'bedrooms' => 3,
                'full_bathrooms' => 2,
                'half_bathrooms' => 0,
                'living_area_sqft' => 1800,
                'year_built' => 2010,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        // Get "excellent" condition estimate
        $responseExcellent = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'type' => 'off_market',
                'lat' => 0,
                'lng' => 0,
                'address' => '100 Main St, Tampa, FL 33602',
                'condition' => 'excellent',
                'property_type' => 'sfr',
                'bedrooms' => 3,
                'full_bathrooms' => 2,
                'half_bathrooms' => 0,
                'living_area_sqft' => 1800,
                'year_built' => 2010,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $this->assertLessThan(
            (float) $responseExcellent->json('home_value_result.point_estimate'),
            (float) $responsePoor->json('home_value_result.point_estimate'),
            'Poor condition should produce a lower estimate than excellent condition',
        );
    }

    public function test_comps_home_value_renovation_credits_applied(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $comps = [];
        for ($i = 0; $i < 8; $i++) {
            $comp = $this->closedComp('stellar:rc'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 1700 + ($i * 100);
            $comp['ClosePrice'] = 340000 + ($i * 20000);
            $comp['YearBuilt'] = 1998 + ($i * 2);
            $comp['GarageSpaces'] = 2;
            $comps[] = $comp;
        }

        Http::fake([
            'maps.googleapis.com/*' => Http::response([
                'status' => 'OK',
                'results' => [[
                    'geometry' => ['location' => ['lat' => 27.95, 'lng' => -82.45]],
                    'formatted_address' => '100 Main St, Tampa, FL 33602, USA',
                    'place_id' => 'test_place_id',
                ]],
            ], 200),
            'bridge.test/*' => Http::response(['value' => $comps], 200),
        ]);

        $renovationYear = (int) date('Y') - 2;

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'type' => 'off_market',
                'lat' => 0,
                'lng' => 0,
                'address' => '100 Main St, Tampa, FL 33602',
                'condition' => 'good',
                'property_type' => 'sfr',
                'bedrooms' => 3,
                'full_bathrooms' => 2,
                'half_bathrooms' => 0,
                'living_area_sqft' => 1800,
                'year_built' => 2000,
                'renovated_kitchen_year' => $renovationYear,
                'renovated_bathrooms_year' => $renovationYear,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);

        $grid = $response->json('home_value_result.adjustment_grid');
        $this->assertNotEmpty($grid);

        $hasKitchenCredit = false;
        $hasBathCredit = false;
        foreach ($grid as $entry) {
            foreach ($entry['lines'] ?? [] as $line) {
                if (($line['feature'] ?? '') === 'renovation_kitchen') {
                    $hasKitchenCredit = true;
                    $this->assertGreaterThan(0, $line['adjustment'], 'Kitchen credit should be positive');
                    $this->assertContains($line['rate_source'], ['market_derived', 'default']);
                }
                if (($line['feature'] ?? '') === 'renovation_bathrooms') {
                    $hasBathCredit = true;
                    $this->assertGreaterThan(0, $line['adjustment'], 'Bathroom credit should be positive');
                    $this->assertContains($line['rate_source'], ['market_derived', 'default']);
                }
            }
        }
        $this->assertTrue($hasKitchenCredit, 'Kitchen renovation credit should be present');
        $this->assertTrue($hasBathCredit, 'Bathroom renovation credit should be present');
    }

    public function test_comps_home_value_renovation_credits_scale_with_market(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        // High-value comps: ~$500/sf market → credits should exceed defaults
        $comps = [];
        for ($i = 0; $i < 8; $i++) {
            $comp = $this->closedComp('stellar:hc'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 2500 + ($i * 100);
            $comp['ClosePrice'] = 1250000 + ($i * 50000);
            $comp['YearBuilt'] = 2010 + $i;
            $comp['GarageSpaces'] = 2;
            $comps[] = $comp;
        }

        Http::fake([
            'maps.googleapis.com/*' => Http::response([
                'status' => 'OK',
                'results' => [[
                    'geometry' => ['location' => ['lat' => 27.95, 'lng' => -82.45]],
                    'formatted_address' => '100 Luxury Ln, Tampa, FL 33602, USA',
                    'place_id' => 'test_place_id',
                ]],
            ], 200),
            'bridge.test/*' => Http::response(['value' => $comps], 200),
        ]);

        $renovationYear = (int) date('Y') - 1;

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'type' => 'off_market',
                'lat' => 0,
                'lng' => 0,
                'address' => '100 Luxury Ln, Tampa, FL 33602',
                'condition' => 'excellent',
                'property_type' => 'sfr',
                'bedrooms' => 4,
                'full_bathrooms' => 3,
                'half_bathrooms' => 0,
                'living_area_sqft' => 2800,
                'year_built' => 2015,
                'renovated_kitchen_year' => $renovationYear,
                'renovated_bathrooms_year' => $renovationYear,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);

        $grid = $response->json('home_value_result.adjustment_grid');
        $this->assertNotEmpty($grid);

        $kitchenCredit = 0;
        $bathCredit = 0;
        foreach ($grid as $entry) {
            foreach ($entry['lines'] ?? [] as $line) {
                if (($line['feature'] ?? '') === 'renovation_kitchen') {
                    $kitchenCredit = $line['adjustment'];
                }
                if (($line['feature'] ?? '') === 'renovation_bathrooms') {
                    $bathCredit = $line['adjustment'];
                }
            }
        }

        // In a ~$500/sf market, kitchen credit (2% of ~$1.4M) should be ~$28K, well above $8K default
        $this->assertGreaterThan(8000, $kitchenCredit, 'Kitchen credit should exceed default in high-value market');
        $this->assertGreaterThan(6000, $bathCredit, 'Bathroom credit should exceed default in high-value market');
    }

    public function test_comps_home_value_with_listing_id_auto_populates_subject(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subjectRecord = $this->subjectProperty();
        $subjectRecord['PropertyCondition'] = 'Good';
        $subjectRecord['BathroomsFull'] = 2;
        $subjectRecord['BathroomsHalf'] = 1;
        $subjectRecord['StoriesTotal'] = 1;
        $subjectRecord['SubdivisionName'] = 'Test Heights';

        $comps = [];
        for ($i = 0; $i < 6; $i++) {
            $comp = $this->closedComp('stellar:li'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 1700 + ($i * 100);
            $comp['ClosePrice'] = 340000 + ($i * 20000);
            $comp['GarageSpaces'] = 2;
            $comps[] = $comp;
        }

        Http::fake(function (Request $request) use ($subjectRecord, $comps) {
            if (preg_match('#/Property\([^?]+\)#', $request->url()) === 1) {
                return Http::response($subjectRecord, 200);
            }

            return Http::response(['value' => $comps], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'listing_id' => '12345',
                'half_bathrooms' => 1,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
            'home_value_params' => ['sold_months_back' => 12, 'max_comps' => 6],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $this->assertNotNull($response->json('home_value_result'));
        $this->assertNotNull($response->json('home_value_result.point_estimate'));
        $response->assertJsonPath('home_value_result.condition', 'good');
        $response->assertJsonPath('subject.property_sub_type', 'Single Family Residence');
        $response->assertJsonPath('subject.listing_id', '12345');
    }

    public function test_comps_home_value_condition_derived_from_public_remarks(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subjectRecord = $this->subjectProperty();
        $subjectRecord['PropertyCondition'] = null;
        $subjectRecord['PublicRemarks'] = 'This fixer upper needs major work but has great bones. Investor special!';
        $subjectRecord['BathroomsFull'] = 2;
        $subjectRecord['BathroomsHalf'] = 0;

        $comps = [];
        for ($i = 0; $i < 6; $i++) {
            $comp = $this->closedComp('stellar:dr'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 1700 + ($i * 100);
            $comp['ClosePrice'] = 340000 + ($i * 20000);
            $comp['GarageSpaces'] = 2;
            $comps[] = $comp;
        }

        Http::fake(function (Request $request) use ($subjectRecord, $comps) {
            if (preg_match('#/Property\([^?]+\)#', $request->url()) === 1) {
                return Http::response($subjectRecord, 200);
            }

            return Http::response(['value' => $comps], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'listing_id' => '12345',
                'half_bathrooms' => 0,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $response->assertJsonPath('home_value_result.condition', 'poor');
    }

    public function test_comps_home_value_condition_null_when_not_derivable(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subjectRecord = $this->subjectProperty();
        $subjectRecord['PropertyCondition'] = null;
        $subjectRecord['PublicRemarks'] = 'Beautiful 3 bedroom home in a great neighborhood.';
        $subjectRecord['BathroomsFull'] = 2;
        $subjectRecord['BathroomsHalf'] = 0;

        $comps = [];
        for ($i = 0; $i < 6; $i++) {
            $comp = $this->closedComp('stellar:nd'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 1700 + ($i * 100);
            $comp['ClosePrice'] = 340000 + ($i * 20000);
            $comp['GarageSpaces'] = 2;
            $comps[] = $comp;
        }

        Http::fake(function (Request $request) use ($subjectRecord, $comps) {
            if (preg_match('#/Property\([^?]+\)#', $request->url()) === 1) {
                return Http::response($subjectRecord, 200);
            }

            return Http::response(['value' => $comps], 200);
        });

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'listing_id' => '12345',
                'half_bathrooms' => 0,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $response->assertJsonPath('home_value_result.condition', null);
    }

    public function test_comps_home_value_condition_override_takes_precedence(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $subjectRecord = $this->subjectProperty();
        $subjectRecord['PropertyCondition'] = 'Excellent';
        $subjectRecord['BathroomsFull'] = 2;
        $subjectRecord['BathroomsHalf'] = 0;

        $comps = [];
        for ($i = 0; $i < 6; $i++) {
            $comp = $this->closedComp('stellar:co'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 1700 + ($i * 100);
            $comp['ClosePrice'] = 340000 + ($i * 20000);
            $comp['GarageSpaces'] = 2;
            $comps[] = $comp;
        }

        Http::fake(function (Request $request) use ($subjectRecord, $comps) {
            if (preg_match('#/Property\([^?]+\)#', $request->url()) === 1) {
                return Http::response($subjectRecord, 200);
            }

            return Http::response(['value' => $comps], 200);
        });

        // User explicitly passes condition=fair, which should override PropertyCondition=Excellent
        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'listing_id' => '12345',
                'half_bathrooms' => 0,
                'condition' => 'fair',
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $response->assertJsonPath('home_value_result.condition', 'fair');
    }

    public function test_comps_home_value_expanded_property_types_accepted(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $this->subscribeMega($user);
        $token = $this->actingAsWithToken($user, 't', ['idx:full']);

        $comps = [];
        for ($i = 0; $i < 6; $i++) {
            $comp = $this->closedComp('stellar:ep'.str_pad((string) $i, 3, '0', STR_PAD_LEFT), $i * 0.003);
            $comp['LivingArea'] = 1700 + ($i * 100);
            $comp['ClosePrice'] = 340000 + ($i * 20000);
            $comp['GarageSpaces'] = 2;
            $comps[] = $comp;
        }

        Http::fake([
            'maps.googleapis.com/*' => Http::response([
                'status' => 'OK',
                'results' => [[
                    'geometry' => ['location' => ['lat' => 27.95, 'lng' => -82.45]],
                    'formatted_address' => '100 Main St, Tampa, FL 33602, USA',
                    'place_id' => 'test_place_id',
                ]],
            ], 200),
            'bridge.test/*' => Http::response(['value' => $comps], 200),
        ]);

        $response = $this->postJson('/api/v1/comps/run', [
            'subject' => [
                'type' => 'off_market',
                'lat' => 0,
                'lng' => 0,
                'address' => '100 Main St, Tampa, FL 33602',
                'property_type' => 'duplex',
                'bedrooms' => 4,
                'full_bathrooms' => 2,
                'half_bathrooms' => 0,
                'living_area_sqft' => 1800,
                'year_built' => 2000,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
        ], [
            'Authorization' => 'Bearer '.$token,
        ]);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $response->assertJsonPath('subject.property_sub_type', 'Duplex');
    }
}
