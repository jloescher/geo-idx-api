<?php

namespace App\Ghl\Webhooks\Handlers;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Services\GhlAuditService;
use App\Ghl\Sync\Models\GhlInstalledLocation;
use App\Ghl\Webhooks\Models\GhlWebhookEvent;

/**
 * Revenue Impact: INSTALL webhook confirms CRM attachment → faster token reconciliation for bulk agency rollout.
 */
class InstallHandler
{
    public function __construct(
        private readonly GhlAuditService $audit,
    ) {}

    public function handle(array $payload, GhlWebhookEvent $event): void
    {
        $companyId = (string) ($payload['companyId'] ?? '');
        $locationId = isset($payload['locationId']) ? (string) $payload['locationId'] : null;

        if ($locationId) {
            $token = GhlOAuthToken::query()
                ->where('ghl_company_id', $companyId)
                ->where('ghl_location_id', $locationId)
                ->orderByDesc('id')
                ->first();

            GhlInstalledLocation::query()->updateOrCreate(
                [
                    'ghl_company_id' => $companyId,
                    'ghl_location_id' => $locationId,
                ],
                [
                    'ghl_oauth_token_id' => $token?->id,
                    'location_name' => (string) ($payload['companyName'] ?? ''),
                    'is_whitelabel' => (bool) ($payload['isWhitelabelCompany'] ?? false),
                    'whitelabel_domain' => data_get($payload, 'whitelabelDetails.domain'),
                    'whitelabel_logo_url' => data_get($payload, 'whitelabelDetails.logoUrl'),
                    'status' => 'active',
                ],
            );
        }

        $this->audit->record([
            'ghl_company_id' => $companyId,
            'ghl_location_id' => $locationId,
            'api_endpoint' => '/webhooks/leadconnector',
            'request_method' => 'POST',
            'sync_status' => 'webhook_install',
            'response_status' => 200,
        ]);
    }
}
