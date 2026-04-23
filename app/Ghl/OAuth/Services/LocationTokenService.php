<?php

namespace App\Ghl\OAuth\Services;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Services\GhlAuditService;
use Illuminate\Support\Facades\Http;

class LocationTokenService
{
    public function __construct(
        private readonly OAuthService $oauthService,
        private readonly GhlAuditService $audit,
    ) {}

    /**
     * Exchange agency token for a location-scoped token row (hybrid revenue path).
     *
     * @return array<string, mixed>
     */
    public function fetchLocationToken(GhlOAuthToken $agencyToken, string $locationId): array
    {
        $plain = $agencyToken->access_token;

        $response = Http::withToken($plain)
            ->withHeaders([
                'Version' => config('ghl.api.version'),
                'Accept' => 'application/json',
            ])
            ->asJson()
            ->timeout((int) config('ghl.api.timeout'))
            ->post(config('ghl.oauth.location_token_url'), [
                'companyId' => $agencyToken->ghl_company_id,
                'locationId' => $locationId,
            ]);

        if ($response->failed()) {
            $this->audit->forToken($agencyToken, '/oauth/locationToken', 'POST', [
                'response_status' => $response->status(),
                'error_details' => $response->body(),
            ]);
            throw new \RuntimeException('locationToken failed: '.$response->body());
        }

        return $response->json();
    }

    public function persistLocationTokenFromResponse(array $payload, GhlOAuthToken $parent): GhlOAuthToken
    {
        $userType = $payload['userType'] ?? 'Location';

        return $this->oauthService->persistTokenFromResponse($payload, $userType);
    }
}
