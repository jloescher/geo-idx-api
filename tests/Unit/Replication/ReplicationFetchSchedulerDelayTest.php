<?php

namespace Tests\Unit\Replication;

use App\Jobs\BridgeSyncFetchPageJob;
use App\Models\ListingSyncCursor;
use App\Services\Bridge\BridgeSyncFetchScheduler;
use Carbon\CarbonImmutable;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Queue;
use Tests\TestCase;

class ReplicationFetchSchedulerDelayTest extends TestCase
{
    use RefreshDatabase;

    public function test_steady_idle_poll_uses_fifteen_minute_delay(): void
    {
        Queue::fake();
        Cache::flush();

        ListingSyncCursor::query()->create([
            'dataset_slug' => 'stellar',
            'last_bridge_modification_timestamp' => CarbonImmutable::now()->subMinutes(5),
            'replication_in_progress' => false,
            'last_sync_finished_at' => now(),
        ]);

        config(['mls.steady_incremental_poll_minutes' => 15]);

        $scheduler = app(BridgeSyncFetchScheduler::class);
        $scheduler->dispatchIncremental('stellar', idlePoll: true);

        Queue::assertPushed(BridgeSyncFetchPageJob::class, function ($job): bool {
            return $job->delay !== null && $job->delay->greaterThan(now()->addMinutes(14));
        });
    }

    public function test_catch_up_chain_uses_rate_limit_delay_only(): void
    {
        Queue::fake();
        Cache::flush();

        config(['bridge.sync_max_requests_per_second' => 2]);

        $scheduler = app(BridgeSyncFetchScheduler::class);
        $scheduler->dispatchNext('stellar', 'replication', chainDepth: 1, idlePoll: false);

        Queue::assertPushed(BridgeSyncFetchPageJob::class, function ($job): bool {
            return $job->delay !== null && $job->delay->lessThan(now()->addSeconds(2));
        });
    }
}
