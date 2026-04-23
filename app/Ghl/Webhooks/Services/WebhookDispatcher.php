<?php

namespace App\Ghl\Webhooks\Services;

use App\Ghl\Webhooks\Handlers\CrmEventAuditHandler;
use App\Ghl\Webhooks\Handlers\InstallHandler;
use App\Ghl\Webhooks\Handlers\UninstallHandler;
use App\Ghl\Webhooks\Models\GhlWebhookEvent;

class WebhookDispatcher
{
    public function __construct(
        private readonly InstallHandler $installHandler,
        private readonly UninstallHandler $uninstallHandler,
        private readonly CrmEventAuditHandler $crmEventAuditHandler,
    ) {}

    public function dispatch(GhlWebhookEvent $event): void
    {
        $payload = $event->payload ?? [];
        $type = strtoupper((string) ($payload['type'] ?? ''));

        match ($type) {
            'INSTALL', 'APPINSTALL' => $this->installHandler->handle($payload, $event),
            'UNINSTALL', 'APPUNINSTALL' => $this->uninstallHandler->handle($payload, $event),
            'CONTACTCREATE', 'CONTACTUPDATE', 'CONTACTDELETE', 'CONTACTTAGUPDATE',
            'OPPORTUNITYCREATE', 'OPPORTUNITYSTATUSUPDATE', 'OPPORTUNITYSTAGEUPDATE',
            'NOTECREATE', 'TASKCREATE' => $this->crmEventAuditHandler->handle($payload, $event),
            default => $this->crmEventAuditHandler->handle($payload, $event),
        };
    }
}
