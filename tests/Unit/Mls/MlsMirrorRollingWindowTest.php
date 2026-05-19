<?php

namespace Tests\Unit\Mls;

use App\Services\Mls\MlsMirrorRollingWindow;
use Carbon\CarbonImmutable;
use Tests\TestCase;

class MlsMirrorRollingWindowTest extends TestCase
{
    public function test_months_reads_from_mls_config(): void
    {
        config(['mls.local_mirror_rolling_months' => 3]);

        $window = new MlsMirrorRollingWindow;

        $this->assertSame(3, $window->months());
    }

    public function test_months_clamps_to_config_bounds(): void
    {
        config(['mls.local_mirror_rolling_months' => 48]);

        $window = new MlsMirrorRollingWindow;

        $this->assertSame(48, $window->months());
    }

    public function test_cutoff_utc_subtracts_configured_months(): void
    {
        config(['mls.local_mirror_rolling_months' => 6]);
        CarbonImmutable::setTestNow('2026-05-19 12:00:00 UTC');

        $window = new MlsMirrorRollingWindow;
        $cutoff = $window->cutoffUtc();

        $this->assertSame('2025-11-19 12:00:00', $cutoff->format('Y-m-d H:i:s'));
        $this->assertSame('UTC', $cutoff->timezoneName);

        CarbonImmutable::setTestNow();
    }

    public function test_modification_timestamp_filter_iso_matches_cutoff(): void
    {
        config(['mls.local_mirror_rolling_months' => 12]);
        CarbonImmutable::setTestNow('2026-01-15 08:30:00 UTC');

        $window = new MlsMirrorRollingWindow;

        $this->assertSame(
            $window->cutoffUtc()->format('Y-m-d\TH:i:s\Z'),
            $window->modificationTimestampFilterIso(),
        );

        CarbonImmutable::setTestNow();
    }

    public function test_mls_local_mirror_env_wins_over_legacy_bridge_env(): void
    {
        putenv('MLS_LOCAL_MIRROR_ROLLING_MONTHS=3');
        putenv('BRIDGE_LOCAL_MIRROR_ROLLING_MONTHS=12');

        $this->refreshApplication();

        $this->assertSame(3, (int) config('mls.local_mirror_rolling_months'));
        $this->assertSame(3, (int) config('bridge.local_mirror_rolling_months'));

        putenv('MLS_LOCAL_MIRROR_ROLLING_MONTHS');
        putenv('BRIDGE_LOCAL_MIRROR_ROLLING_MONTHS');
    }
}
