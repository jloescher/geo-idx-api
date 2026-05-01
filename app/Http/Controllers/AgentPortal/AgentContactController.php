<?php

namespace App\Http\Controllers\AgentPortal;

use App\Ghl\Sync\Models\QuantyraLead;
use App\Http\Controllers\Controller;
use App\Http\Requests\AgentPortal\AgentContactCreateHandoffAlertRequest;
use App\Http\Requests\AgentPortal\AgentContactHandoffAlertRequest;
use App\Http\Requests\AgentPortal\AgentContactUpdateRequest;
use App\Models\AgentActivityEvent;
use App\Models\AgentAlert;
use App\Models\AgentSearch;
use App\Models\User;
use App\Services\AgentPortal\AgentActivityTracker;
use App\Services\AgentPortal\ContactTagService;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use App\Services\SubscriberLeadScopeService;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\StreamedResponse;

class AgentContactController extends Controller
{
    public function index(Request $request, SubscriberLeadScopeService $scope): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $locationIds = $scope->locationIdsForUser($user);
        $perPage = max(1, min(100, (int) $request->query('per_page', 20)));
        $page = max(1, (int) $request->query('page', 1));
        if ($page === 1 && $this->contactsIndexHasActiveFilters($request)) {
            app(AgentActivityTracker::class)->record(
                $user,
                'apply_contacts_filter',
                'Applied contact list filters',
                'completed',
                ['query' => $request->only(['tab', 'search', 'status', 'domain', 'stage', 'tag', 'activity', 'date_from', 'date_to'])]
            );
        }

        $query = QuantyraLead::query()
            ->whereIn('ghl_location_id', $locationIds)
            ->latest('id');

        $query = $this->applyFilters($query, $request);
        $total = (clone $query)->count();
        $items = $query->forPage($page, $perPage)->get();

