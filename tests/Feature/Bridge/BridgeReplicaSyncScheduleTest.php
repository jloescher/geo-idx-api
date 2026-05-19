<?php

namespace Tests\Feature\Bridge;

use App\Jobs\MlsReplicationKickoffJob;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Queue;
use Tests\TestCase;

class BridgeReplicaSyncScheduleTest extends TestCase
{
    public function test_schedule_run_dispatches_catch_up_kickoff_job(): void
    {
        Queue::fake();

        Carbon::setTestNow(Carbon::create(2026, 5, 15, 12, 1, 0, 'UTC'));

        $this->artisan('schedule:run')->assertSuccessful();

        Queue::assertPushed(MlsReplicationKickoffJob::class);

        Carbon::setTestNow();
    }
}
