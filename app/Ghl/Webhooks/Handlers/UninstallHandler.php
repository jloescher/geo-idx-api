<?php

namespace App\Ghl\Webhooks\Handlers;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Services\GhlAuditService;
use App\Ghl\Sync\Models\GhlInstalledLocation;
use App\Ghl\Webhooks\Models\GhlWebhookEvent;

/**
 * Revenue Impact: Clean uninstall stops stale sync → avoids spamming dead CRMs → brand safety.
 */
class UninstallHandler
{
    public function __construct(
        private readonly GhlAuditService $audit,
    ) {}

    public function handle(array $payload, GhlWebhookEvent $event): void
    {
        $locationId = isset($payload['locationId']) ? (string) $payload['locationId'] : null;
        $companyId = (string) ($payload['companyId'] ?? '');

        if ($locationId) {
            GhlInstalledLocation::query()
                ->where('ghl_location_id', $locationId)
                ->update([
                    'status' => 'uninstalled',
                    'uninstalled_at' => now(),
                ]);

            GhlOAuthToken::query()
                ->where('ghl_location_id', $locationId)
                ->update([
                    'status' => 'revoked',
                    'revoked_at' => now(),
                    'revoke_reason' => 'GHL_UNINSTALL_WEBHOOK',
                ]);
        } elseif ($companyId !== '') {
            GhlOAuthToken::query()
                ->where('ghl_company_id', $companyId)
                ->whereNull('ghl_location_id')
                ->update([
                    'status' => 'revoked',
                    'revoked_at' => now(),
                    'revoke_reason' => 'GHL_UNINSTALL_WEBHOOK_AGENCY',
                ]);
        }

        $this->audit->record([
            'ghl_company_id' => $companyId,
            'ghl_location_id' => $locationId,
            'api_endpoint' => '/webhooks/leadconnector',
            'request_method' => 'POST',
            'sync_status' => 'webhook_uninstall',
            'response_status' => 200,
        ]);
    }
}
