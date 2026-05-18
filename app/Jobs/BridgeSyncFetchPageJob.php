<?php

namespace App\Jobs;

use App\Services\Bridge\BridgeReplicaCursorPatch;
use App\Services\Bridge\BridgeSyncPageResult;
use App\Services\Bridge\BridgeSyncService;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\Bus;
use Illuminate\Support\Facades\Log;

/**
 * Revenue impact: one Bridge page per job respects burst/hourly limits; DB work is delegated
 * to chained BridgePersistReplicaChunkJob batches so HTTP and Postgres scale independently.
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
    ) {}

    public function handle(BridgeSyncService $sync): void
    {
        $maxChain = (int) config('bridge.sync_max_chained_fetch_pages', 0);
        if ($maxChain > 0 && $this->chainDepth >= $maxChain) {
            Log::warning('bridge.sync.chain_cap', [
                'dataset' => $this->dataset,
                'mode' => $this->mode,
                'chain_depth' => $this->chainDepth,
            ]);

            return;
        }

        $cursor = $sync->cursorForDataset($this->dataset);

        $result = $this->mode === 'replication'
            ? $sync->fetchReplicationPage($this->dataset, $cursor)
            : $sync->fetchIncrementalPage($this->dataset, $cursor, $this->incrementalSkip);

        $queue = (string) config('bridge.sync_queue', 'bridge-sync');

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

        if ($result->rows === [] && $this->mode === 'incremental' && ! $result->incrementalHasMore) {
            $sync->applyCursorPatch($this->dataset, new BridgeReplicaCursorPatch(
                markSyncFinished: true,
            ));

            return;
        }

        [$patch, $dispatchIncremental, $nextFetch] = $this->continuationPlan($result);

        $this->dispatchPersistChain(
            $queue,
            $result->rows,
            $patch,
            $dispatchIncremental,
            $nextFetch['mode'] ?? null,
            $nextFetch['skip'] ?? 0,
            $nextFetch['chain'] ?? 0,
        );
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     */
    private function dispatchPersistChain(
        string $queue,
        array $rows,
        ?BridgeReplicaCursorPatch $patch,
        bool $dispatchIncremental,
        ?string $nextFetchMode,
        int $nextIncrementalSkip,
        int $nextChainDepth,
    ): void {
        if ($rows === []) {
            BridgePersistReplicaFinalizeJob::dispatch(
                $this->dataset,
                $patch,
                $dispatchIncremental,
                $nextFetchMode,
                $nextIncrementalSkip,
                $nextChainDepth,
            )->onQueue($queue);

            return;
        }

        $chunkSize = max(1, (int) config('bridge.sync_persist_job_chunk_size', 100));
        $chunks = array_chunk($rows, $chunkSize);
        $lastIndex = count($chunks) - 1;

        $chain = [];
        foreach ($chunks as $index => $chunk) {
            $isLast = $index === $lastIndex;
            $chain[] = new BridgePersistReplicaChunkJob(
                dataset: $this->dataset,
                rows: $chunk,
                cursorPatch: $isLast ? $patch : null,
                dispatchIncrementalAfter: $isLast && $dispatchIncremental,
                nextFetchMode: $isLast ? $nextFetchMode : null,
                nextIncrementalSkip: $nextIncrementalSkip,
                nextChainDepth: $nextChainDepth,
            );
        }

        Bus::chain($chain)
            ->onQueue($queue)
            ->dispatch();
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
