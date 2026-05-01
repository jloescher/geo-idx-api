<?php

namespace App\Http\Controllers\AgentPortal;

use App\Ghl\Sync\Models\QuantyraLead;
use App\Http\Controllers\Controller;
use App\Models\AgentActivityEvent;
use App\Models\AgentAlert;
use App\Models\AgentAlertRun;
use App\Models\AgentSearch;
use App\Models\AgentShareLink;
use App\Models\User;
use App\Services\SubscriberLeadScopeService;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AgentDashboardSummaryController extends Controller
{
    public function __invoke(Request $request, SubscriberLeadScopeService $scope): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $periodDays = max(1, min(365, (int) ($request->query('period', 7))));
        $since = now()->subDays($periodDays);

        $locationIds = $scope->locationIdsForUser($user);

        $leadCount = QuantyraLead::query()
            ->whereIn('ghl_location_id', $locationIds)
            ->when($periodDays <= 90, fn ($q) => $q->where('created_at', '>=', $since))
            ->count();

        $activeAlertCount = AgentAlert::query()
            ->where('user_id', $user->id)
            ->where('status', 'active')
            ->count();

        $savedSearchCount = AgentSearch::query()
            ->where('user_id', $user->id)
            ->when($periodDays <= 90, fn ($q) => $q->where('created_at', '>=', $since))
            ->count();

        $activeShareLinkCount = AgentShareLink::query()
            ->where('user_id', $user->id)
            ->whereNull('expires_at')
            ->count();

        $leadActivity = QuantyraLead::query()
            ->whereIn('ghl_location_id', $locationIds)
            ->where('created_at', '>=', $since)
            ->latest('id')
            ->limit(8)
            ->get()
            ->map(static function (QuantyraLead $lead): array {
                $payload = is_array($lead->payload) ? $lead->payload : [];

                return [
                    'type' => 'lead',
                    'title' => (string) ($payload['name'] ?? 'Lead #'.$lead->id),
                    'status' => (string) ($payload['status'] ?? 'new'),
                    'at' => optional($lead->created_at)->toIso8601String(),
                ];
            })
            ->values()
            ->all();

        $alertActivity = AgentAlert::query()
            ->where('user_id', $user->id)
            ->where('created_at', '>=', $since)
            ->latest('id')
            ->limit(8)
            ->get()
            ->map(static fn (AgentAlert $alert): array => [
                'type' => 'alert',
                'title' => (string) $alert->name,
                'status' => (string) $alert->status,
                'at' => optional($alert->created_at)->toIso8601String(),
            ])
            ->values()
            ->all();

        $searchActivity = AgentSearch::query()
            ->where('user_id', $user->id)
            ->where('created_at', '>=', $since)
            ->latest('id')
            ->limit(5)
            ->get()
            ->map(static fn (AgentSearch $search): array => [
                'type' => 'search',
                'title' => (string) $search->name,
                'status' => 'saved',
                'at' => optional($search->created_at)->toIso8601String(),
            ])
            ->values()
            ->all();

        $shareActivity = AgentShareLink::query()
            ->where('user_id', $user->id)
            ->where('created_at', '>=', $since)
            ->latest('id')
            ->limit(5)
            ->get()
            ->map(static fn (AgentShareLink $link): array => [
                'type' => 'share_link',
                'title' => 'Share link created',
                'status' => (string) ($link->status ?? 'active'),
                'at' => optional($link->created_at)->toIso8601String(),
            ])
            ->values()
            ->all();

        $customEvents = AgentActivityEvent::query()
            ->where('user_id', $user->id)
            ->where('created_at', '>=', $since)
            ->latest()
            ->limit(10)
            ->get()
            ->map(static fn (AgentActivityEvent $event): array => [
                'type' => $event->event_type,
                'title' => $event->title,
                'status' => $event->status,
                'at' => optional($event->created_at)->toIso8601String(),
            ])
            ->values()
            ->all();

        $alertRunActivity = AgentAlertRun::query()
            ->whereHas('agentAlert', static function (Builder $query) use ($user): void {
                $query->where('user_id', $user->id);
            })
            ->whereNotNull('ran_at')
            ->where('ran_at', '>=', $since)
            ->latest('ran_at')
            ->limit(12)
            ->with(['agentAlert:id,user_id,name,alert_type'])
            ->get()
            ->map(static function (AgentAlertRun $run): array {
                $alert = $run->agentAlert;
                $name = $alert !== null ? (string) $alert->name : 'Alert';
                $alertType = $alert !== null ? (string) $alert->alert_type : '';
                $meta = is_array($run->metadata_json) ? $run->metadata_json : [];
                $listingQuery = is_array($meta['listing_query'] ?? null) ? $meta['listing_query'] : [];
                $newCount = (int) ($listingQuery['new_match_count'] ?? 0);
                $title = $newCount > 0
                    ? sprintf('Alert: %d new listing match(es) for %s', $newCount, $name)
                    : 'Alert sent: '.$name;

                return [
                    'type' => 'alert_run',
                    'title' => $title,
                    'status' => (string) $run->status,
                    'at' => optional($run->ran_at)->toIso8601String(),
                    'alert_type' => $alertType,
                    'alert_id' => $alert?->id,
                    'new_match_count' => $newCount,
                ];
            })
            ->values()
            ->all();

        $feed = collect(array_merge($leadActivity, $alertActivity, $searchActivity, $shareActivity, $customEvents, $alertRunActivity))
            ->sortByDesc('at')
            ->take(15)
            ->values()
            ->all();

        $upcomingAlerts = AgentAlert::query()
            ->where('user_id', $user->id)
            ->where('status', 'active')
            ->orderByRaw('next_run_at IS NULL')
            ->orderBy('next_run_at')
            ->limit(6)
            ->get()
            ->map(static function (AgentAlert $alert): array {
                $schedule = is_array($alert->schedule_json) ? $alert->schedule_json : [];

                return [
                    'id' => $alert->id,
                    'name' => (string) $alert->name,
                    'alert_type' => (string) $alert->alert_type,
                    'cadence' => (string) ($schedule['cadence'] ?? 'daily'),
                    'next_run_at' => optional($alert->next_run_at)->toIso8601String(),
                ];
            })
            ->values()
            ->all();

        return response()->json([
            'data' => [
                'kpis' => [
                    'contacts' => $leadCount,
                    'active_alerts' => $activeAlertCount,
                    'saved_searches' => $savedSearchCount,
                    'active_share_links' => $activeShareLinkCount,
                ],
                'activity_feed' => $feed,
                'upcoming_alerts' => $upcomingAlerts,
                'period_days' => $periodDays,
            ],
        ]);
    }

    public function recordEvent(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $data = $request->validate([
            'event_type' => ['required', 'string', 'max:64'],
            'title' => ['required', 'string', 'max:255'],
            'status' => ['sometimes', 'string', 'max:32'],
            'metadata' => ['sometimes', 'array'],
        ]);

        $event = AgentActivityEvent::query()->create([
            'user_id' => $user->id,
            'event_type' => $data['event_type'],
            'title' => $data['title'],
            'status' => $data['status'] ?? 'completed',
            'metadata_json' => $data['metadata'] ?? null,
        ]);

        return response()->json(['data' => $event], 201);
    }
}
