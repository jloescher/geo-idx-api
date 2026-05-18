<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeReplicaPageStore;
use App\Services\Bridge\BridgeSyncPageResult;
use App\Services\Bridge\BridgeSyncTelemetry;
use App\Services\Spark\SparkSyncService;
use Illuminate\Bus\Batch;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\Bus;
use Throwable;

class SparkSyncFetchPageJob implements ShouldQueue
{
    use Dispatchable, Queueable;

    public int $timeout = 300;

    public function __construct(
        public string $dataset,
        public string $mode,
        public int $incrementalSkip = 0,
        public int $chainDepth = 0,
    ) {
        $this->onQueue((string) config('spark.sync_fetch_queue', 'spark-sync-fetch'));
    }

    /**
     * @return list<string>
     */
    public function tags(): array
    {
        return ['spark-replication', 'dataset:'.$this->dataset, 'mode:'.$this->mode];
    }

    public function handle(
        SparkSyncService $sync,
        BridgeSyncTelemetry $telemetry,
        BridgeReplicaPageStore $pageStore,
    ): void {
        $maxChain = (int) config('spark.sync_max_chained_fetch_pages', 0);
        if ($maxChain > 0 && $this->chainDepth >= $maxChain) {
            $telemetry->recordPageFailed(
                dataset: $this->dataset,
                mode: $this->mode,
                failureType: 'chain_cap',
                message: 'Spark sync fetch chain depth cap reached',
            );

            return;
        }

        $cursor = $sync->cursorForDataset($this->dataset);

        if ($this->mode === 'incremental' && $cursor->replication_in_progress) {
            return;
        }

        if ($this->mode === 'incremental' && $this->chainDepth === 0) {
            $sync->prepareIncrementalWindow($this->dataset);
            $cursor = $sync->cursorForDataset($this->dataset);
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
                hasNextPage: ! $result->replicationComplete,
                chainDepth: $this->chainDepth,
            );
        }

        if ($result->rows === [] && $this->mode === 'incremental' && $result->replicationComplete) {
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
        int $nextChainDepth,
        BridgeSyncTelemetry $telemetry,
        BridgeReplicaPageStore $pageStore,
    ): void {
        $persistQueue = (string) config('spark.sync_persist_queue', 'spark-sync-persist');

        if ($result->rows === []) {
            SparkPersistReplicaFinalizeJob::dispatch(
                dataset: $this->dataset,
                replicaPageId: null,
                cursorPatch: $patch,
                dispatchIncrementalAfter: $dispatchIncremental,
                nextFetchMode: $nextFetchMode,
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
            provider: 'spark',
        );

        $chunkSize = max(1, (int) config('spark.sync_persist_job_chunk_size', 50));
        $chunkTotal = (int) max(1, (int) ceil(count($result->rows) / $chunkSize));
        $jobs = [];

        for ($index = 0; $index < $chunkTotal; $index++) {
            $jobs[] = new SparkPersistReplicaChunkJob(
                dataset: $this->dataset,
                pageId: $pageId,
                chunkIndex: $index + 1,
                chunkTotal: $chunkTotal,
            );
        }

        $dataset = $this->dataset;
        $mode = $this->mode;

        $batch = Bus::batch($jobs)
            ->name('spark-replica-persist:'.$dataset)
            ->onQueue($persistQueue)
            ->then(function () use (
                $dataset,
                $pageId,
                $patch,
                $dispatchIncremental,
                $nextFetchMode,
                $nextChainDepth,
                $persistQueue,
            ): void {
                SparkPersistReplicaFinalizeJob::dispatch(
                    dataset: $dataset,
                    replicaPageId: $pageId,
                    cursorPatch: $patch,
                    dispatchIncrementalAfter: $dispatchIncremental,
                    nextFetchMode: $nextFetchMode,
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
     * @return array{0: ?BridgeReplicaCursorPatch, 1: bool, 2: ?array{mode: string, chain: int}}
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
                    'chain' => $this->chainDepth + 1,
                ]];
            }

            $dispatchIncremental = $result->replicationComplete && $result->maxBridgeTs !== null;

            return [$patch, $dispatchIncremental, null];
        }

        $patch = new BridgeReplicaCursorPatch(
            applyReplicationState: true,
            replicationNextUrl: $result->replicationComplete ? null : $result->nextReplicationUrl,
            replicationInProgress: false,
            maxBridgeTs: $result->maxBridgeTs,
            markSyncFinished: $result->replicationComplete,
        );

        if (! $result->replicationComplete && $result->nextReplicationUrl !== null) {
            return [$patch, false, [
                'mode' => 'incremental',
                'chain' => $this->chainDepth + 1,
            ]];
        }

        return [$patch, false, null];
    }
}
