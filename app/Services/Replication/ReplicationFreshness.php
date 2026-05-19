<?php

declare(strict_types=1);

namespace App\Services\Replication;

use App\Models\Listing;
use App\Models\ListingSyncCursor;
use Carbon\CarbonImmutable;

/**
 * Revenue impact: catch-up mode maximizes replication throughput until maps are within
 * freshness SLA; steady mode throttles polls to protect MLS egress margins.
 */
final class ReplicationFreshness
{
    public const MODE_CATCH_UP = 'catch_up';

    public const MODE_STEADY = 'steady';

    public function __construct(
        private readonly ReplicaPageStore $pageStore,
    ) {}

    public function mode(string $datasetSlug, string $provider): string
    {
        if ($this->isCurrent($datasetSlug, $provider)) {
            return self::MODE_STEADY;
        }

        return self::MODE_CATCH_UP;
    }

    public function isCurrent(string $datasetSlug, string $provider): bool
    {
        if ($this->pageStore->hasActivePage($datasetSlug, $provider)) {
            return false;
        }

        $cursor = ListingSyncCursor::query()->find($datasetSlug);
        if ($cursor === null) {
            return false;
        }

        if ($cursor->replication_in_progress) {
            return false;
        }

        if ($cursor->replication_next_url !== null && $cursor->replication_next_url !== '') {
            return false;
        }

        $mirrorSeeded = $cursor->last_sync_finished_at !== null
            || Listing::query()->where('dataset_slug', $datasetSlug)->exists();

        if (! $mirrorSeeded) {
            return false;
        }

        $lastTs = $cursor->last_bridge_modification_timestamp;
        if ($lastTs === null) {
            return false;
        }

        $threshold = max(1, (int) config('mls.freshness_threshold_minutes', 15));

        return CarbonImmutable::parse($lastTs->format(\DateTimeInterface::ATOM))
            ->gte(now()->subMinutes($threshold));
    }

    public function minutesBehindMls(string $datasetSlug): ?int
    {
        $cursor = ListingSyncCursor::query()->find($datasetSlug);
        if ($cursor === null || $cursor->last_bridge_modification_timestamp === null) {
            return null;
        }

        $last = CarbonImmutable::parse($cursor->last_bridge_modification_timestamp->format(\DateTimeInterface::ATOM));

        return (int) max(0, $last->diffInMinutes(now()));
    }

    public function freshnessThresholdMinutes(): int
    {
        return max(1, (int) config('mls.freshness_threshold_minutes', 15));
    }

    public function steadyPollMinutes(): int
    {
        return max(1, (int) config('mls.steady_incremental_poll_minutes', 15));
    }
}
