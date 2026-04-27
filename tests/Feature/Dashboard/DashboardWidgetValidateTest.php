<?php

namespace Tests\Feature\Dashboard;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Foundation\Testing\WithoutMiddleware;
use Tests\TestCase;

class DashboardWidgetValidateTest extends TestCase
{
    use RefreshDatabase;
    use WithoutMiddleware;

    public function test_guest_is_rejected_from_dashboard_widget_validate(): void
    {
        $this->post('http://localhost/dashboard/widget-validate', [
            'token' => 'qh_test',
            'hostname' => 'localhost',
        ])->assertStatus(401);
    }

    public function test_authenticated_user_can_validate_widget_for_registered_host(): void
    {
        config(['billing.plans.pro.stripe_price_monthly' => 'price_pro_monthly']);

        $oauth = $this->createGhlOAuthToken('co-dw', 'loc-dw');
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $oauth->id,
            'ghl_location_id' => 'loc-dw',
            'ghl_company_id' => 'co-dw',
            'primary_url' => 'https://customer-site.example',
            'widget_api_key' => 'qh_testkey5234567890123456789012ij',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $this->actingAs($user);

        $this->postJson('http://localhost/dashboard/widget-validate', [
            'token' => $row->widget_api_key,
            'hostname' => 'customer-site.example',
            'requireFooter' => true,
        ])->assertOk()
            ->assertJsonPath('ok', true)
            ->assertJsonPath('requiresFooter', true);
    }

    private function createGhlOAuthToken(string $companyId, string $locationId): GhlOAuthToken
    {
        $access = 'access-'.$companyId.'-'.$locationId.'-'.uniqid('', true);
        $refresh = 'refresh-'.$companyId.'-'.$locationId.'-'.uniqid('', true);

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
}
