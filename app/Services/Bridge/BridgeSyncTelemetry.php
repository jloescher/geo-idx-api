<?php

declare(strict_types=1);

namespace App\Services\Bridge;

use App\Events\Bridge\BridgeReplicationBatchFailed;
use App\Events\Bridge\BridgeReplicationPageFetched;
use App\Events\Bridge\BridgeReplicationPagePersisted;
use Illuminate\Support\Facades\Log;

final class BridgeSyncTelemetry
{
    /**
     * @param  list<array<string, mixed>>  $rows
     * @return array<string, int>
     */
    public static function statusCountsFromRows(array $rows): array
    {
        $counts = [];

        foreach ($rows as $row) {
            if (! is_array($row)) {
                continue;
            }

            $status = strtolower(trim((string) ($row['StandardStatus'] ?? 'unknown')));
            if ($status === '') {
                $status = 'unknown';
            }

            $counts[$status] = ($counts[$status] ?? 0) + 1;
        }

        ksort($counts);

        return $counts;
    }

    public static function sanitizeBridgeUrl(string $url): string
    {
        $parts = parse_url($url);
        if ($parts === false) {
            return $url;
        }

        $path = $parts['path'] ?? $url;
        $query = $parts['query'] ?? '';
        if ($query !== '') {
            parse_str($query, $params);
            unset($params['access_token'], $params['accessToken']);

            return $path.'?'.http_build_query($params);
        }

        return $path;
    }

    /**
     * @param  array<string, mixed>  $odataQuery
     * @param  array<string, int>  $statusCounts
     */
    public function recordPageFetched(
        string $dataset,
        string $mode,
        string $bridgeUrl,
        array $odataQuery,
        int $httpStatus,
        int $listingsDownloaded,
        array $statusCounts,
        bool $replicationStarting,
        bool $hasNextPage,
        int $chainDepth,
    ): void {
        $context = [
            'dataset' => $dataset,
            'mode' => $mode,
            'bridge_url' => self::sanitizeBridgeUrl($bridgeUrl),
            'odata_query' => $odataQuery,
            'http_status' => $httpStatus,
            'listings_downloaded' => $listingsDownloaded,
            'status_counts' => $statusCounts,
            'replication_starting' => $replicationStarting,
            'has_next_page' => $hasNextPage,
            'chain_depth' => $chainDepth,
        ];

        Log::info('bridge.replication.page_fetched', $context);

        event(new BridgeReplicationPageFetched(
            dataset: $dataset,
            mode: $mode,
            bridgeUrl: self::sanitizeBridgeUrl($bridgeUrl),
            odataQuery: $odataQuery,
            httpStatus: $httpStatus,
            listingsDownloaded: $listingsDownloaded,
            statusCounts: $statusCounts,
            replicationStarting: $replicationStarting,
            hasNextPage: $hasNextPage,
            chainDepth: $chainDepth,
        ));
    }

    public function recordPagePersisted(
        string $dataset,
        BridgeReplicaPersistStats $stats,
        ?int $chunkIndex = null,
        ?int $chunkTotal = null,
    ): void {
        $context = array_merge(
            ['dataset' => $dataset],
            $stats->toArray(),
        );

        if ($chunkIndex !== null) {
            $context['chunk_index'] = $chunkIndex;
        }

        if ($chunkTotal !== null) {
            $context['chunk_total'] = $chunkTotal;
        }

        Log::info('bridge.replication.page_persisted', $context);

        event(new BridgeReplicationPagePersisted(
            dataset: $dataset,
            stats: $stats,
            chunkIndex: $chunkIndex,
            chunkTotal: $chunkTotal,
        ));
    }

    /**
     * @param  array<string, mixed>  $odataQuery
     */
    public function recordPageFailed(
        string $dataset,
        string $mode,
        string $failureType,
        string $message,
        ?string $batchId = null,
        ?int $httpStatus = null,
        ?string $bridgeUrl = null,
        array $odataQuery = [],
    ): void {
        $context = array_filter([
            'dataset' => $dataset,
            'mode' => $mode,
            'failure_type' => $failureType,
            'message' => $message,
            'batch_id' => $batchId,
            'http_status' => $httpStatus,
            'bridge_url' => $bridgeUrl !== null ? self::sanitizeBridgeUrl($bridgeUrl) : null,
            'odata_query' => $odataQuery !== [] ? $odataQuery : null,
        ], fn ($value) => $value !== null);

        Log::warning('bridge.replication.failed', $context);

        event(new BridgeReplicationBatchFailed(
            dataset: $dataset,
            mode: $mode,
            failureType: $failureType,
            message: $message,
            batchId: $batchId,
            httpStatus: $httpStatus,
            bridgeUrl: $bridgeUrl !== null ? self::sanitizeBridgeUrl($bridgeUrl) : null,
            odataQuery: $odataQuery,
        ));
    }

    /**
     * @param  list<string>  $datasets
     */
    public function recordKickoff(array $datasets, string $fetchQueue, string $persistQueue): void
    {
        Log::info('bridge.replication.kickoff', [
            'datasets' => $datasets,
            'fetch_queue' => $fetchQueue,
            'persist_queue' => $persistQueue,
        ]);
    }
}
