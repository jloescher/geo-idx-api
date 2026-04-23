<?php

namespace Tests\Feature\Ghl;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class GhlMarketplaceFlowTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();
        config([
            'ghl.webhooks.require_signature' => false,
            'ghl.oauth.client_id' => 'test-client',
            'ghl.oauth.client_secret' => 'test-secret',
            'ghl.oauth.redirect_uri' => 'https://idx-api.quantyralabs.cc/oauth/leadconnector/callback',
            'ghl.urls.api_public' => 'https://idx-api.quantyralabs.cc',
            'ghl.urls.idx_platform' => 'https://idx.quantyralabs.cc',
        ]);
    }

    public function test_install_landing_is_ok(): void
    {
        $this->get('/leadconnector/install')->assertOk();
    }

    public function test_oauth_authorize_redirects_to_marketplace(): void
    {
        $response = $this->get('/oauth/leadconnector/authorize');
        $response->assertRedirect();
        $this->assertStringContainsString('marketplace.gohighlevel.com', $response->headers->get('Location'));
        $this->assertStringContainsString('test-client', $response->headers->get('Location'));
    }

    public function test_webhook_accepts_install_payload(): void
    {
        $this->postJson('/webhooks/leadconnector', [
            'type' => 'INSTALL',
            'webhookId' => 'wh_'.uniqid(),
            'appId' => 'app1',
            'companyId' => 'co1',
            'locationId' => 'loc1',
            'userId' => 'u1',
        ])->assertStatus(202);

        $this->assertDatabaseHas('ghl_webhook_events', [
            'event_type' => 'INSTALL',
        ]);
    }

    public function test_widget_origin_rejected_when_not_registered(): void
    {
        $token = $this->createToken('co-x', 'loc-x');
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $token->id,
            'ghl_location_id' => 'loc-x',
            'ghl_company_id' => 'co-x',
            'primary_url' => 'https://allowed.example',
            'widget_api_key' => 'qh_testkey1234567890123456789012ab',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
        ]);

        $this->get('/widget/search/'.$row->widget_api_key, [
            'Origin' => 'https://evil.example',
        ])->assertForbidden();
    }

    public function test_widget_search_allows_registered_origin(): void
    {
        $token = $this->createToken('co-y', 'loc-y');
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $token->id,
            'ghl_location_id' => 'loc-y',
            'ghl_company_id' => 'co-y',
            'primary_url' => 'https://allowed.example',
            'widget_api_key' => 'qh_testkey2234567890123456789012cd',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
        ]);

        $this->get('/widget/search/'.$row->widget_api_key, [
            'Origin' => 'https://allowed.example',
        ])->assertOk();
    }

    private function createToken(string $companyId, string $locationId): GhlOAuthToken
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
