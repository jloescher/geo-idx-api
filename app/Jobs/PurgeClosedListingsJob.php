<?php

namespace App\Jobs;

use App\Models\Listing;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;

/**
 * Revenue impact: evicting Closed + stale Closed rows trims storage footprint and keeps teaser
 * risk surfaces aligned with IDX display scope.
 *
 * Compliance: data retention aligns with IDX policy; authoritative history remains on MLS via Bridge fallback.
 */
class PurgeClosedListingsJob implements ShouldQueue
{
    use Queueable;

    public function handle(): void
    {
        /*
         * Revenue impact: trims cold rows so Postgres buffer cache stays biased toward monetized map views;
         * rolling window aligns with BRIN / partial indexes on Active+Pending ingestion paths.
         */
        $months = max(1, min(48, (int) config('bridge.local_mirror_rolling_months', 12)));
        $cutoff = now()->subMonths($months)->startOfDay();

        Listing::query()
            ->where(function ($q) use ($cutoff): void {
                $q->whereRaw('LOWER(TRIM(COALESCE(standard_status, \'\'))) = ?', ['closed'])
                    ->orWhere(function ($q2) use ($cutoff): void {
                        $q2->whereNotNull('close_date')
                            ->where('close_date', '<', $cutoff->toDateString());
                    })
                    ->orWhere(function ($q3) use ($cutoff): void {
                        $q3->whereNotNull('modification_timestamp')
                            ->where('modification_timestamp', '<', $cutoff);
                    });
            })
            ->delete();
    }
}
