<?php

namespace Tests\Feature\Ghl;

use App\Ghl\OAuth\Support\OAuthStateToken;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class GhlOAuthCallbackTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();
        config([
            'ghl.oauth.client_id' => 'test-client',
            'ghl.oauth.client_secret' => 'test-secret',
            'ghl.oauth.redirect_uri' => 'https://idx-api.quantyralabs.cc/oauth/leadconnector/callback',
            'ghl.oauth.token_url' => 'https://services.leadconnectorhq.com/oauth/token',
        ]);
    }

    public function test_oauth_callback_accepts_encrypted_state_without_prior_session(): void
    {
        Http::fake([
            'https://services.leadconnectorhq.com/oauth/token' => Http::response([
                'access_token' => 'access-test',
                'refresh_token' => 'refresh-test',
                'expires_in' => 3600,
                'userId' => 'u-callback-1',
                'companyId' => 'co-callback-1',
                'locationId' => 'loc-callback-1',
                'userType' => 'Location',
            ], 200),
        ]);

        $state = OAuthStateToken::encode('Location');

        $this->get('/oauth/leadconnector/callback?code=auth-code-test&state='.rawurlencode($state))
            ->assertRedirect(route('leadconnector.register-urls'));

        Http::assertSent(function (Request $request): bool {
            if ($request->url() !== 'https://services.leadconnectorhq.com/oauth/token') {
                return false;
            }
            $ct = $request->header('Content-Type')[0] ?? '';

            return str_contains($ct, 'application/x-www-form-urlencoded');
        });

        $this->assertDatabaseHas('ghl_oauth_tokens', [
            'ghl_company_id' => 'co-callback-1',
            'ghl_location_id' => 'loc-callback-1',
        ]);
    }

    public function test_oauth_callback_still_accepts_plain_state_when_session_matches(): void
    {
        Http::fake([
            'https://services.leadconnectorhq.com/oauth/token' => Http::response([
                'access_token' => 'access-test-2',
                'refresh_token' => 'refresh-test-2',
                'expires_in' => 3600,
                'userId' => 'u-callback-2',
                'companyId' => 'co-callback-2',
                'locationId' => 'loc-callback-2',
                'userType' => 'Location',
            ], 200),
        ]);

        $plain = 'plain_state_'.str_repeat('a', 32);

        $this->withSession([
            'ghl_oauth_state' => $plain,
            'ghl_oauth_user_type' => 'Company',
        ])->get('/oauth/leadconnector/callback?code=auth-code-plain&state='.$plain)
            ->assertRedirect(route('leadconnector.register-urls'));

        $this->assertDatabaseHas('ghl_oauth_tokens', [
            'ghl_company_id' => 'co-callback-2',
            'ghl_location_id' => 'loc-callback-2',
        ]);
    }

    public function test_oauth_callback_rejects_bad_state(): void
    {
        Http::fake();

        $this->get('/oauth/leadconnector/callback?code=x&state=not-valid')
            ->assertForbidden();
    }
}
