<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeReplicaPageStore;
use App\Services\Bridge\BridgeSyncPageResult;
use App\Services\Bridge\BridgeSyncService;
use App\Services\Bridge\BridgeSyncTelemetry;
use Illuminate\Bus\Batch;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\Bus;
use Throwable;

/**
 * Revenue impact: one Bridge page per job respects burst/hourly limits; DB work runs in parallel
 * persist batches on a dedicated queue so HTTP throttling and Postgres throughput stay independent.
 */
class BridgeSyncFetchPageJob implements ShouldQueue
{
    use Dispatchable, Queueable;

    public int $timeout = 120;

    public function __construct(
        public string $dataset,
        public string $mode,
        public int $incrementalSkip = 0,
        public int $chainDepth = 0,
    ) {
        $this->onQueue((string) config('bridge.sync_fetch_queue', 'bridge-sync-fetch'));
    }

    /**
     * @return list<string>
     */
    public function tags(): array
    {
        return ['bridge-replication', 'dataset:'.$this->dataset, 'mode:'.$this->mode];
    }

    public function handle(
        BridgeSyncService $sync,
        BridgeSyncTelemetry $telemetry,
        BridgeReplicaPageStore $pageStore,
    ): void {
        $maxChain = (int) config('bridge.sync_max_chained_fetch_pages', 0);
        if ($maxChain > 0 && $this->chainDepth >= $maxChain) {
            $telemetry->recordPageFailed(
                dataset: $this->dataset,
                mode: $this->mode,
                failureType: 'chain_cap',
                message: 'Bridge sync fetch chain depth cap reached',
            );

            return;
        }

        $cursor = $sync->cursorForDataset($this->dataset);

        if ($this->mode === 'incremental' && $cursor->replication_in_progress) {
            return;
        }

        $result = $this->mode === 'replication'
            ? $sync->fetchReplicationPage($this->dataset, $cursor)
            : $sync->fetchIncrementalPage($this->dataset, $cursor, $this->incrementalSkip);

        if ($result->forbidden) {
            $sync->applyCursorPatch($this->dataset, new BridgeReplicaCursorPatch(
                applyReplicationState: true,
                replicationNextUrl: null,
                replicationInProgress: false,
            ));

            return;
        }

        if ($result->httpError) {
            return;
        }

        if ($result->bridgeUrl !== null) {
            $telemetry->recordPageFetched(
                dataset: $this->dataset,
                mode: $this->mode,
                bridgeUrl: $result->bridgeUrl,
                odataQuery: $result->odataQuery,
                httpStatus: $result->httpStatus,
                listingsDownloaded: count($result->rows),
                statusCounts: BridgeSyncTelemetry::statusCountsFromRows($result->rows),
                replicationStarting: $result->replicationStarting,
                hasNextPage: $this->mode === 'replication'
                    ? ! $result->replicationComplete
                    : $result->incrementalHasMore,
                chainDepth: $this->chainDepth,
            );
        }

        if ($result->rows === [] && $this->mode === 'incremental' && ! $result->incrementalHasMore) {
            $sync->applyCursorPatch($this->dataset, new BridgeReplicaCursorPatch(
                markSyncFinished: true,
            ));

            return;
        }

        [$patch, $dispatchIncremental, $nextFetch] = $this->continuationPlan($result);

        $this->dispatchPersistBatch(
            $result,
            $patch,
            $dispatchIncremental,
            $nextFetch['mode'] ?? null,
            $nextFetch['skip'] ?? 0,
            $nextFetch['chain'] ?? 0,
            $telemetry,
            $pageStore,
        );
    }

