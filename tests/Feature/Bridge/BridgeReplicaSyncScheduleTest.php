<?php

namespace Tests\Feature\Bridge;

use App\Jobs\BridgeSyncJob;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Queue;
use Tests\TestCase;

class BridgeReplicaSyncScheduleTest extends TestCase
{
    public function test_schedule_run_dispatches_bridge_sync_kickoff_on_bridge_sync_queue(): void
    {
        config(['bridge.sync_queue' => 'bridge-sync']);
        Queue::fake();

        Carbon::setTestNow(Carbon::create(2026, 5, 15, 12, 15, 0, 'UTC'));

        $this->artisan('schedule:run')->assertSuccessful();

        Queue::assertPushed(BridgeSyncJob::class);
        Queue::assertPushedOn('bridge-sync', BridgeSyncJob::class);

        Carbon::setTestNow();
    }
}
