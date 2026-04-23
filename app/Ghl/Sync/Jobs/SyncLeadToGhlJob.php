<?php

namespace App\Ghl\Sync\Jobs;

use App\Ghl\Sync\Models\QuantyraLead;
use App\Ghl\Sync\Services\LeadSyncService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

/**
 * Revenue Impact: Async CRM sync keeps widget submissions fast → higher completion rates → more paid seats.
 */
class SyncLeadToGhlJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public function __construct(
        public int $quantyraLeadId,
    ) {
        $this->onQueue(config('ghl.sync.queues.sync'));
    }

    public function handle(LeadSyncService $sync): void
    {
        $lead = QuantyraLead::query()->find($this->quantyraLeadId);
        if (! $lead) {
            return;
        }

        $sync->sync($lead);
    }
}
