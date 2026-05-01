<?php

namespace App\Services\AgentPortal;

use App\Models\SubscriberFeedAccess;
use App\Models\User;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;

final class SubscriberFeedAccessService
{
    public const DEFAULT_FEED_ID = 'bridge';

    /**
     * Sync persisted feed rows from the user's assigned MLS datasets (and active domains when present).
     */
    public function syncForUser(User $user): void
    {
        $existingRows = SubscriberFeedAccess::query()
            ->where('user_id', $user->id)
            ->get()
            ->keyBy(fn (SubscriberFeedAccess $r): string => (string) $r->feed_id.'|'.(string) $r->dataset_code);

        $byKey = [];

        foreach ($user->assignedDatasets() as $datasetCode) {
            $datasetCode = trim((string) $datasetCode);
            if ($datasetCode === '') {
                continue;
            }
            $key = self::DEFAULT_FEED_ID.'|'.$datasetCode;
            $byKey[$key] = [
                'user_id' => $user->id,
                'mls_code' => $datasetCode,
                'feed_id' => self::DEFAULT_FEED_ID,
                'dataset_code' => $datasetCode,
                'status' => 'active',
                'permissions_json' => ['idx:search' => true, 'idx:alerts' => true],
                'connected_at' => now(),
                'last_verified_at' => now(),
            ];
        }

        foreach ($user->domains()->where('is_active', true)->cursor() as $domain) {
            $ds = $domain->getMlsDataset();
            if (! is_string($ds) || trim($ds) === '') {
                continue;
            }
            $code = trim($ds);
            $key = self::DEFAULT_FEED_ID.'|'.$code;
            $byKey[$key] = [
                'user_id' => $user->id,
                'mls_code' => $code,
                'feed_id' => self::DEFAULT_FEED_ID,
                'dataset_code' => $code,
                'status' => 'active',
                'permissions_json' => ['idx:search' => true, 'idx:alerts' => true],
                'connected_at' => now(),
                'last_verified_at' => now(),
            ];
        }

        foreach ($byKey as $key => &$payload) {
            $existing = $existingRows->get($key);
            if ($existing === null) {
                continue;
            }
            $prev = $existing->permissions_json;
            if (! is_array($prev) || $prev === []) {
                continue;
            }
            $defaults = is_array($payload['permissions_json'] ?? null) ? $payload['permissions_json'] : [];
            $payload['permissions_json'] = array_merge($defaults, $prev);
        }
        unset($payload);

        DB::transaction(function () use ($user, $byKey): void {
            SubscriberFeedAccess::query()->where('user_id', $user->id)->delete();
            foreach ($byKey as $payload) {
                SubscriberFeedAccess::query()->create($payload);
            }
        });
    }

    /**
     * @return list<array{mls_code: string, dataset_code: string, feed_id: string, status: string}>
     */
    public function resolvedScopesForUser(User $user): array
    {
        return $this->mapActiveScopesToPayload($this->syncedActiveFeedRows($user));
    }

    /**
     * Active feed rows the subscriber may use for MLS search execution and lookup fan-out.
     *
     * @return list<array{mls_code: string, dataset_code: string, feed_id: string, status: string}>
     */
    public function resolvedSearchScopesForUser(User $user): array
    {
        return $this->mapActiveScopesToPayload(
            $this->syncedActiveFeedRows($user)
                ->filter(fn (SubscriberFeedAccess $r): bool => self::permissionsAllowIdxSearch($r->permissions_json))
        );
    }

    /**
     * Active feed rows the subscriber may use for IDX-backed alert creation and scheduling.
     *
     * @return list<array{mls_code: string, dataset_code: string, feed_id: string, status: string}>
     */
    public function resolvedAlertScopesForUser(User $user): array
    {
        return $this->mapActiveScopesToPayload(
            $this->syncedActiveFeedRows($user)
                ->filter(fn (SubscriberFeedAccess $r): bool => self::permissionsAllowIdxAlerts($r->permissions_json))
        );
    }

    /**
     * Full feed rows for dashboards and settings, including permission flags for UI gating.
     *
     * @return list<array{mls_code: string, dataset_code: string, feed_id: string, status: string, permissions_json: array<string, mixed>|null, can_search: bool, can_alerts: bool}>
     */
    public function detailedFeedScopesForUser(User $user): array
    {
        return $this->syncedActiveFeedRows($user)
            ->map(function (SubscriberFeedAccess $r): array {
                $permissions = is_array($r->permissions_json) ? $r->permissions_json : null;

                return [
                    'mls_code' => (string) $r->mls_code,
                    'dataset_code' => (string) $r->dataset_code,
                    'feed_id' => (string) $r->feed_id,
                    'status' => (string) $r->status,
                    'permissions_json' => $permissions,
                    'can_search' => self::permissionsAllowIdxSearch($permissions),
                    'can_alerts' => self::permissionsAllowIdxAlerts($permissions),
                ];
            })
            ->values()
            ->all();
    }

    /**
     * @param  array<string, mixed>|null  $permissionsJson
     */
    public static function permissionsAllowIdxSearch(?array $permissionsJson): bool
    {
        if ($permissionsJson === null) {
            return true;
        }

        return ($permissionsJson['idx:search'] ?? true) !== false;
    }

    /**
     * @param  array<string, mixed>|null  $permissionsJson
     */
    public static function permissionsAllowIdxAlerts(?array $permissionsJson): bool
    {
        if ($permissionsJson === null) {
            return true;
        }

        return ($permissionsJson['idx:alerts'] ?? true) !== false;
    }

    /**
     * @return Collection<int, SubscriberFeedAccess>
     */
    private function syncedActiveFeedRows(User $user): Collection
    {
        $this->syncForUser($user);

        return SubscriberFeedAccess::query()
            ->where('user_id', $user->id)
            ->where('status', 'active')
            ->orderBy('dataset_code')
            ->get();
    }

    /**
     * @param  Collection<int, SubscriberFeedAccess>  $rows
     * @return list<array{mls_code: string, dataset_code: string, feed_id: string, status: string}>
     */
    private function mapActiveScopesToPayload(Collection $rows): array
    {
        return $rows
            ->map(fn (SubscriberFeedAccess $r): array => [
                'mls_code' => (string) $r->mls_code,
                'dataset_code' => (string) $r->dataset_code,
                'feed_id' => (string) $r->feed_id,
                'status' => (string) $r->status,
            ])
            ->values()
            ->all();
    }
}
