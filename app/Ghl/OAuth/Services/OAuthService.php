<?php

namespace App\Ghl\OAuth\Services;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Services\GhlAuditService;
use App\Ghl\Sync\Models\GhlInstalledLocation;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Str;

class OAuthService
{
    public function __construct(
        private readonly GhlAuditService $audit,
    ) {}

    public function buildAuthorizationUrl(string $state): string
    {
        $query = http_build_query([
            'response_type' => 'code',
            'client_id' => config('ghl.oauth.client_id'),
            'redirect_uri' => config('ghl.oauth.redirect_uri'),
            'scope' => trim((string) config('ghl.oauth.scopes')),
            'state' => $state,
        ], '', '&', PHP_QUERY_RFC3986);

        return config('ghl.oauth.authorize_url').'?'.$query;
    }

    /**
     * @return array<string, mixed>
     */
    public function exchangeAuthorizationCode(string $code, string $userType): array
    {
        $response = Http::asJson()
            ->acceptJson()
            ->timeout((int) config('ghl.api.timeout'))
            ->post(config('ghl.oauth.token_url'), [
                'client_id' => config('ghl.oauth.client_id'),
                'client_secret' => config('ghl.oauth.client_secret'),
                'grant_type' => 'authorization_code',
                'code' => $code,
                'user_type' => $userType,
                'redirect_uri' => config('ghl.oauth.redirect_uri'),
            ]);

        if ($response->failed()) {
            throw new \RuntimeException('GHL token exchange failed: '.$response->body());
        }

        return $response->json();
    }

    public function persistTokenFromResponse(array $payload, string $userType): GhlOAuthToken
    {
        $access = $payload['access_token'] ?? throw new \InvalidArgumentException('Missing access_token');
        $refresh = $payload['refresh_token'] ?? throw new \InvalidArgumentException('Missing refresh_token');
        $expiresIn = (int) ($payload['expires_in'] ?? 86400);

        $resolvedUserType = $payload['userType'] ?? $userType;
        $companyId = $payload['companyId'] ?? '';
        $locationId = $payload['locationId'] ?? null;
        $userId = $payload['userId'] ?? 'unknown';
        $isBulk = (bool) ($payload['isBulkInstallation'] ?? false);

        $scopeRaw = $payload['scope'] ?? '';
        $scopes = is_string($scopeRaw) ? $scopeRaw : (is_array($scopeRaw) ? implode(' ', $scopeRaw) : json_encode($scopeRaw));

        $token = new GhlOAuthToken([
            'ghl_company_id' => $companyId,
            'ghl_location_id' => $locationId,
            'ghl_user_id' => $userId,
            'user_type' => $resolvedUserType,
            'scopes' => $scopes,
            'is_bulk_install' => $isBulk,
            'installed_at' => now(),
            'expires_at' => now()->addSeconds($expiresIn),
            'refresh_expires_at' => now()->addYear(),
            'status' => 'active',
            'refresh_token_id' => $payload['refreshTokenId'] ?? null,
            'access_token_hash' => hash('sha256', $access),
        ]);

        $token->access_token = $access;
        $token->refresh_token = $refresh;
        $token->save();

        if ($locationId) {
            GhlInstalledLocation::query()->updateOrCreate(
                [
                    'ghl_company_id' => $companyId,
                    'ghl_location_id' => $locationId,
                ],
                [
                    'ghl_oauth_token_id' => $token->id,
                    'status' => 'active',
                ],
            );
        }

        $this->audit->record([
            'ghl_oauth_token_id' => $token->id,
            'ghl_company_id' => $companyId,
            'ghl_location_id' => $locationId,
            'ghl_user_id' => $userId,
            'api_endpoint' => '/oauth/token',
            'request_method' => 'POST',
            'sync_status' => 'success',
            'response_status' => 200,
            'lead_type' => null,
        ]);

        return $token;
    }

    public function randomState(): string
    {
        return Str::random(40);
    }
}
