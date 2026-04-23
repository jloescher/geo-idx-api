<?php

namespace App\Ghl\Webhooks\Handlers;

use App\Ghl\Services\GhlAuditService;
use App\Ghl\Webhooks\Models\GhlWebhookEvent;

/**
 * Revenue Impact: Full-funnel webhook audit trail → pipeline analytics → upsell automation later.
 */
class CrmEventAuditHandler
{
    public function __construct(
        private readonly GhlAuditService $audit,
    ) {}

    public function handle(array $payload, GhlWebhookEvent $event): void
    {
        $this->audit->record([
            'ghl_company_id' => $payload['companyId'] ?? null,
            'ghl_location_id' => $payload['locationId'] ?? null,
            'ghl_user_id' => $payload['userId'] ?? null,
            'api_endpoint' => '/webhooks/leadconnector#'.strtolower((string) $event->event_type),
            'request_method' => 'POST',
            'sync_status' => 'webhook_'.strtolower((string) $event->event_type),
            'response_status' => 200,
            'lead_type' => null,
        ]);
    }
}
