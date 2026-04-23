<?php

namespace App\Ghl\OAuth\Services;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Services\GhlAuditService;
use Illuminate\Support\Facades\Http;

class TokenRefreshService
{
    public function __construct(
        private readonly GhlAuditService $audit,
    ) {}

    public function refresh(GhlOAuthToken $token): GhlOAuthToken
    {
        $plainRefresh = $token->refresh_token;

        $response = Http::asForm()
            ->acceptJson()
            ->timeout((int) config('ghl.api.timeout'))
            ->post(config('ghl.oauth.token_url'), [
                'client_id' => config('ghl.oauth.client_id'),
                'client_secret' => config('ghl.oauth.client_secret'),
                'grant_type' => 'refresh_token',
                'refresh_token' => $plainRefresh,
                'user_type' => $token->user_type,
                'redirect_uri' => config('ghl.oauth.redirect_uri'),
            ]);

        if ($response->failed()) {
            $this->audit->forToken($token, '/oauth/token', 'POST', [
                'sync_status' => 'failed',
                'response_status' => $response->status(),
                'error_details' => $response->body(),
            ]);
            throw new \RuntimeException('GHL refresh failed: '.$response->body());
        }

        $data = $response->json();
        $access = $data['access_token'] ?? throw new \RuntimeException('Missing access_token on refresh');
        $refresh = $data['refresh_token'] ?? $plainRefresh;
        $expiresIn = (int) ($data['expires_in'] ?? 86400);

        $token->access_token = $access;
        $token->refresh_token = $refresh;
        $token->access_token_hash = hash('sha256', $access);
        $token->expires_at = now()->addSeconds($expiresIn);
        $token->last_refreshed_at = now();
        $token->status = 'active';
        if (isset($data['refreshTokenId'])) {
            $token->refresh_token_id = $data['refreshTokenId'];
        }
        $token->save();

        $this->audit->forToken($token, '/oauth/token', 'POST', [
            'sync_status' => 'success',
            'response_status' => 200,
        ]);

        return $token;
    }
}
