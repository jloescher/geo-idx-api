<?php

namespace App\Services\Bridge;

use App\Enums\ListingMirrorProvider;
use App\Models\Listing;
use App\Models\ListingSyncCursor;
use App\Services\Mls\ListingMirrorWriter;
use App\Services\Replication\MlsReplicationService;
use App\Services\Replication\ReplicationCursorPatch;
use App\Services\Replication\ReplicationPageResult;
use Carbon\CarbonImmutable;
use Illuminate\Http\Client\Response;
use Illuminate\Support\Facades\Log;

/**
 * Revenue impact: compliant Bridge replication lowers per-search MLS costs while keeping
 * idx-api maps fast; reckless polling risks API suspension — see Bridge RESO docs and
 * docs/bridge_interactive/reso_web_api.md ($top 200 / 2000, Link next, ModificationTimestamp).
 *
 * Compliance: Stellar MLS IDX / MLS Grid Rule 11 — server-token access only; display rules
 * remain enforced downstream. Replication is MLS-discretionary — handle 403 without tight retries.
 *
 * Mirror policy: indexed rows are Active + Pending only; Closed / other statuses reconcile via
 * delete batches so teaser surfaces never leak wrong lifecycle states from local cache.
 */
final class BridgeSyncService extends MlsReplicationService
{
    public function __construct(
        private readonly BridgeHttpService $http,
        private readonly BridgeSyncTelemetry $telemetry,
        private readonly ListingMirrorWriter $mirrorWriter,
    ) {}

