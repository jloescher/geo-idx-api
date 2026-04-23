<?php

namespace App\Ghl\Sync\Jobs;

use App\Ghl\Sync\Services\SubscriptionSyncService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

/**
 * Revenue Impact: Stripe → GHL tag sync surfaces paywall state inside CRM → faster upgrade conversations.
 */
class SyncSubscriptionStatusJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public function __construct(
        public string $ghlLocationId,
        public string $stripeStatus,
        public ?string $stripeSubscriptionId = null,
    ) {
        $this->onQueue(config('ghl.sync.queues.sync'));
    }

    public function handle(SubscriptionSyncService $subscription): void
    {
        $subscription->applyStripeStatus($this->ghlLocationId, $this->stripeStatus, $this->stripeSubscriptionId);

        // Future: resolve token + contact and push subscription tags via TagManager.
    }
}
