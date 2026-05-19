<?php

namespace Tests\Feature\Bridge;

use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Tests\TestCase;

class BridgeMigrateLegacyQueueJobsCommandTest extends TestCase
{
    use RefreshDatabase;

    public function test_migrates_jobs_from_legacy_bridge_sync_queue(): void
    {
        DB::table('jobs')->insert([
            'queue' => 'bridge-sync',
            'payload' => '{}',
            'attempts' => 0,
            'reserved_at' => null,
            'available_at' => now()->timestamp,
            'created_at' => now()->timestamp,
        ]);

        $this->artisan('bridge:migrate-legacy-queue-jobs')
            ->assertSuccessful();

        $this->assertSame(0, DB::table('jobs')->where('queue', 'bridge-sync')->count());
        $this->assertSame(1, DB::table('jobs')->where('queue', 'bridge-sync-fetch')->count());
    }

    public function test_dry_run_does_not_update_rows(): void
    {
        DB::table('jobs')->insert([
            'queue' => 'bridge-sync',
            'payload' => '{}',
            'attempts' => 0,
            'reserved_at' => null,
            'available_at' => now()->timestamp,
            'created_at' => now()->timestamp,
        ]);

        $this->artisan('bridge:migrate-legacy-queue-jobs', ['--dry-run' => true])
            ->assertSuccessful();

        $this->assertSame(1, DB::table('jobs')->where('queue', 'bridge-sync')->count());
    }
}
