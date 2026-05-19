<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Listing;
use App\Models\ListingSyncCursor;
use App\Services\Bridge\MlsDatasetResolver;
use App\Services\Mls\MlsDatasetRegistry;
use App\Services\Mls\MlsFeedResolver;
use App\Services\Replication\ReplicationFreshness;
use Illuminate\Http\JsonResponse;

/**
 * Revenue impact: operators can verify replica freshness — stale mirrors push traffic to live MLS OData ($).
 *
 * Compliance: aggregates contain no PHI; fields stay within IDX policy.
 */
class BridgeReplicationStatsController extends Controller
{
    public function __construct(
        private readonly MlsDatasetResolver $datasets,
        private readonly MlsFeedResolver $feeds,
        private readonly ReplicationFreshness $freshness,
        private readonly MlsDatasetRegistry $registry,
    ) {}

    public function __invoke(): JsonResponse
    {
        $catalog = $this->datasets->getAvailableDatasets();
        $datasets = [];

        foreach ($catalog as $feedCode) {
            $mirrorSlug = $this->feeds->mirrorDatasetSlug($feedCode);
            $cursor = ListingSyncCursor::query()->find($mirrorSlug);

            $provider = $this->registry->provider($mirrorSlug);

            $datasets[] = [
                'feed' => $feedCode,
                'slug' => $mirrorSlug,
                'provider' => $provider,
                'replication_mode' => $this->freshness->mode($mirrorSlug, $provider),
                'freshness_threshold_minutes' => $this->freshness->freshnessThresholdMinutes(),
                'minutes_behind_mls' => $this->freshness->minutesBehindMls($mirrorSlug),
                'listing_count_total' => Listing::query()->where('dataset_slug', $mirrorSlug)->count(),
                'listing_count_active_pending' => Listing::query()
                    ->where('dataset_slug', $mirrorSlug)
                    ->where(function ($q): void {
                        $q->whereRaw('LOWER(COALESCE(standard_status, \'\')) = ?', ['active'])
                            ->orWhereRaw('LOWER(COALESCE(standard_status, \'\')) = ?', ['pending']);
                    })
                    ->count(),
                'last_bridge_modification_timestamp' => $cursor?->last_bridge_modification_timestamp?->toAtomString(),
                'incremental_window_end' => $cursor?->incremental_window_end?->toAtomString(),
                'last_sync_finished_at' => $cursor?->last_sync_finished_at?->toAtomString(),
                'replication_in_progress' => (bool) ($cursor?->replication_in_progress ?? false),
            ];
        }

        return response()->json(['datasets' => $datasets]);
    }
}