    private function dispatchPersistBatch(
        BridgeSyncPageResult $result,
        ?BridgeReplicaCursorPatch $patch,
        bool $dispatchIncremental,
        ?string $nextFetchMode,
        int $nextIncrementalSkip,
        int $nextChainDepth,
        BridgeSyncTelemetry $telemetry,
        BridgeReplicaPageStore $pageStore,
    ): void {
        $persistQueue = (string) config('bridge.sync_persist_queue', 'bridge-sync-persist');

        if ($result->rows === []) {
            BridgePersistReplicaFinalizeJob::dispatch(
                dataset: $this->dataset,
                replicaPageId: null,
                cursorPatch: $patch,
                dispatchIncrementalAfter: $dispatchIncremental,
                nextFetchMode: $nextFetchMode,
                nextIncrementalSkip: $nextIncrementalSkip,
                nextChainDepth: $nextChainDepth,
            )->onQueue($persistQueue);

            return;
        }

        $pageId = $pageStore->storePage(
            datasetSlug: $this->dataset,
            mode: $this->mode,
            rows: $result->rows,
            bridgeUrl: $result->bridgeUrl,
            odataQuery: $result->odataQuery,
        );

        $chunkSize = max(1, (int) config('bridge.sync_persist_job_chunk_size', 100));
        $chunkTotal = (int) max(1, (int) ceil(count($result->rows) / $chunkSize));
        $jobs = [];

        for ($index = 0; $index < $chunkTotal; $index++) {
            $jobs[] = new BridgePersistReplicaChunkJob(
                dataset: $this->dataset,
                pageId: $pageId,
                chunkIndex: $index + 1,
                chunkTotal: $chunkTotal,
            );
        }

        $dataset = $this->dataset;
        $mode = $this->mode;

        $batch = Bus::batch($jobs)
            ->name('bridge-replica-persist:'.$dataset)
            ->onQueue($persistQueue)
            ->then(function () use (
                $dataset,
                $pageId,
                $patch,
                $dispatchIncremental,
                $nextFetchMode,
                $nextIncrementalSkip,
                $nextChainDepth,
                $persistQueue,
            ): void {
                BridgePersistReplicaFinalizeJob::dispatch(
                    dataset: $dataset,
                    replicaPageId: $pageId,
                    cursorPatch: $patch,
                    dispatchIncrementalAfter: $dispatchIncremental,
                    nextFetchMode: $nextFetchMode,
                    nextIncrementalSkip: $nextIncrementalSkip,
                    nextChainDepth: $nextChainDepth,
                )->onQueue($persistQueue);
            })
            ->catch(function (Batch $batch, Throwable $e) use ($dataset, $mode, $pageId, $telemetry, $pageStore): void {
                $pageStore->markFailed($pageId);
                $telemetry->recordPageFailed(
                    dataset: $dataset,
                    mode: $mode,
                    failureType: 'persist_batch_failed',
                    message: $e->getMessage(),
                    batchId: $batch->id,
                );
            })
            ->dispatch();

        $pageStore->markProcessing($pageId, $batch->id);
    }

    /**
     * @return array{0: ?BridgeReplicaCursorPatch, 1: bool, 2: ?array{mode: string, skip: int, chain: int}}
     */
    private function continuationPlan(BridgeSyncPageResult $result): array
    {
        if ($this->mode === 'replication') {
            $patch = new BridgeReplicaCursorPatch(
                applyReplicationState: true,
                replicationNextUrl: $result->nextReplicationUrl,
                replicationInProgress: ! $result->replicationComplete,
                maxBridgeTs: $result->maxBridgeTs,
            );

            if (! $result->replicationComplete && $result->nextReplicationUrl !== null) {
                return [$patch, false, [
                    'mode' => 'replication',
                    'skip' => 0,
                    'chain' => $this->chainDepth + 1,
                ]];
            }

            $dispatchIncremental = $result->replicationComplete && $result->maxBridgeTs !== null;

            return [$patch, $dispatchIncremental, null];
        }

        $patch = new BridgeReplicaCursorPatch(
            maxBridgeTs: $result->maxBridgeTs,
            markSyncFinished: ! $result->incrementalHasMore,
        );

        if ($result->incrementalHasMore) {
            return [$patch, false, [
                'mode' => 'incremental',
                'skip' => $result->nextIncrementalSkip,
                'chain' => $this->chainDepth + 1,
            ]];
        }

        return [$patch, false, null];
    }
}
