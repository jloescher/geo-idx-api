<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Listing;
use App\Models\ListingSyncCursor;
use App\Services\Bridge\MlsDatasetResolver;
use Illuminate\Http\JsonResponse;

/**
 * Revenue impact: operators can verify replica freshness — stale mirrors push traffic to Bridge OData ($).
 *
 * Compliance: aggregates contain no PHI; fields stay within IDX policy.
 */
class BridgeReplicationStatsController extends Controller
{
    public function __construct(
        private readonly MlsDatasetResolver $datasets,
    ) {}

    public function __invoke(): JsonResponse
    {
        $catalog = $this->datasets->getAvailableDatasets();
        $datasets = [];

        foreach ($catalog as $slug) {
            $cursor = ListingSyncCursor::query()->find($slug);

            $datasets[] = [
                'slug' => $slug,
                'listing_count_total' => Listing::query()->where('dataset_slug', $slug)->count(),
                'listing_count_active_pending' => Listing::query()
                    ->where('dataset_slug', $slug)
                    ->where(function ($q): void {
                        $q->whereRaw('LOWER(COALESCE(standard_status, \'\')) = ?', ['active'])
                            ->orWhereRaw('LOWER(COALESCE(standard_status, \'\')) = ?', ['pending']);
                    })
                    ->count(),
                'last_bridge_modification_timestamp' => $cursor?->last_bridge_modification_timestamp?->toAtomString(),
                'last_sync_finished_at' => $cursor?->last_sync_finished_at?->toAtomString(),
                'replication_in_progress' => (bool) ($cursor?->replication_in_progress ?? false),
            ];
        }

        return response()->json(['datasets' => $datasets]);
    }
}
