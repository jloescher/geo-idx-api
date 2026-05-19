<?php

namespace Tests\Unit\Replication;

use App\Models\ReplicaPage;
use App\Services\Replication\ReplicaPageStore;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Tests\TestCase;

class ReplicaPageStoreTest extends TestCase
{
    use RefreshDatabase;

    public function test_store_page_and_load_chunk_from_v2_multi_part_payload(): void
    {
        $store = app(ReplicaPageStore::class);
        $rows = [];
        for ($i = 0; $i < 3; $i++) {
            $rows[] = [
                'ListingKey' => 'K'.$i,
                'StandardStatus' => 'Active',
            ];
        }

        config(['mls.datasets.stellar.persist_chunk_size' => 2]);

        $pageId = $store->storePage(
            datasetSlug: 'stellar',
            mode: 'replication',
            rows: $rows,
            bridgeUrl: 'https://bridge.test/page',
            odataQuery: ['$top' => 2000],
        );

        $this->assertSame(2, $store->chunkCountForPage($pageId));

        $chunkOne = $store->rowsForChunk($pageId, 1, 2);
        $this->assertCount(2, $chunkOne);
        $this->assertSame('K0', $chunkOne[0]['ListingKey']);

        $chunkTwo = $store->rowsForChunk($pageId, 2, 2);
        $this->assertCount(1, $chunkTwo);
        $this->assertSame('K2', $chunkTwo[0]['ListingKey']);
    }

    public function test_purge_eligible_rows_deletes_old_completed_pages(): void
    {
        $now = now();
        DB::table('replica_pages')->insert([
            'provider' => 'bridge',
            'dataset_slug' => 'stellar',
            'mode' => 'replication',
            'status' => ReplicaPage::STATUS_COMPLETED,
            'row_count' => 0,
            'fetched_at' => $now,
            'processed_at' => $now->copy()->subDays(2),
            'created_at' => $now,
            'updated_at' => $now,
        ]);

        $deleted = app(ReplicaPageStore::class)->purgeEligibleRows();

        $this->assertSame(1, $deleted);
        $this->assertDatabaseCount('replica_pages', 0);
    }
}
