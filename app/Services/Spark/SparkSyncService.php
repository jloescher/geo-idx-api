<?php

declare(strict_types=1);

namespace App\Services\Spark;

use App\Enums\ListingMirrorProvider;
use App\Models\Listing;
use App\Models\ListingSyncCursor;
use App\Services\Bridge\BridgeReplicaPersistStats;
use App\Services\Bridge\BridgeSyncTelemetry;
use App\Services\Mls\ListingMirrorWriter;
use App\Services\Replication\MlsReplicationService;
use App\Services\Replication\ReplicationCursorPatch;
use App\Services\Replication\ReplicationPageResult;
use Carbon\CarbonImmutable;
use Illuminate\Http\Client\Response;
use Illuminate\Support\Facades\Log;

/**
 * Revenue impact: BeachesMLS mirror in Postgres cuts live Spark OData spend for map/search.
 *
 * Compliance: replication stores Active/Pending only; closed inventory stays live API.
 */
final class SparkSyncService extends MlsReplicationService
{
    public function __construct(
        private readonly SparkHttpService $http,
        private readonly BridgeSyncTelemetry $telemetry,
        private readonly ListingMirrorWriter $mirrorWriter,
    ) {}

    public function fetchReplicationPage(string $dataset, ListingSyncCursor $cursor): ReplicationPageResult
    {
        $top = (int) config('spark.sync_replication_top', 1000);
        $replicationStarting = ($cursor->replication_next_url === null || $cursor->replication_next_url === '')
            && ! $cursor->replication_in_progress
            && ! Listing::query()->where('dataset_slug', $dataset)->exists();

        if ($cursor->replication_next_url !== null && $cursor->replication_next_url !== '') {
            $url = $cursor->replication_next_url;
            $query = [];
        } else {
            $url = $this->http->replicationPropertyCollectionUrl();
            $query = $this->replicationQuery($top);
        }

        return $this->executePageFetch($dataset, $url, $query, $replicationStarting, 'replication');
    }

    public function fetchIncrementalPage(string $dataset, ListingSyncCursor $cursor, int $skip): ReplicationPageResult
    {
        if ($cursor->last_bridge_modification_timestamp === null) {
            return new ReplicationPageResult(
                rows: [],
                nextReplicationUrl: null,
                replicationComplete: true,
                incrementalHasMore: false,
                nextIncrementalSkip: 0,
                maxBridgeTs: null,
            );
        }

        if ($cursor->replication_next_url !== null && $cursor->replication_next_url !== '') {
            $url = $cursor->replication_next_url;
            $query = [];
        } else {
            $windowEnd = $cursor->incremental_window_end ?? CarbonImmutable::now('UTC')->subMinute();
            $windowStart = CarbonImmutable::parse($cursor->last_bridge_modification_timestamp->format(\DateTimeInterface::ATOM));
            $url = $this->http->replicationPropertyCollectionUrl();
            $query = $this->incrementalQuery($windowStart, $windowEnd, (int) config('spark.sync_incremental_top', 1000));
        }

        return $this->executePageFetch($dataset, $url, $query, false, 'incremental');
    }

    /**
     * @param  array<string, scalar|list<string>>  $query
     */
    private function executePageFetch(
        string $dataset,
        string $url,
        array $query,
        bool $replicationStarting,
        string $mode,
    ): ReplicationPageResult {
        $response = $this->http->serverJsonGet($url, $query);

        if ($response->status() === 403) {
            $this->telemetry->recordPageFailed(
                dataset: $dataset,
                mode: $mode,
                failureType: 'forbidden',
                message: 'Spark replication returned 403',
                httpStatus: 403,
                bridgeUrl: $url,
                odataQuery: $query,
            );

            return ReplicationPageResult::forbidden($url, $query, 403);
        }

        if (! $response->successful()) {
            $this->telemetry->recordPageFailed(
                dataset: $dataset,
                mode: $mode,
                failureType: 'http_error',
                message: 'Spark replication HTTP error',
                httpStatus: $response->status(),
                bridgeUrl: $url,
                odataQuery: $query,
            );

            return ReplicationPageResult::httpError($url, $query, $response->status());
        }

        $body = $response->json();
        $value = is_array($body['value'] ?? null) ? $body['value'] : [];
        $next = $this->extractNextUrl($response);
        $maxTs = $this->maxModificationTimestampFromRows($value);

        return new ReplicationPageResult(
            rows: $value,
            nextReplicationUrl: $next,
            replicationComplete: $next === null,
            incrementalHasMore: $next !== null,
            nextIncrementalSkip: 0,
            maxBridgeTs: $maxTs,
            replicationStarting: $replicationStarting,
            bridgeUrl: $url,
            odataQuery: $query,
            httpStatus: $response->status(),
        );
    }

