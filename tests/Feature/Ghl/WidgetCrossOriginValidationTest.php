<?php

namespace Tests\Feature\Ghl;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class WidgetCrossOriginValidationTest extends TestCase
{
    use RefreshDatabase;

    public function test_widget_validate_options_preflight_includes_cors_headers(): void
    {
        $this->options('/api/widgets/validate', [], [
            'Origin' => 'https://localhost',
        ])
            ->assertNoContent()
            ->assertHeader('access-control-allow-origin', 'https://localhost')
            ->assertHeader('access-control-allow-methods', 'POST, OPTIONS');
    }

    public function test_widget_validate_post_accepts_trusted_platform_origin_for_dashboard_preview(): void
    {
        $token = $this->createGhlToken();
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $token->id,
            'ghl_location_id' => 'loc-preview',
            'ghl_company_id' => 'co-preview',
            'primary_url' => 'https://customer-embed.example',
            'widget_api_key' => 'qh_previewkey1234567890123456789012ab',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);

        $this->postJson('http://localhost/api/widgets/validate', [
            'token' => $row->widget_api_key,
            'hostname' => 'localhost',
            'referrer' => null,
            'requireFooter' => true,
        ], [
            'Origin' => 'https://localhost',
        ])->assertOk()
            ->assertHeader('access-control-allow-origin', 'https://localhost')
            ->assertJsonPath('ok', true);
    }

    public function test_widget_config_allows_trusted_platform_origin(): void
    {
        $token = $this->createGhlToken('co-cfg', 'loc-cfg');
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $token->id,
            'ghl_location_id' => 'loc-cfg',
            'ghl_company_id' => 'co-cfg',
            'primary_url' => 'https://customer-cfg.example',
            'widget_api_key' => 'qh_cfgkey1234567890123456789012cd',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);

        $this->withHeaders([
            'Origin' => 'https://localhost',
        ])->get('http://localhost/widget/config/'.$row->widget_api_key)
            ->assertOk()
            ->assertHeader('access-control-allow-origin', 'https://localhost')
            ->assertJsonPath('location_id', 'loc-cfg');
    }

    private function createGhlToken(string $companyId = 'co-preview', string $locationId = 'loc-preview'): GhlOAuthToken
    {
        $access = 'access-preview-'.uniqid('', true);
        $refresh = 'refresh-preview-'.uniqid('', true);

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
