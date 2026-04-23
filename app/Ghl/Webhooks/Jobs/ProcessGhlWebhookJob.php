<?php

namespace App\Ghl\Webhooks\Jobs;

use App\Ghl\Webhooks\Models\GhlWebhookEvent;
use App\Ghl\Webhooks\Services\WebhookDispatcher;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\DB;

class ProcessGhlWebhookJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public function __construct(
        public int $webhookEventId,
    ) {
        $this->onQueue(config('ghl.sync.queues.webhooks'));
    }

    public function handle(WebhookDispatcher $dispatcher): void
    {
        DB::transaction(function () use ($dispatcher): void {
            $event = GhlWebhookEvent::query()->lockForUpdate()->find($this->webhookEventId);
            if (! $event || $event->processing_status === 'processed') {
                return;
            }

            $event->update(['processing_status' => 'processing']);

            try {
                $dispatcher->dispatch($event);
                $event->update([
                    'processing_status' => 'processed',
                    'processed_at' => now(),
                    'handler_class' => WebhookDispatcher::class,
                ]);
            } catch (\Throwable $e) {
                $event->update([
                    'processing_status' => 'failed',
                    'processing_error' => $e->getMessage(),
                ]);
                throw $e;
            }
        });
    }
}
