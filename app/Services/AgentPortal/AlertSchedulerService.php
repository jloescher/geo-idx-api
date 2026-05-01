<?php

namespace App\Services\AgentPortal;

use App\Models\AgentAlert;
use App\Models\AgentAlertRun;
use App\Models\AgentSearch;
use App\Models\User;
use Carbon\CarbonImmutable;

final class AlertSchedulerService
{
    /**
     * Compute the next run time based on cadence and optional schedule config.
     *
     * @param  array<string, mixed>|null  $scheduleJson
     */
    public function computeNextRunAt(string $cadence, ?array $scheduleJson = null): CarbonImmutable
    {
        $now = CarbonImmutable::now();
        $timeOfDay = (string) ($scheduleJson['time_of_day'] ?? '08:00');
        $dayOfWeek = isset($scheduleJson['day_of_week']) ? (int) $scheduleJson['day_of_week'] : 1;

        $parts = explode(':', $timeOfDay);
        $hour = isset($parts[0]) ? (int) $parts[0] : 8;
        $minute = isset($parts[1]) ? (int) $parts[1] : 0;

        return match ($cadence) {
            'immediate' => $now->addMinutes(5),
            'daily' => $now->setTime($hour, $minute, 0)->isAfter($now)
                ? $now->setTime($hour, $minute, 0)
                : $now->addDay()->setTime($hour, $minute, 0),
            'weekly' => $this->nextDayOfWeek($now, $dayOfWeek, $hour, $minute),
            'monthly' => $now->addMonthNoOverflow()->startOfMonth()->setTime($hour, $minute, 0),
            default => $now->addDay()->setTime($hour, $minute, 0),
        };
    }

    /**
     * Check whether an alert is within its cooldown period.
     */
    public function isWithinCooldown(AgentAlert $alert, int $cooldownMinutes = 60): bool
    {
        $lastRun = $alert->runs()->latest('ran_at')->first();
        if ($lastRun === null) {
            return false;
        }

        return $lastRun->ran_at !== null && $lastRun->ran_at->gt(now()->subMinutes($cooldownMinutes));
    }

    /**
     * Process all alerts that are due to run.
     *
     * @return int Number of alerts processed
     */
    public function processDueAlerts(): int
    {
        $due = AgentAlert::query()
            ->where('status', 'active')
            ->where(fn ($q) => $q->whereNull('next_run_at')->orWhere('next_run_at', '<=', now()))
            ->with('user')
            ->get();

        $feeds = app(SubscriberFeedAccessService::class);
        $count = 0;
        foreach ($due as $alert) {
            $owner = $alert->user;
            if (! $owner instanceof User || $feeds->resolvedAlertScopesForUser($owner) === []) {
                continue;
            }

            if ($this->isWithinCooldown($alert)) {
                continue;
            }

            $schedule = is_array($alert->schedule_json) ? $alert->schedule_json : null;
            $cadence = (string) ($schedule['cadence'] ?? 'daily');

            $metadata = [
                'triggered_by' => 'scheduler',
                'cadence' => $cadence,
            ];

            if ($alert->alert_type === 'listing' && $alert->agent_search_id !== null) {
                $search = AgentSearch::query()
                    ->where('id', $alert->agent_search_id)
                    ->where('user_id', $owner->id)
                    ->first();
                if ($search === null) {
                    $metadata['listing_query'] = [
                        'success' => false,
                        'reason' => 'search_not_found',
                    ];
                } else {
                    $exec = app(AgentListingAlertSearchExecutor::class)->executeForOwner($owner, $search);
                    $listingQuery = $this->summarizeListingAlertExecution($exec);
                    if (($listingQuery['success'] ?? false) === true) {
                        $listingQuery = $this->appendNewListingMatches($alert, $listingQuery);
                    }
                    $metadata['listing_query'] = $listingQuery;
                }
            }

            AgentAlertRun::query()->create([
                'agent_alert_id' => $alert->id,
                'status' => 'sent',
                'metadata_json' => $metadata,
                'ran_at' => now(),
            ]);

            $alert->update([
                'next_run_at' => $this->computeNextRunAt($cadence, $schedule),
            ]);

            $count++;
        }

        return $count;
    }

    /**
     * Set the initial next_run_at for a newly created alert.
     *
     * @param  array<string, mixed>|null  $scheduleJson
     */
    public function setInitialNextRunAt(AgentAlert $alert, ?array $scheduleJson = null): void
    {
        $cadence = (string) ($scheduleJson['cadence'] ?? 'daily');
        $alert->update([
            'next_run_at' => $this->computeNextRunAt($cadence, $scheduleJson),
        ]);
    }

