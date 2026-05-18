<?php

namespace Tests\Unit\Bridge;

use App\Models\BridgeReplicaPage;
use App\Services\Bridge\BridgeReplicaPageStore;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Tests\TestCase;

class BridgeReplicaPageStoreTest extends TestCase
{
    use RefreshDatabase;

    public function test_store_and_load_rows_round_trip(): void
    {
        $store = app(BridgeReplicaPageStore::class);

        $pageId = $store->storePage(
            datasetSlug: 'stellar',
            mode: 'replication',
            rows: [
                ['ListingKey' => 'STELLAR-1', 'StandardStatus' => 'Active'],
                ['ListingKey' => 'STELLAR-2', 'StandardStatus' => 'Pending'],
            ],
            bridgeUrl: 'https://example.test',
            odataQuery: ['$top' => 2],
        );

        $rows = $store->loadRows($pageId);

        $this->assertCount(2, $rows);
        $this->assertSame('STELLAR-1', $rows[0]['ListingKey']);
    }

    public function test_purge_removes_old_completed_pages(): void
    {
        config([
            'bridge.replica_page_retention_hours' => 24,
            'bridge.replica_page_failed_retention_days' => 7,
        ]);

        $old = now()->subDays(2);
        DB::table('bridge_replica_pages')->insert([
            'dataset_slug' => 'stellar',
            'mode' => 'replication',
            'status' => BridgeReplicaPage::STATUS_COMPLETED,
            'compressed_payload' => base64_encode(gzencode('[]', 9) ?: ''),
            'row_count' => 0,
            'fetched_at' => $old,
            'processed_at' => $old,
            'created_at' => $old,
            'updated_at' => $old,
        ]);

        $deleted = app(BridgeReplicaPageStore::class)->purgeEligibleRows();

        $this->assertSame(1, $deleted);
        $this->assertDatabaseCount('bridge_replica_pages', 0);
    }
}
