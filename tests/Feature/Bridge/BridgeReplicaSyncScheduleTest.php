<?php

namespace Tests\Feature\Bridge;

use App\Jobs\BridgeSyncJob;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Queue;
use Tests\TestCase;

class BridgeReplicaSyncScheduleTest extends TestCase
{
    public function test_schedule_run_dispatches_bridge_sync_kickoff_on_fetch_queue(): void
    {
        config(['bridge.sync_fetch_queue' => 'bridge-sync-fetch']);
        Queue::fake();

        Carbon::setTestNow(Carbon::create(2026, 5, 15, 12, 15, 0, 'UTC'));

        $this->artisan('schedule:run')->assertSuccessful();

        Queue::assertPushed(BridgeSyncJob::class);
        Queue::assertPushedOn('bridge-sync-fetch', BridgeSyncJob::class);

        Carbon::setTestNow();
    }
}