        return response()->json([
            'data' => [
                'items' => $items,
                'meta' => [
                    'total' => $total,
                    'page' => $page,
                    'per_page' => $perPage,
                ],
            ],
        ]);
    }

    public function show(Request $request, int $contactId, SubscriberLeadScopeService $scope): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $locationIds = $scope->locationIdsForUser($user);
        $contact = QuantyraLead::query()
            ->where('id', $contactId)
            ->whereIn('ghl_location_id', $locationIds)
            ->firstOrFail();

        return response()->json(['data' => $contact]);
    }

    public function activity(Request $request, int $contactId, SubscriberLeadScopeService $scope): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $contact = $this->resolveScopedContact($contactId, $scope->locationIdsForUser($user));
        $payload = is_array($contact->payload) ? $contact->payload : [];
        $events = $this->buildActivityTimeline($contact, $payload, $user);

        return response()->json(['data' => $events]);
    }

    public function update(
        AgentContactUpdateRequest $request,
        int $contactId,
        SubscriberLeadScopeService $scope
    ): JsonResponse {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $locationIds = $scope->locationIdsForUser($user);
        $contact = $this->resolveScopedContact($contactId, $locationIds);

        $payload = is_array($contact->payload) ? $contact->payload : [];
        $validated = $request->validated();

        $changes = [];
        if (array_key_exists('status', $validated)) {
            $payload['status'] = $validated['status'];
            $changes[] = 'status';
        }
        if (array_key_exists('stage', $validated)) {
            $payload['stage'] = $validated['stage'];
            $changes[] = 'stage';
        }
        if (array_key_exists('tags', $validated)) {
            $payload['tags'] = array_values($validated['tags']);
            app(ContactTagService::class)->syncTags($user, (string) $contact->id, $payload['tags']);
            $changes[] = 'tags';
        }
        if (array_key_exists('notes', $validated)) {
            $payload['notes'] = $validated['notes'];
            $changes[] = 'notes';
        }
        if ($changes !== []) {
            $payload['activity_log'] = $this->appendActivity(
                $payload,
                [
                    'type' => 'contact_updated',
                    'at' => now()->toIso8601String(),
                    'changes' => $changes,
                ]
            );
            AgentActivityEvent::query()->create([
                'user_id' => $user->id,
                'event_type' => 'contact_updated',
                'title' => 'Updated contact #'.$contact->id,
                'status' => 'completed',
                'metadata_json' => [
                    'contact_id' => $contact->id,
                    'changes' => $changes,
                ],
            ]);
        }

        $contact->payload = $payload;
        $contact->save();

        return response()->json(['data' => $contact->fresh()]);
    }

    public function getTags(Request $request, int $contactId, SubscriberLeadScopeService $scope): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $locationIds = $scope->locationIdsForUser($user);
        $contact = $this->resolveScopedContact($contactId, $locationIds);
        $tags = app(ContactTagService::class)->getTagsForLead($user, (string) $contact->id);

        return response()->json(['data' => ['contact_id' => $contact->id, 'tags' => $tags]]);
    }

    public function syncTags(Request $request, int $contactId, SubscriberLeadScopeService $scope): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $data = $request->validate([
            'tags' => ['required', 'array'],
            'tags.*' => ['string', 'max:128'],
        ]);

        $locationIds = $scope->locationIdsForUser($user);
        $contact = $this->resolveScopedContact($contactId, $locationIds);
        $tags = app(ContactTagService::class)->syncTags($user, (string) $contact->id, $data['tags']);

        $payload = is_array($contact->payload) ? $contact->payload : [];
        $payload['tags'] = $tags;
        $payload['activity_log'] = $this->appendActivity(
            $payload,
            [
                'type' => 'contact_tags_synced',
                'at' => now()->toIso8601String(),
                'tag_count' => count($tags),
            ]
        );
        $contact->payload = $payload;
        $contact->save();
        AgentActivityEvent::query()->create([
            'user_id' => $user->id,
            'event_type' => 'contact_tags_synced',
            'title' => 'Synced tags for contact #'.$contact->id,
            'status' => 'completed',
            'metadata_json' => [
                'contact_id' => $contact->id,
                'tag_count' => count($tags),
                'tags' => $tags,
            ],
        ]);

        return response()->json(['data' => ['contact_id' => $contact->id, 'tags' => $tags]]);
    }

    public function handoffToAlert(
        AgentContactHandoffAlertRequest $request,
        int $contactId,
        SubscriberLeadScopeService $scope,
        SubscriberFeedAccessService $feeds
    ): JsonResponse {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        if (($blocked = $this->rejectIfNoAlertFeeds($user, $feeds)) !== null) {
            return $blocked;
        }

        $locationIds = $scope->locationIdsForUser($user);
        $contact = $this->resolveScopedContact($contactId, $locationIds);

        $validated = $request->validated();
        $search = AgentSearch::query()
            ->where('id', (int) $validated['agent_search_id'])
            ->where('user_id', $user->id)
            ->firstOrFail();

        $payload = is_array($contact->payload) ? $contact->payload : [];
        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'name' => (string) $validated['name'],
            'alert_type' => (string) ($validated['alert_type'] ?? 'listing'),
            'status' => (string) ($validated['status'] ?? 'active'),
            'schedule_json' => [
                'cadence' => (string) ($validated['cadence'] ?? 'daily'),
                'handoff' => [
                    'contact_lead_id' => $contact->id,
                    'contact_email' => $payload['email'] ?? null,
                    'contact_name' => $payload['name'] ?? null,
                ],
            ],
            'next_run_at' => null,
        ]);

        $payload['activity_log'] = $this->appendActivity(
            $payload,
            [
                'type' => 'search_to_alert_handoff',
                'at' => now()->toIso8601String(),
                'alert_id' => $alert->id,
                'agent_search_id' => $search->id,
            ]
        );
        $contact->payload = $payload;
        $contact->save();
        AgentActivityEvent::query()->create([
            'user_id' => $user->id,
            'event_type' => 'search_to_alert_handoff',
            'title' => 'Handoff contact #'.$contact->id.' to alert #'.$alert->id,
            'status' => 'completed',
            'metadata_json' => [
                'contact_id' => $contact->id,
                'alert_id' => $alert->id,
                'agent_search_id' => $search->id,
            ],
        ]);

        return response()->json([
            'data' => [
                'alert' => $alert->load(['agentSearch', 'runs']),
                'contact' => $contact->fresh(),
            ],
        ], 201);
    }

    public function createAndHandoffToAlert(
        AgentContactCreateHandoffAlertRequest $request,
        SubscriberFeedAccessService $feeds
    ): JsonResponse {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        if (($blocked = $this->rejectIfNoAlertFeeds($user, $feeds)) !== null) {
            return $blocked;
        }

        $validated = $request->validated();
        $search = AgentSearch::query()
            ->where('id', (int) $validated['agent_search_id'])
            ->where('user_id', $user->id)
            ->firstOrFail();

        $contactPayload = is_array($validated['contact'] ?? null)
            ? $validated['contact']
            : [];
        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'agent_portal',
            'payload' => [
                'name' => (string) ($contactPayload['name'] ?? 'New Contact'),
                'email' => $contactPayload['email'] ?? null,
                'phone' => $contactPayload['phone'] ?? null,
                'status' => (string) ($contactPayload['status'] ?? 'new'),
                'activity_log' => [
                    [
                        'type' => 'contact_created_from_handoff',
                        'at' => now()->toIso8601String(),
                        'agent_search_id' => $search->id,
                    ],
                ],
            ],
            'quantyra_domain' => null,
        ]);

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'name' => (string) $validated['name'],
            'alert_type' => (string) ($validated['alert_type'] ?? 'listing'),
            'status' => (string) ($validated['status'] ?? 'active'),
            'schedule_json' => [
                'cadence' => (string) ($validated['cadence'] ?? 'daily'),
                'handoff' => [
                    'contact_lead_id' => $lead->id,
                    'contact_email' => $contactPayload['email'] ?? null,
                    'contact_name' => $contactPayload['name'] ?? null,
                ],
            ],
            'next_run_at' => null,
        ]);

        return response()->json([
            'data' => [
                'alert' => $alert->load(['agentSearch', 'runs']),
                'contact' => $lead,
            ],
        ], 201);
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

    private function contactsIndexHasActiveFilters(Request $request): bool
    {
        $tab = trim((string) $request->query('tab', 'all'));
        if ($tab !== '' && $tab !== 'all') {
            return true;
        }

        foreach (['search', 'status', 'domain', 'stage', 'tag', 'activity', 'date_from', 'date_to'] as $key) {
            if (trim((string) $request->query($key, '')) !== '') {
                return true;
            }
        }

        return false;
    }

    private function applyFilters(Builder $query, Request $request): Builder
    {
        $search = trim((string) $request->query('search', ''));
        $status = trim((string) $request->query('status', ''));
        $domain = trim((string) $request->query('domain', ''));
        $stage = trim((string) $request->query('stage', ''));
        $activity = trim((string) $request->query('activity', ''));
        $tag = trim((string) $request->query('tag', ''));
        $dateFrom = trim((string) $request->query('date_from', ''));
        $dateTo = trim((string) $request->query('date_to', ''));
        $tab = trim((string) $request->query('tab', 'all'));

        if ($tab === 'smart' || $tab === 'nurtured') {
            $query->where(function (Builder $smart): void {
                $smart->where('payload->status', 'hot')
                    ->orWhereRaw('LOWER(payload->>\'stage\') = ?', ['nurture'])
                    ->orWhereRaw('LOWER(payload->>\'stage\') = ?', ['qualified']);
            });
        }
        if ($tab === 'awaiting') {
            $query->where(function (Builder $awaiting): void {
                $awaiting->where('payload->status', 'new')
                    ->orWhere('payload->status', 'contacted');
            });
        }
        if ($tab === 'archive') {
            $query->where(function (Builder $archived): void {
                $archived->where('payload->status', 'archived')
                    ->orWhere('payload->status', 'inactive');
            });
        }

        if ($search !== '') {
            $searchLike = '%'.mb_strtolower($search).'%';
            $query->where(function (Builder $inner) use ($searchLike): void {
                $inner->whereRaw('LOWER(payload->>\'name\') like ?', [$searchLike])
                    ->orWhereRaw('LOWER(payload->>\'first_name\') like ?', [$searchLike])
                    ->orWhereRaw('LOWER(payload->>\'last_name\') like ?', [$searchLike])
                    ->orWhereRaw('LOWER(payload->>\'email\') like ?', [$searchLike])
                    ->orWhereRaw('LOWER(payload->>\'phone\') like ?', [$searchLike])
                    ->orWhereRaw('LOWER(payload->>\'property_interest\') like ?', [$searchLike]);
            });
        }

        if ($status !== '') {
            $query->where('payload->status', $status);
        }
        if ($domain !== '') {
            $query->where('quantyra_domain', $domain);
        }
        if ($stage !== '') {
            $query->whereRaw('LOWER(payload->>\'stage\') = ?', [mb_strtolower($stage)]);
        }
        if ($tag !== '') {
            $query->whereRaw('LOWER(CAST(payload->\'tags\' as text)) like ?', ['%'.mb_strtolower($tag).'%']);
        }
        if ($activity === 'recent') {
            $query->where('created_at', '>=', now()->subDays(7));
        }
        if ($activity === 'stale') {
            $query->where('created_at', '<', now()->subDays(30));
        }
        if ($dateFrom !== '') {
            $query->whereDate('created_at', '>=', $dateFrom);
        }
        if ($dateTo !== '') {
            $query->whereDate('created_at', '<=', $dateTo);
        }

        return $query;
    }

    /**
     * @param  list<string>  $locationIds
     */
    private function resolveScopedContact(int $contactId, array $locationIds): QuantyraLead
    {
        return QuantyraLead::query()
            ->where('id', $contactId)
            ->whereIn('ghl_location_id', $locationIds)
            ->firstOrFail();
    }

    /**
     * @param  array<string, mixed>  $payload
     * @return list<array<string, mixed>>
     */
    private function appendActivity(array $payload, array $entry): array
    {
        $events = [];
        if (isset($payload['activity_log']) && is_array($payload['activity_log'])) {
            foreach ($payload['activity_log'] as $raw) {
                if (is_array($raw)) {
                    $events[] = $raw;
                }
            }
        }
        $events[] = $entry;

        return $events;
    }

    /**
     * @param  array<string, mixed>  $payload
     * @return list<array<string, mixed>>
     */
    private function buildActivityTimeline(QuantyraLead $contact, array $payload, User $user): array
    {
        $events = [
            [
                'type' => 'lead_created',
                'at' => optional($contact->created_at)->toIso8601String(),
                'lead_id' => $contact->id,
            ],
        ];

        if ($contact->updated_at !== null && $contact->updated_at->ne($contact->created_at)) {
            $events[] = [
                'type' => 'lead_updated',
                'at' => $contact->updated_at->toIso8601String(),
                'lead_id' => $contact->id,
            ];
        }

        if (isset($payload['activity_log']) && is_array($payload['activity_log'])) {
            foreach ($payload['activity_log'] as $entry) {
                if (is_array($entry)) {
                    $events[] = $entry;
                }
            }
        }

        $recentEvents = AgentActivityEvent::query()
            ->where('user_id', $user->id)
            ->latest()
            ->limit(200)
            ->get();
        foreach ($recentEvents as $event) {
            $metadata = is_array($event->metadata_json) ? $event->metadata_json : [];
            if ((string) ($metadata['contact_id'] ?? '') !== (string) $contact->id) {
                continue;
            }
            $events[] = [
                'type' => $event->event_type,
                'at' => optional($event->created_at)->toIso8601String(),
                'title' => $event->title,
                'status' => $event->status,
                'metadata' => $metadata,
            ];
        }

        usort($events, static function (array $a, array $b): int {
            $left = (string) ($a['at'] ?? '');
            $right = (string) ($b['at'] ?? '');

            return strcmp($right, $left);
        });

        return array_map(fn (array $event): array => $this->enrichActivityEvent($event), $events);
    }

    /**
     * @param  array<string, mixed>  $event
     * @return array<string, mixed>
     */
    private function enrichActivityEvent(array $event): array
    {
        $type = (string) ($event['type'] ?? '');
        $event['channel'] = $this->activityEventChannel($type);

        return $event;
    }

    private function activityEventChannel(string $type): string
    {
        $t = mb_strtolower($type);

        if (
            str_contains($t, 'email')
            || str_contains($t, 'mail_opened')
            || str_contains($t, 'smtp')
            || str_contains($t, 'newsletter')
        ) {
            return 'email';
        }

        if (
            str_contains($t, 'listing_view')
            || str_contains($t, 'page_view')
            || str_contains($t, 'site_visit')
            || str_contains($t, 'widget_')
            || str_contains($t, 'property_view')
            || str_contains($t, 'idx_session')
        ) {
            return 'site';
        }

        if (str_starts_with($t, 'contact_')) {
            return 'crm';
        }

        if (str_contains($t, 'alert') || str_contains($t, 'handoff')) {
            return 'alert';
        }

        if ($t === 'lead_created' || $t === 'lead_updated') {
            return 'system';
        }

        if (in_array($t, ['apply_filter', 'apply_contacts_filter', 'map_draw', 'map_draw_search', 'save_search', 'search_execute'], true)) {
            return 'crm';
        }

        return 'other';
    }

    public function bulkStatus(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $data = $request->validate([
            'contact_ids' => ['required', 'array', 'min:1'],
            'contact_ids.*' => ['integer'],
            'status' => ['required', 'string', 'max:64'],
        ]);

        $scope = app(SubscriberLeadScopeService::class);
        $locationIds = $scope->locationIdsForUser($user);

        $updated = QuantyraLead::query()
            ->whereIn('ghl_location_id', $locationIds)
            ->whereIn('id', $data['contact_ids'])
            ->get();

        $count = 0;
        foreach ($updated as $lead) {
            $payload = is_array($lead->payload) ? $lead->payload : [];
            $payload['status'] = $data['status'];
            $lead->payload = $payload;
            $lead->save();
            $count++;
        }

        return response()->json(['data' => ['updated' => $count]]);
    }

    public function bulkDelete(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $data = $request->validate([
            'contact_ids' => ['required', 'array', 'min:1'],
            'contact_ids.*' => ['integer'],
        ]);

        $scope = app(SubscriberLeadScopeService::class);
        $locationIds = $scope->locationIdsForUser($user);

        $count = QuantyraLead::query()
            ->whereIn('ghl_location_id', $locationIds)
            ->whereIn('id', $data['contact_ids'])
            ->delete();

        return response()->json(['data' => ['deleted' => $count]]);
    }

    public function exportCsv(Request $request): StreamedResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $scope = app(SubscriberLeadScopeService::class);
        $locationIds = $scope->locationIdsForUser($user);

        $leads = QuantyraLead::query()
            ->whereIn('ghl_location_id', $locationIds)
            ->latest('id')
            ->limit(5000)
            ->get();

        $headers = [
            'Content-Type' => 'text/csv',
            'Content-Disposition' => 'attachment; filename="contacts-'.date('Y-m-d').'.csv"',
        ];

        return response()->streamDownload(function () use ($leads): void {
            $file = fopen('php://output', 'w');
            fputcsv($file, ['ID', 'Name', 'Email', 'Phone', 'Status', 'Source', 'Created']);

            foreach ($leads as $lead) {
                $payload = is_array($lead->payload) ? $lead->payload : [];
                fputcsv($file, [
                    $lead->id,
                    $payload['name'] ?? '',
                    $payload['email'] ?? '',
                    $payload['phone'] ?? '',
                    $payload['status'] ?? 'new',
                    $lead->source ?? '',
                    $lead->created_at?->toDateTimeString() ?? '',
                ]);
            }
            fclose($file);
        }, 'contacts-'.date('Y-m-d').'.csv', $headers);
    }
}