    private function nextDayOfWeek(CarbonImmutable $now, int $dayOfWeek, int $hour, int $minute): CarbonImmutable
    {
        $target = $now->startOfDay()->setTime($hour, $minute, 0);
        while ($target->dayOfWeekIso !== $dayOfWeek || $target->isAfter($now)) {
            if ($target->isAfter($now)) {
                break;
            }
            $target = $target->addDay();
        }

        if ($target->isBefore($now) || $target->dayOfWeekIso !== $dayOfWeek) {
            $target = $now->next($dayOfWeek)->setTime($hour, $minute, 0);
        }

        return $target;
    }

    /**
     * @param  array<string, mixed>  $exec
     * @return array<string, mixed>
     */
    private function summarizeListingAlertExecution(array $exec): array
    {
        if (($exec['success'] ?? false) === true && isset($exec['merged']) && is_array($exec['merged'])) {
            /** @var array<string, mixed> $merged */
            $merged = $exec['merged'];
            $items = (array) ($merged['items'] ?? []);
            $meta = is_array($merged['meta'] ?? null) ? $merged['meta'] : [];

            $allListingIds = [];
            foreach ($items as $item) {
                if (! is_array($item)) {
                    continue;
                }
                if (! isset($item['listingId'])) {
                    continue;
                }
                $lid = trim((string) $item['listingId']);
                if ($lid === '') {
                    continue;
                }
                $allListingIds[] = $lid;
            }
            $allListingIds = array_values(array_unique($allListingIds));

            return [
                'success' => true,
                'total_items' => (int) ($meta['total_items'] ?? count($items)),
                'sources' => (int) ($meta['sources'] ?? 0),
                'sample_listing_ids' => array_values(array_filter(array_map(
                    static function (array $item): ?string {
                        return isset($item['listingId']) ? (string) $item['listingId'] : null;
                    },
                    array_slice($items, 0, 12)
                ))),
                'listing_ids_snapshot' => array_slice($allListingIds, 0, 250),
            ];
        }

        $row = [
            'success' => false,
            'reason' => (string) ($exec['reason'] ?? 'unknown'),
        ];
        if (isset($exec['errors']) && is_array($exec['errors'])) {
            $row['error_keys'] = array_keys($exec['errors']);
        }

        return $row;
    }

    /**
     * Compare current listing IDs to the most recent prior run for this alert (stored before this insert).
     *
     * @param  array<string, mixed>  $listingQuery
     * @return array<string, mixed>
     */
    private function appendNewListingMatches(AgentAlert $alert, array $listingQuery): array
    {
        $current = is_array($listingQuery['listing_ids_snapshot'] ?? null)
            ? $listingQuery['listing_ids_snapshot']
            : [];
        $current = array_values(array_unique(array_map(
            static fn (mixed $id): string => trim((string) $id),
            $current
        )));
        $current = array_values(array_filter($current, static fn (string $id): bool => $id !== ''));

        $prevRun = AgentAlertRun::query()
            ->where('agent_alert_id', $alert->id)
            ->whereNotNull('ran_at')
            ->orderByDesc('ran_at')
            ->first();

        $prevIds = [];
        if ($prevRun !== null) {
            $pmeta = is_array($prevRun->metadata_json) ? $prevRun->metadata_json : [];
            $plq = is_array($pmeta['listing_query'] ?? null) ? $pmeta['listing_query'] : [];
            if (is_array($plq['listing_ids_snapshot'] ?? null) && $plq['listing_ids_snapshot'] !== []) {
                $prevIds = $plq['listing_ids_snapshot'];
            } elseif (is_array($plq['sample_listing_ids'] ?? null)) {
                $prevIds = $plq['sample_listing_ids'];
            }
            $prevIds = array_values(array_unique(array_map(
                static fn (mixed $id): string => trim((string) $id),
                $prevIds
            )));
            $prevIds = array_values(array_filter($prevIds, static fn (string $id): bool => $id !== ''));
        }

        $prevSet = array_fill_keys($prevIds, true);
        $newIds = [];
        foreach ($current as $id) {
            if (! isset($prevSet[$id])) {
                $newIds[] = $id;
            }
        }

        $listingQuery['new_match_count'] = count($newIds);
        $listingQuery['new_listing_ids'] = array_slice($newIds, 0, 24);

        return $listingQuery;
    }
}
