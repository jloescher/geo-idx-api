<?php

namespace App\Services\Mls;

use Carbon\CarbonImmutable;

/**
 * Revenue impact: one rolling window for mirror purge, PostGIS search, and listings-cache OData
 * so staging (e.g. 3 months) and production (12 months) differ only by MLS_LOCAL_MIRROR_ROLLING_MONTHS.
 */
final class MlsMirrorRollingWindow
{
    public function months(): int
    {
        return (int) config('mls.local_mirror_rolling_months', 12);
    }

    public function cutoffUtc(): CarbonImmutable
    {
        return CarbonImmutable::now('UTC')->subMonths($this->months());
    }

    public function modificationTimestampFilterIso(): string
    {
        return $this->cutoffUtc()->format('Y-m-d\TH:i:s\Z');
    }
}