    public function fetchReplicationPage(string $dataset, ListingSyncCursor $cursor): ReplicationPageResult
    {
        $top = (int) config('bridge.sync_replication_top', 2000);
        $replicationStarting = ($cursor->replication_next_url === null || $cursor->replication_next_url === '')
            && ! $cursor->replication_in_progress
            && ! Listing::query()->where('dataset_slug', $dataset)->exists();

        if ($cursor->replication_next_url !== null && $cursor->replication_next_url !== '') {
            $url = $cursor->replication_next_url;
            $query = [];
        } else {
            $url = $this->buildPropertyReplicationUrl($dataset);
            $query = array_merge([
                '$filter' => $this->replicationActivePendingFilter(),
                '$top' => $top,
                '$select' => $this->replicationSelectList($dataset),
            ], $this->replicationMediaQueryParams());
        }

        $response = $this->http->serverJsonGet($url, $query);

        if ($response->status() === 403) {
            $this->telemetry->recordPageFailed(
                dataset: $dataset,
                mode: 'replication',
                failureType: 'forbidden',
                message: 'Bridge replication returned 403',
                httpStatus: 403,
                bridgeUrl: $url,
                odataQuery: $query,
            );

            return ReplicationPageResult::forbidden($url, $query, 403);
        }

        if (! $response->successful()) {
            $this->telemetry->recordPageFailed(
                dataset: $dataset,
                mode: 'replication',
                failureType: 'http_error',
                message: 'Bridge replication HTTP error',
                httpStatus: $response->status(),
                bridgeUrl: $url,
                odataQuery: $query,
            );

            return ReplicationPageResult::httpError($url, $query, $response->status());
        }

        $body = $response->json();
        $value = is_array($body['value'] ?? null) ? $body['value'] : [];
        $next = $this->extractNextUrl($response);
        $maxBridgeTs = $this->maxBridgeTimestampFromRows($value);

        return new ReplicationPageResult(
            rows: $value,
            nextReplicationUrl: $next,
            replicationComplete: $next === null,
            incrementalHasMore: false,
            nextIncrementalSkip: 0,
            maxBridgeTs: $maxBridgeTs,
            replicationStarting: $replicationStarting,
            bridgeUrl: $url,
            odataQuery: $query,
            httpStatus: $response->status(),
        );
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

        $top = (int) config('bridge.sync_incremental_top', 200);
        $select = $this->syncSelectList($dataset);
        $filterTs = CarbonImmutable::parse($cursor->last_bridge_modification_timestamp->format(\DateTimeInterface::ATOM));
        $filterLiteral = "ModificationTimestamp gt datetime'".$filterTs->utc()->format('Y-m-d\TH:i:s\Z')."'";
        $orderby = 'ModificationTimestamp asc';
        $url = $this->buildPropertyCollectionUrl($dataset);

        $query = array_merge([
            '$filter' => $filterLiteral,
            '$orderby' => $orderby,
            '$top' => $top,
            '$skip' => $skip,
            '$select' => $select,
        ], $this->mediaQueryParams());

        $response = $this->http->serverJsonGet($url, $query);

        if ($response->status() === 400 || $response->status() === 501) {
            Log::info('bridge.incremental.fallback_bridge_ts', ['dataset' => $dataset]);
            $filterLiteral = "BridgeModificationTimestamp gt datetime'".$filterTs->utc()->format('Y-m-d\TH:i:s\Z')."'";
            $response = $this->http->serverJsonGet($url, array_merge([
                '$filter' => $filterLiteral,
                '$orderby' => 'BridgeModificationTimestamp asc',
                '$top' => $top,
                '$skip' => $skip,
                '$select' => $select,
            ], $this->mediaQueryParams()));
        }

        if (! $response->successful()) {
            $this->telemetry->recordPageFailed(
                dataset: $dataset,
                mode: 'incremental',
                failureType: 'http_error',
                message: 'Bridge incremental HTTP error',
                httpStatus: $response->status(),
                bridgeUrl: $url,
                odataQuery: $query,
            );

            return ReplicationPageResult::httpError($url, $query, $response->status());
        }

        $body = $response->json();
        $value = is_array($body['value'] ?? null) ? $body['value'] : [];
        if ($value === []) {
            return new ReplicationPageResult(
                rows: [],
                nextReplicationUrl: null,
                replicationComplete: true,
                incrementalHasMore: false,
                nextIncrementalSkip: $skip,
                maxBridgeTs: null,
                bridgeUrl: $url,
                odataQuery: $query,
                httpStatus: $response->status(),
            );
        }

        $maxBridgeTs = $this->maxBridgeTimestampFromRows($value);
        $hasMore = count($value) >= $top;
        $nextSkip = $skip + $top;

        if ($hasMore && $nextSkip >= 10_000) {
            Log::warning('bridge.incremental.skip_cap', [
                'dataset' => $dataset,
                'message' => 'Exceeded 10k skip; use replication for catch-up per Bridge docs.',
            ]);
            $hasMore = false;
        }

        return new ReplicationPageResult(
            rows: $value,
            nextReplicationUrl: null,
            replicationComplete: true,
            incrementalHasMore: $hasMore,
            nextIncrementalSkip: $nextSkip,
            maxBridgeTs: $maxBridgeTs,
            bridgeUrl: $url,
            odataQuery: $query,
            httpStatus: $response->status(),
        );
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     */
    public function persistChunk(string $dataset, array $rows, ?ReplicationCursorPatch $patch = null): BridgeReplicaPersistStats
    {
        $stats = $rows === []
            ? new BridgeReplicaPersistStats
            : $this->mirrorWriter->hydrateReplicaBatch($dataset, $rows, ListingMirrorProvider::Bridge);

        if ($patch !== null) {
            $this->applyCursorPatch($dataset, $patch);
        }

        return $stats;
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     */
    private function maxBridgeTimestampFromRows(array $rows): ?CarbonImmutable
    {
        $maxTs = null;
        foreach ($rows as $row) {
            if (! is_array($row)) {
                continue;
            }
            $maxTs = $this->maxBridgeTimestamp($maxTs, $row);
        }

        return $maxTs;
    }

    /**
     * Revenue impact: replication seeds only IDX-facing inventory; Closed stays on-demand via Bridge API.
     */
    private function replicationActivePendingFilter(): string
    {
        return "(StandardStatus eq 'Active' or StandardStatus eq 'Pending')";
    }

    /**
     * @return array<string, int|string>
     */
    private function replicationMediaQueryParams(): array
    {
        return [];
    }

    private function replicationSelectList(string $dataset): string
    {
        $fields = explode(',', $this->syncSelectList($dataset));
        if (! in_array('Media', $fields, true)) {
            $fields[] = 'Media';
        }

        return implode(',', $fields);
    }

    /**
     * @return array<string, int|string>
     */
    private function mediaQueryParams(): array
    {
        if (config('bridge.sync_include_media', false)) {
            return [];
        }

        return ['$unselect' => 'Media'];
    }

    private function maxBridgeTimestamp(?CarbonImmutable $current, array $row): ?CarbonImmutable
    {
        $candidates = [];
        foreach (['ModificationTimestamp', 'BridgeModificationTimestamp'] as $key) {
            $raw = $row[$key] ?? null;
            if (! is_string($raw) || $raw === '') {
                continue;
            }
            try {
                $candidates[] = CarbonImmutable::parse($raw);
            } catch (\Throwable) {
                // ignore
            }
        }
        if ($candidates === []) {
            return $current;
        }
        $max = $candidates[0];
        foreach ($candidates as $c) {
            if ($c->gt($max)) {
                $max = $c;
            }
        }

        if ($current === null || $max->gt($current)) {
            return $max;
        }

        return $current;
    }

    private function extractNextUrl(Response $response): ?string
    {
        $link = $response->header('Link');
        if (is_string($link) && $link !== '') {
            foreach (explode(',', $link) as $segment) {
                if (preg_match('/<([^>]+)>\s*;\s*rel\s*=\s*"next"/i', trim($segment), $m)) {
                    return html_entity_decode($m[1], ENT_QUOTES);
                }
                if (preg_match("/<([^>]+)>\s*;\s*rel\s*=\s*'next'/i", trim($segment), $m)) {
                    return html_entity_decode($m[1], ENT_QUOTES);
                }
            }
        }

        $json = $response->json();
        if (is_array($json) && isset($json['@odata.nextLink']) && is_string($json['@odata.nextLink'])) {
            return $json['@odata.nextLink'];
        }

        return null;
    }

    private function buildPropertyCollectionUrl(string $dataset): string
    {
        return $this->buildODataPropertyBase($dataset).'/Property';
    }

    /**
     * Property replication feed (MLS-discretionary).
     */
    private function buildPropertyReplicationUrl(string $dataset): string
    {
        return $this->buildODataPropertyBase($dataset).'/Property/replication';
    }

    private function buildODataPropertyBase(string $dataset): string
    {
        $host = rtrim((string) config('bridge.host'), '/');
        $prefix = trim((string) config('bridge.path_prefix', ''), '/');
        $resoRoot = trim((string) config('bridge.reso_root', ''), '/');

        $basePath = $prefix !== ''
            ? "{$prefix}/OData/{$dataset}"
            : ($resoRoot !== ''
                ? "{$resoRoot}/OData/{$dataset}"
                : "OData/{$dataset}");

        return "{$host}/{$basePath}";
    }

    private function syncSelectList(string $dataset): string
    {
        $datasetUpper = strtoupper($dataset);

        $fields = array_unique(array_merge([
            'ListingKey',
            'ListingId',
            'BridgeModificationTimestamp',
            'ModificationTimestamp',
            'StandardStatus',
            'ListPrice',
            'ClosePrice',
            'PreviousListPrice',
            'PriceChangeTimestamp',
            'BedroomsTotal',
            'BathroomsTotalDecimal',
            'LivingArea',
            'LotSizeAcres',
            'YearBuilt',
            'StoriesTotal',
            'City',
            'CountyOrParish',
            'PostalCode',
            'StateOrProvince',
            'PropertyType',
            'PropertySubType',
            'OnMarketDate',
            'CloseDate',
            'Latitude',
            'Longitude',
            'Coordinates',
            'WaterfrontYN',
            'PoolPrivateYN',
            'NewConstructionYN',
            'GarageYN',
            'AssociationYN',
            'SpaYN',
            'FireplaceYN',
            'SeniorCommunityYN',
            'SpecialListingConditions',
            'SubdivisionName',
            'ElementarySchool',
            'MiddleOrJuniorSchool',
            'HighSchool',
            'StreetNumber',
            'StreetName',
            'ListAgentMlsId',
            'ListOfficeMlsId',
            'Media',
        ], [
            "{$datasetUpper}_FloodZoneCode",
            "{$datasetUpper}_TotalMonthlyFees",
        ]));

        if (! config('bridge.sync_include_media', false)) {
            $fields = array_values(array_filter($fields, static fn (string $field): bool => $field !== 'Media'));
        }

        // Stellar replication OData rejects DockYN in $select (HTTP 400).
        $fields = array_values(array_filter($fields, static fn (string $field): bool => $field !== 'DockYN'));

        return implode(',', $fields);
    }

    private function str(mixed $v): ?string
    {
        if (! is_string($v)) {
            return null;
        }
        $t = trim($v);

        return $t === '' ? null : $t;
    }

    private function num(mixed $v): ?float
    {
        return is_numeric($v) ? (float) $v : null;
    }

    private function intOrNull(mixed $v): ?int
    {
        return is_numeric($v) ? (int) $v : null;
    }

    private function boolOrNull(mixed $v): ?bool
    {
        return is_bool($v) ? $v : null;
    }

    private function dateStr(mixed $v): ?string
    {
        if (! is_string($v) || trim($v) === '') {
            return null;
        }
        try {
            return CarbonImmutable::parse($v)->format('Y-m-d');
        } catch (\Throwable) {
            return null;
        }
    }
}