    /**
     * @return array<string, scalar|list<string>>
     */
    private function replicationQuery(int $top): array
    {
        return [
            '$top' => $top,
            '$expand' => (string) config('spark.sync_expand', 'Media,Unit,Room,OpenHouse'),
            '$filter' => $this->activePendingFilter(),
        ];
    }

    /**
     * @return array<string, scalar|list<string>>
     */
    private function incrementalQuery(CarbonImmutable $start, CarbonImmutable $end, int $top): array
    {
        $startIso = $start->utc()->format('Y-m-d\TH:i:s\Z');
        $endIso = $end->utc()->format('Y-m-d\TH:i:s\Z');

        return [
            '$top' => $top,
            '$expand' => (string) config('spark.sync_expand', 'Media,Unit,Room,OpenHouse'),
            '$filter' => "({$this->activePendingFilter()}) and ModificationTimestamp gt {$startIso} and ModificationTimestamp lt {$endIso}",
        ];
    }

    private function activePendingFilter(): string
    {
        return "StandardStatus eq 'Active' or StandardStatus eq 'Pending'";
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     */
    public function persistChunk(string $dataset, array $rows, ?ReplicationCursorPatch $patch = null): BridgeReplicaPersistStats
    {
        $stats = $rows === []
            ? new BridgeReplicaPersistStats
            : $this->mirrorWriter->hydrateReplicaBatch($dataset, $rows, ListingMirrorProvider::Spark);

        if ($patch !== null) {
            $this->applyCursorPatch($dataset, $patch);
        }

        return $stats;
    }

    protected function onSyncFinished(ListingSyncCursor $cursor, ReplicationCursorPatch $patch): void
    {
        if ($cursor->incremental_window_end instanceof \DateTimeInterface) {
            $cursor->last_bridge_modification_timestamp = CarbonImmutable::parse(
                $cursor->incremental_window_end->format(\DateTimeInterface::ATOM)
            );
        }
    }

    public function prepareIncrementalWindow(string $dataset): CarbonImmutable
    {
        $windowEnd = CarbonImmutable::now('UTC')->subMinute();
        $cursor = $this->cursorForDataset($dataset);
        $cursor->incremental_window_end = $windowEnd;
        $cursor->save();

        return $windowEnd;
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     */
    private function maxModificationTimestampFromRows(array $rows): ?CarbonImmutable
    {
        $maxTs = null;
        foreach ($rows as $row) {
            if (! is_array($row)) {
                continue;
            }
            $raw = $row['ModificationTimestamp'] ?? null;
            if (! is_string($raw) || $raw === '') {
                continue;
            }
            try {
                $parsed = CarbonImmutable::parse($raw);
                if ($maxTs === null || $parsed->gt($maxTs)) {
                    $maxTs = $parsed;
                }
            } catch (\Throwable) {
                Log::info('spark.sync.invalid_modification_timestamp', ['value' => $raw]);
            }
        }

        return $maxTs;
    }

    private function extractNextUrl(Response $response): ?string
    {
        $json = $response->json();
        if (is_array($json) && isset($json['@odata.nextLink']) && is_string($json['@odata.nextLink'])) {
            return $json['@odata.nextLink'];
        }

        return null;
    }
}
