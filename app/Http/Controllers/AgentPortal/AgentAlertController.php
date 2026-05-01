<?php

namespace App\Http\Controllers\AgentPortal;

use App\Http\Controllers\Controller;
use App\Http\Requests\AgentPortal\AgentAlertFromTemplateRequest;
use App\Http\Requests\AgentPortal\AgentAlertUpsertRequest;
use App\Models\AgentAlert;
use App\Models\AgentAlertRun;
use App\Models\AgentAlertTemplate;
use App\Models\User;
use App\Services\AgentPortal\AgentActivityTracker;
use App\Services\AgentPortal\AlertSchedulerService;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use Carbon\CarbonImmutable;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AgentAlertController extends Controller
{
    private const RECENT_RUNS_INDEX_LIMIT = 8;

    private const RECENT_RUNS_SHOW_LIMIT = 15;

    public function index(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $type = trim((string) $request->query('alert_type', ''));
        $allowedTypes = ['listing', 'market_activity', 'home_value'];
        if ($type !== '' && ! in_array($type, $allowedTypes, true)) {
            return response()->json([
                'message' => 'Invalid alert_type filter.',
                'errors' => [
                    'alert_type' => ['Must be one of: listing, market_activity, home_value.'],
                ],
            ], 422);
        }

        $rows = AgentAlert::query()
            ->where('user_id', $user->id)
            ->when($type !== '', static fn (Builder $query) => $query->where('alert_type', $type))
            ->withCount('runs')
            ->with([
                'agentSearch',
                'runs' => static function ($query): void {
                    $query->orderByDesc('ran_at')->orderByDesc('id')->limit(self::RECENT_RUNS_INDEX_LIMIT);
                },
            ])
            ->latest('id')
            ->get();

        foreach ($rows as $alert) {
            $this->sanitizeRunsMetadata($alert);
        }

        return response()->json(['data' => $rows]);
    }

    public function runs(Request $request, int $alertId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        AgentAlert::query()
            ->where('id', $alertId)
            ->where('user_id', $user->id)
            ->firstOrFail();

        $limit = max(1, min(50, (int) $request->query('limit', 20)));

        $runs = AgentAlertRun::query()
            ->where('agent_alert_id', $alertId)
            ->orderByDesc('ran_at')
            ->orderByDesc('id')
            ->limit($limit)
            ->get(['id', 'status', 'metadata_json', 'ran_at', 'created_at']);

        $data = $runs->map(static function (AgentAlertRun $run): array {
            $meta = is_array($run->metadata_json) ? $run->metadata_json : [];

            return [
                'id' => $run->id,
                'status' => (string) $run->status,
                'ran_at' => $run->ran_at?->toIso8601String(),
                'created_at' => $run->created_at?->toIso8601String(),
                'metadata' => self::sanitizeAlertRunMetadataForClient($meta),
            ];
        })->values()->all();

        return response()->json(['data' => $data]);
    }

    public function show(Request $request, int $alertId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $row = AgentAlert::query()
            ->where('id', $alertId)
            ->where('user_id', $user->id)
            ->withCount('runs')
            ->with([
                'agentSearch',
                'runs' => static function ($query): void {
                    $query->orderByDesc('ran_at')->orderByDesc('id')->limit(self::RECENT_RUNS_SHOW_LIMIT);
                },
            ])
            ->firstOrFail();

        $this->sanitizeRunsMetadata($row);

        return response()->json(['data' => $row]);
    }

    public function summary(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $alertsQuery = AgentAlert::query()->where('user_id', $user->id);
        $alerts = $alertsQuery->get();
        $runsLast30Days = AgentAlertRun::query()
            ->whereHas('agentAlert', static function (Builder $query) use ($user): void {
                $query->where('user_id', $user->id);
            })
            ->where('ran_at', '>=', now()->subDays(30))
            ->count();

        return response()->json([
            'data' => [
                'total' => $alerts->count(),
                'active' => $alerts->where('status', 'active')->count(),
                'paused' => $alerts->where('status', 'paused')->count(),
                'by_type' => [
                    'listing' => $alerts->where('alert_type', 'listing')->count(),
                    'market_activity' => $alerts->where('alert_type', 'market_activity')->count(),
                    'home_value' => $alerts->where('alert_type', 'home_value')->count(),
                ],
                'runs_last_30_days' => $runsLast30Days,
            ],
        ]);
    }

    public function history(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $days = max(1, min(90, (int) $request->query('days', 30)));
        $today = CarbonImmutable::now()->startOfDay();
        $start = $today->subDays($days - 1);

        $runs = AgentAlertRun::query()
            ->whereHas('agentAlert', static function (Builder $query) use ($user): void {
                $query->where('user_id', $user->id);
            })
            ->whereNotNull('ran_at')
            ->where('ran_at', '>=', $start)
            ->with(['agentAlert:id,alert_type'])
            ->get(['agent_alert_id', 'ran_at']);

        $buckets = [];
        for ($i = 0; $i < $days; $i++) {
            $date = $start->addDays($i)->toDateString();
            $buckets[$date] = [
                'date' => $date,
                'total_runs' => 0,
                'listing_runs' => 0,
                'market_activity_runs' => 0,
                'home_value_runs' => 0,
            ];
        }

        foreach ($runs as $run) {
            $date = $run->ran_at?->toDateString();
            if ($date === null || ! isset($buckets[$date])) {
                continue;
            }
            $buckets[$date]['total_runs']++;
            $alertType = (string) ($run->agentAlert?->alert_type ?? '');
            if ($alertType === 'listing') {
                $buckets[$date]['listing_runs']++;
            } elseif ($alertType === 'market_activity') {
                $buckets[$date]['market_activity_runs']++;
            } elseif ($alertType === 'home_value') {
                $buckets[$date]['home_value_runs']++;
            }
        }

        $totalRuns = array_sum(array_map(static fn (array $bucket): int => $bucket['total_runs'], $buckets));

        return response()->json([
            'data' => [
                'buckets' => array_values($buckets),
                'meta' => [
                    'days' => $days,
                    'total_runs' => $totalRuns,
                ],
            ],
        ]);
    }

    public function store(AgentAlertUpsertRequest $request, SubscriberFeedAccessService $feeds): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        if (($blocked = $this->rejectIfNoAlertFeeds($user, $feeds)) !== null) {
            return $blocked;
        }

        $validated = $request->validated();
        $scheduleJson = $validated['schedule_json'] ?? null;
        $row = AgentAlert::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $validated['agent_search_id'] ?? null,
            'name' => (string) $validated['name'],
            'alert_type' => (string) $validated['alert_type'],
            'status' => (string) ($validated['status'] ?? 'active'),
            'schedule_json' => $scheduleJson,
            'next_run_at' => $validated['next_run_at'] ?? null,
        ]);

        if ($row->next_run_at === null && $row->status === 'active') {
            $scheduler = app(AlertSchedulerService::class);
            $scheduler->setInitialNextRunAt($row, is_array($scheduleJson) ? $scheduleJson : null);
        }

        app(AgentActivityTracker::class)->record(
            $user,
            'create_alert',
            "Created alert: {$row->name}",
        );

        $row->loadCount('runs');
        $row->load([
            'agentSearch',
            'runs' => static function ($query): void {
                $query->orderByDesc('ran_at')->orderByDesc('id')->limit(self::RECENT_RUNS_SHOW_LIMIT);
            },
        ]);
        $this->sanitizeRunsMetadata($row);

        return response()->json(['data' => $row], 201);
    }

    public function update(AgentAlertUpsertRequest $request, int $alertId, SubscriberFeedAccessService $feeds): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $row = AgentAlert::query()
            ->where('id', $alertId)
            ->where('user_id', $user->id)
            ->firstOrFail();

        $validated = $request->validated();
        $nextStatus = (string) ($validated['status'] ?? $row->status);
        if ($nextStatus === 'active' && ($blocked = $this->rejectIfNoAlertFeeds($user, $feeds)) !== null) {
            return $blocked;
        }

        $row->update([
            'agent_search_id' => $validated['agent_search_id'] ?? null,
            'name' => (string) $validated['name'],
            'alert_type' => (string) $validated['alert_type'],
            'status' => (string) ($validated['status'] ?? 'active'),
            'schedule_json' => $validated['schedule_json'] ?? null,
            'next_run_at' => $validated['next_run_at'] ?? null,
        ]);

        $row->loadCount('runs');
        $row->load([
            'agentSearch',
            'runs' => static function ($query): void {
                $query->orderByDesc('ran_at')->orderByDesc('id')->limit(self::RECENT_RUNS_SHOW_LIMIT);
            },
        ]);
        $this->sanitizeRunsMetadata($row);

        return response()->json(['data' => $row]);
    }

    public function destroy(Request $request, int $alertId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        AgentAlert::query()
            ->where('id', $alertId)
            ->where('user_id', $user->id)
            ->firstOrFail();
        AgentAlert::query()
            ->where('id', $alertId)
            ->where('user_id', $user->id)
            ->delete();

        return response()->json([], 204);
    }

    public function fromTemplate(AgentAlertFromTemplateRequest $request, SubscriberFeedAccessService $feeds): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        if (($blocked = $this->rejectIfNoAlertFeeds($user, $feeds)) !== null) {
            return $blocked;
        }

        $validated = $request->validated();
        $template = AgentAlertTemplate::query()
            ->where('id', (int) $validated['template_id'])
            ->where('user_id', $user->id)
            ->firstOrFail();

        $templateBody = is_array($template->body_json) ? $template->body_json : [];
        $schedule = is_array($template->schedule_json) && $template->schedule_json !== []
            ? $template->schedule_json
            : (isset($templateBody['schedule']) && is_array($templateBody['schedule']) ? $templateBody['schedule'] : null);
        $status = is_string($templateBody['status'] ?? null)
            ? (string) $templateBody['status']
            : 'active';

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $validated['agent_search_id'] ?? null,
            'name' => (string) $validated['name'],
            'alert_type' => (string) $template->template_type,
            'status' => $status,
            'schedule_json' => $schedule,
            'next_run_at' => null,
        ]);

        AgentAlertTemplate::query()->where('id', $template->id)->increment('usage_count', 1);
        $template->update(['last_used_at' => now()]);

        $alert->loadCount('runs');
        $alert->load([
            'agentSearch',
            'runs' => static function ($query): void {
                $query->orderByDesc('ran_at')->orderByDesc('id')->limit(self::RECENT_RUNS_SHOW_LIMIT);
            },
        ]);
        $this->sanitizeRunsMetadata($alert);

        return response()->json(['data' => $alert], 201);
    }

    /**
     * @return JsonResponse|null HTTP 422 when the subscriber has no feeds permitted for IDX alerts.
     */
    private function rejectIfNoAlertFeeds(User $user, SubscriberFeedAccessService $feeds): ?JsonResponse
    {
        if ($feeds->resolvedAlertScopesForUser($user) !== []) {
            return null;
        }

        return response()->json([
            'message' => 'IDX alerts are not enabled for your MLS feeds.',
            'errors' => [
                'feed_access' => ['No feeds are enabled for IDX alerts.'],
            ],
        ], 422);
    }

    private function sanitizeRunsMetadata(AgentAlert $alert): void
    {
        foreach ($alert->runs as $run) {
            $run->setAttribute(
                'metadata_json',
                self::sanitizeAlertRunMetadataForClient(is_array($run->metadata_json) ? $run->metadata_json : [])
            );
        }
    }

    /**
     * @param  array<string, mixed>  $metadata
     * @return array<string, mixed>
     */
    private static function sanitizeAlertRunMetadataForClient(array $metadata): array
    {
        if (! isset($metadata['listing_query']) || ! is_array($metadata['listing_query'])) {
            return $metadata;
        }

        $listingQuery = $metadata['listing_query'];
        unset($listingQuery['listing_ids_snapshot']);
        $metadata['listing_query'] = $listingQuery;

        return $metadata;
    }
}
