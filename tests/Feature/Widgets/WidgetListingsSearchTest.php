<?php

namespace Tests\Feature\Widgets;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Ghl\Widgets\Models\GhlWidgetConfig;
use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class WidgetListingsSearchTest extends TestCase
{
    use RefreshDatabase;

    private function subscribePro(User $user): void
    {
        $priceId = 'price_pro_monthly';
        config(['billing.plans.pro.stripe_price_monthly' => $priceId]);

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

    private function sampleBridgeProperty(string $listingKey = 'stellar:99999'): array
    {
        return [
            'ListingKey' => $listingKey,
            'StandardStatus' => 'Active',
            'ListPrice' => 425000,
            'BedroomsTotal' => 2,
            'BathroomsTotalDecimal' => 2.0,
            'LivingArea' => 1400,
            'City' => 'Tampa',
            'StateOrProvince' => 'FL',
            'PostalCode' => '33602',
            'StreetNumber' => '9',
            'StreetName' => 'Widget Ave',
            'PropertyType' => 'Residential',
            'OnMarketDate' => now()->subDays(3)->format('Y-m-d'),
            'Coordinates' => ['coordinates' => [-82.45, 27.95]],
        ];
    }

    private function createGhlToken(string $companyId, string $locationId): GhlOAuthToken
    {
        $access = 'access-wls-'.uniqid('', true);
        $refresh = 'refresh-wls-'.uniqid('', true);

        $t = new GhlOAuthToken([
            'ghl_company_id' => $companyId,
            'ghl_location_id' => $locationId,
            'ghl_user_id' => 'u1',
            'user_type' => 'Location',
            'scopes' => 'contacts.write',
            'is_bulk_install' => false,
            'expires_at' => now()->addDay(),
            'status' => 'active',
            'access_token_hash' => hash('sha256', $access),
        ]);
        $t->access_token = $access;
        $t->refresh_token = $refresh;
        $t->save();

        return $t->fresh();
    }

    public function test_direct_site_widget_can_search_listings_with_approved_domain_origin(): void
    {
        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty()]);

        /** @var User $user */
        $user = User::factory()->createOne(['mls_membership_status' => 'active']);
        $this->subscribePro($user);
        $key = $user->ensureWidgetEmbedSiteKey();

        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'embed-client.test',
            'is_active' => true,
            'verification_status' => 'verified',
        ]);

        $this->withHeaders([
            'Origin' => 'https://embed-client.test',
        ])->postJson('http://localhost/widget/api/listings-search?api_key='.urlencode($key), [
            'city' => 'Tampa',
            'active_only' => true,
            'page.limit' => 12,
            'page.skip' => 0,
        ])->assertOk()
            ->assertJsonPath('total_count', 1)
            ->assertJsonPath('results.0.city', 'Tampa');
    }

    public function test_ghl_widget_search_requires_matching_approved_domain_host(): void
    {
        $this->fakeBridgePropertyResponse([$this->sampleBridgeProperty('stellar:888')]);

        Domain::query()->create([
            'domain_slug' => 'customer-site.example',
            'is_active' => true,
        ]);

        $oauth = $this->createGhlToken('co-wls', 'loc-wls');
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $oauth->id,
            'ghl_location_id' => 'loc-wls',
            'ghl_company_id' => 'co-wls',
            'primary_url' => 'https://customer-site.example',
            'widget_api_key' => 'qh_wlskey1234567890123456789012ef',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);

        GhlWidgetConfig::query()->create([
            'ghl_location_id' => 'loc-wls',
            'ghl_registered_url_id' => $row->id,
        ]);

        $this->withHeaders([
            'Origin' => 'https://customer-site.example',
        ])->postJson('http://localhost/widget/api/listings-search?api_key='.urlencode($row->widget_api_key), [
            'city' => 'Tampa',
            'active_only' => true,
            'page.limit' => 8,
            'page.skip' => 0,
        ])->assertOk()
            ->assertJsonPath('total_count', 1);
    }
}
