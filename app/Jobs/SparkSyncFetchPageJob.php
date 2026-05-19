<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeSyncTelemetry;
use App\Services\Mls\MlsDatasetRegistry;
use App\Services\Replication\ReplicaPageStore;
use App\Services\Replication\ReplicationCursorPatch;
use App\Services\Replication\ReplicationPageResult;
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
        ReplicaPageStore $pageStore,
        MlsDatasetRegistry $datasets,
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
            $sync->applyCursorPatch($this->dataset, new ReplicationCursorPatch(
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
            $sync->applyCursorPatch($this->dataset, new ReplicationCursorPatch(
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
            $datasets,
        );
    }

    private function dispatchPersistBatch(
        ReplicationPageResult $result,
        ?ReplicationCursorPatch $patch,
        bool $dispatchIncremental,
        ?string $nextFetchMode,
        int $nextChainDepth,
        BridgeSyncTelemetry $telemetry,
        ReplicaPageStore $pageStore,
        MlsDatasetRegistry $datasets,
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

        $chunkTotal = $pageStore->chunkCountForPage($pageId);
        $chunkJobs = [];

        for ($index = 0; $index < $chunkTotal; $index++) {
            $chunkJobs[] = new SparkPersistReplicaChunkJob(
                dataset: $this->dataset,
                pageId: $pageId,
                chunkIndex: $index + 1,
                chunkTotal: $chunkTotal,
            );
        }

        $finalizeJob = new SparkPersistReplicaFinalizeJob(
            dataset: $this->dataset,
            replicaPageId: $pageId,
            cursorPatch: $patch,
            dispatchIncrementalAfter: $dispatchIncremental,
            nextFetchMode: $nextFetchMode,
            nextChainDepth: $nextChainDepth,
        );

        $dataset = $this->dataset;
        $mode = $this->mode;

        if ($datasets->persistSequential($dataset)) {
            $chainJobs = $chunkJobs;
            $chainJobs[] = $finalizeJob;
            Bus::chain($chainJobs)->onQueue($persistQueue)->dispatch();
            $pageStore->markProcessing($pageId);

            return;
        }

        $batch = Bus::batch($chunkJobs)
            ->name('spark-replica-persist:'.$dataset)
            ->onQueue($persistQueue)
            ->then(function () use ($finalizeJob, $persistQueue): void {
                dispatch($finalizeJob)->onQueue($persistQueue);
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
     * @return array{0: ?ReplicationCursorPatch, 1: bool, 2: ?array{mode: string, chain: int}}
     */
    private function continuationPlan(ReplicationPageResult $result): array
    {
        if ($this->mode === 'replication') {
            $patch = new ReplicationCursorPatch(
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

        $patch = new ReplicationCursorPatch(
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
