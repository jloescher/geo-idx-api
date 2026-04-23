<?php

namespace App\Ghl\Webhooks\Controllers;

use App\Ghl\Webhooks\Jobs\ProcessGhlWebhookJob;
use App\Ghl\Webhooks\Models\GhlWebhookEvent;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Revenue Impact: Webhooks keep install state truthful → accurate billing + sync → revenue integrity.
 */
class WebhookController
{
    public function __invoke(Request $request): JsonResponse
    {
        $payload = $request->all();
        $type = (string) ($payload['type'] ?? 'unknown');
        $webhookId = (string) ($payload['webhookId'] ?? hash('sha256', $request->getContent()));

        $event = GhlWebhookEvent::query()->firstOrCreate(
            ['webhook_id' => $webhookId],
            [
                'event_type' => strtoupper($type),
                'ghl_app_id' => isset($payload['appId']) ? (string) $payload['appId'] : null,
                'ghl_company_id' => isset($payload['companyId']) ? (string) $payload['companyId'] : null,
                'ghl_location_id' => isset($payload['locationId']) ? (string) $payload['locationId'] : null,
                'ghl_user_id' => isset($payload['userId']) ? (string) $payload['userId'] : null,
                'payload' => $payload,
                'processing_status' => 'received',
            ]
        );

        if ($event->processing_status === 'processed') {
            return response()->json(['ok' => true, 'duplicate' => true]);
        }

        ProcessGhlWebhookJob::dispatch($event->id);

        return response()->json(['accepted' => true], 202);
    }
}
