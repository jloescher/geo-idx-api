<?php

namespace Tests\Unit\Replication;

use App\Models\ListingSyncCursor;
use App\Services\Replication\ReplicationFreshness;
use Carbon\CarbonImmutable;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class ReplicationFreshnessTest extends TestCase
{
    use RefreshDatabase;

    public function test_empty_mirror_is_catch_up(): void
    {
        $freshness = app(ReplicationFreshness::class);

        $this->assertSame(
            ReplicationFreshness::MODE_CATCH_UP,
            $freshness->mode('stellar', 'bridge'),
        );
    }

    public function test_recent_cursor_is_steady(): void
    {
        ListingSyncCursor::query()->create([
            'dataset_slug' => 'stellar',
            'last_bridge_modification_timestamp' => CarbonImmutable::now()->subMinutes(5),
            'replication_in_progress' => false,
            'last_sync_finished_at' => now(),
        ]);

        $freshness = app(ReplicationFreshness::class);

        $this->assertSame(
            ReplicationFreshness::MODE_STEADY,
            $freshness->mode('stellar', 'bridge'),
        );
    }

    public function test_stale_cursor_is_catch_up(): void
    {
        ListingSyncCursor::query()->create([
            'dataset_slug' => 'stellar',
            'last_bridge_modification_timestamp' => CarbonImmutable::now()->subMinutes(30),
            'replication_in_progress' => false,
            'last_sync_finished_at' => now()->subHour(),
        ]);

        $freshness = app(ReplicationFreshness::class);

        $this->assertSame(
            ReplicationFreshness::MODE_CATCH_UP,
            $freshness->mode('stellar', 'bridge'),
        );
    }
}
