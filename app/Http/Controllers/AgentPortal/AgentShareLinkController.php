<?php

namespace App\Http\Controllers\AgentPortal;

use App\Http\Controllers\Controller;
use App\Http\Requests\AgentPortal\AgentShareLinkRequest;
use App\Models\AgentSearch;
use App\Models\AgentSeoLandingPage;
use App\Models\AgentShareLink;
use App\Models\User;
use App\Services\AgentPortal\FeatureFlagService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Collection;
use Illuminate\Support\Str;
use Symfony\Component\HttpFoundation\StreamedResponse;

class AgentShareLinkController extends Controller
{
    public function operations(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $pruneDays = (int) config('agent_portal.share_links.prune_days', 90);
        $cutoff = now()->subDays($pruneDays);
        $candidateCount = AgentShareLink::query()
            ->where('user_id', $user->id)
            ->whereNotNull('expires_at')
            ->where('expires_at', '<=', $cutoff)
            ->count();

        return response()->json([
            'data' => [
                'prune_days' => $pruneDays,
                'prune_candidate_count' => $candidateCount,
                'schedule' => 'daily 03:30',
                'prune_command' => sprintf('agent:prune-share-links --days=%d', $pruneDays),
            ],
        ]);
    }

    public function operationsEstimate(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $validated = $request->validate([
            'days' => ['required', 'integer', 'min:1', 'max:3650'],
        ]);
        $days = (int) $validated['days'];

        $cutoff = now()->subDays($days);
        $candidateCount = AgentShareLink::query()
            ->where('user_id', $user->id)
            ->whereNotNull('expires_at')
            ->where('expires_at', '<=', $cutoff)
            ->count();

        return response()->json([
            'data' => [
                'days' => $days,
                'prune_candidate_count' => $candidateCount,
                'cutoff' => $cutoff->toIso8601String(),
            ],
        ]);
    }

    public function metrics(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $rows = AgentShareLink::query()
            ->where('user_id', $user->id)
            ->get();
        $windowDays = $this->resolveWindowDays($request);
        $summary = $this->metricsSummary($rows, $windowDays);

        return response()->json([
            'data' => $summary,
        ]);
    }

    public function metricsHistory(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $windowDays = (int) $request->query('days', 7);
        if (! in_array($windowDays, [7, 30], true)) {
            $windowDays = 7;
        }
        $buckets = $this->historyBucketsForUser($user, $windowDays);

        return response()->json([
            'data' => [
                'window_days' => $windowDays,
                'buckets' => $buckets,
            ],
        ]);
    }

    public function seoLandings(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $status = trim((string) $request->query('status', ''));
        $rows = AgentSeoLandingPage::query()
            ->where('user_id', $user->id)
            ->when($status !== '', fn ($query) => $query->where('status', $status))
            ->with(['search:id,name', 'shareLink:id,token'])
            ->latest('id')
            ->get()
            ->map(static fn (AgentSeoLandingPage $row): array => [
                'id' => $row->id,
                'agent_search_id' => $row->agent_search_id,
                'agent_search_name' => $row->search?->name,
                'agent_share_link_id' => $row->agent_share_link_id,
                'token' => $row->shareLink?->token,
                'slug' => $row->slug,
                'canonical_path' => $row->canonical_path,
                'canonical_url' => $row->canonical_url,
                'status' => $row->status,
                'published_at' => optional($row->published_at)?->toIso8601String(),
                'created_at' => optional($row->created_at)?->toIso8601String(),
            ])
            ->values();

        return response()->json(['data' => $rows]);
    }

    public function metricsHistoryCsv(Request $request): StreamedResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $windowDays = $this->resolveWindowDays($request);
        $buckets = $this->historyBucketsForUser($user, $windowDays);

        return response()->streamDownload(function () use ($buckets): void {
            $out = fopen('php://output', 'wb');
            if ($out === false) {
                return;
            }
            fputcsv($out, ['date', 'created', 'visits']);
            foreach ($buckets as $bucket) {
                fputcsv($out, [
                    (string) ($bucket['date'] ?? ''),
                    (string) ($bucket['created'] ?? 0),
                    (string) ($bucket['visits'] ?? 0),
                ]);
            }
            fclose($out);
        }, 'agent-share-links-history.csv', [
            'Content-Type' => 'text/csv; charset=UTF-8',
        ]);
    }

    public function index(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $forbidden = $this->seoLandingListFilterForbiddenResponse($user, $request);
        if ($forbidden !== null) {
            return $forbidden;
        }

        $rows = $this->filteredLinksForUser($request, $user)
            ->map(fn (AgentShareLink $link): array => $this->toPayload($link));

        return response()->json(['data' => $rows->values()]);
    }

    public function exportCsv(Request $request): JsonResponse|StreamedResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $forbidden = $this->seoLandingListFilterForbiddenResponse($user, $request);
        if ($forbidden !== null) {
            return $forbidden;
        }

        $rows = $this->filteredLinksForUser($request, $user);
        $windowDays = $this->resolveWindowDays($request);
        $summary = $this->metricsSummary(
            AgentShareLink::query()->where('user_id', $user->id)->get(),
            $windowDays,
        );

        return response()->streamDownload(function () use ($rows, $summary): void {
            $out = fopen('php://output', 'wb');
            if ($out === false) {
                return;
            }

            fputcsv($out, ['metric', 'window_days', 'created_in_window', 'created_previous_window', 'created_delta', 'visits_in_window', 'visits_previous_window', 'visits_delta']);
            fputcsv($out, [
                'summary',
                (string) $summary['window_days'],
                (string) $summary['created_in_window'],
                (string) $summary['created_previous_window'],
                (string) $summary['created_delta'],
                (string) $summary['visits_in_window'],
                (string) $summary['visits_previous_window'],
                (string) $summary['visits_delta'],
            ]);
            fputcsv($out, []);
            fputcsv($out, ['token', 'template_kind', 'status', 'canonical_path', 'created_at']);
            foreach ($rows as $link) {
                $payload = $this->toPayload($link);
                $kind = (string) (($payload['attribution_json']['template_kind'] ?? 'standard'));
                fputcsv($out, [
                    (string) $payload['token'],
                    $kind,
                    (string) $payload['status'],
                    (string) $payload['canonical_path'],
                    (string) ($payload['created_at'] ?? ''),
                ]);
            }
            fclose($out);
        }, 'agent-share-links.csv', [
            'Content-Type' => 'text/csv; charset=UTF-8',
        ]);
    }

    public function store(AgentShareLinkRequest $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $validated = $request->validated();
        $searchId = $validated['agent_search_id'] ?? null;
        $templateKind = (string) ($validated['template_kind'] ?? 'standard');

        if ($templateKind === 'seo_landing' && ! app(FeatureFlagService::class)->isEnabled($user, 'seo_landing_pages')) {
            return response()->json([
                'message' => 'The seo_landing_pages feature is disabled for this account.',
                'module' => 'seo_landing_pages',
            ], 403);
        }

        if ($searchId !== null) {
            AgentSearch::query()
                ->where('id', $searchId)
                ->where('user_id', $user->id)
                ->firstOrFail();
        }

        if ($templateKind === 'seo_landing' && $searchId !== null) {
            $existing = AgentShareLink::query()
                ->where('user_id', $user->id)
                ->where('agent_search_id', $searchId)
                ->where(function ($query): void {
                    $query->whereNull('expires_at')
                        ->orWhere('expires_at', '>', now());
                })
                ->latest('id')
                ->get()
                ->first(function (AgentShareLink $row): bool {
                    $data = is_array($row->attribution_json) ? $row->attribution_json : [];

                    return ($data['template_kind'] ?? null) === 'seo_landing';
                });

            if ($existing instanceof AgentShareLink) {
                $this->upsertSeoLandingPage($user, $existing);

                return response()->json(['data' => $this->toPayload($existing->load('agentSearch'), true)]);
            }
        }

        $token = $this->generateUniqueToken();
        $attribution = is_array($validated['attribution_json'] ?? null) ? $validated['attribution_json'] : [];
        if ($templateKind === 'seo_landing') {
            $attribution['template_kind'] = 'seo_landing';
        }

        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $searchId,
            'token' => $token,
            'attribution_json' => $attribution,
            'expires_at' => $validated['expires_at'] ?? null,
        ]);
        $link->load('agentSearch');
        if ($templateKind === 'seo_landing') {
            $this->upsertSeoLandingPage($user, $link);
        }

        return response()->json(['data' => $this->toPayload($link, false)], 201);
    }

    public function destroy(Request $request, int $shareLinkId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        /** @var AgentShareLink $link */
        $link = AgentShareLink::query()
            ->where('id', $shareLinkId)
            ->where('user_id', $user->id)
            ->firstOrFail();

        $link->delete($link->getKey());

        return response()->json([], 204);
    }

    public function update(Request $request, int $shareLinkId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $validated = $request->validate([
            'status' => ['required', 'string', 'in:active,inactive'],
        ]);

        /** @var AgentShareLink $link */
        $link = AgentShareLink::query()
            ->where('id', $shareLinkId)
            ->where('user_id', $user->id)
            ->firstOrFail();

        $status = (string) $validated['status'];
        $attrs = is_array($link->attribution_json) ? $link->attribution_json : [];

        if ($status === 'active' && ($attrs['template_kind'] ?? 'standard') === 'seo_landing' && ! app(FeatureFlagService::class)->isEnabled($user, 'seo_landing_pages')) {
            return response()->json([
                'message' => 'The seo_landing_pages feature is disabled for this account.',
                'module' => 'seo_landing_pages',
            ], 403);
        }

        $link->forceFill([
            'expires_at' => $status === 'inactive' ? now() : null,
        ])->save();
        $link->load('agentSearch');

        if ($status === 'active' && ($attrs['template_kind'] ?? 'standard') === 'seo_landing') {
            $this->upsertSeoLandingPage($user, $link);
        }

        return response()->json([
            'data' => $this->toPayload($link->fresh('agentSearch'), false),
        ]);
    }

    private function upsertSeoLandingPage(User $user, AgentShareLink $link): void
    {
        $payload = $this->toPayload($link);
        $slug = $link->agentSearch !== null ? Str::slug((string) $link->agentSearch->name) : null;

        AgentSeoLandingPage::query()->updateOrCreate(
            [
                'user_id' => $user->id,
                'agent_search_id' => $link->agent_search_id,
            ],
            [
                'agent_share_link_id' => $link->id,
                'slug' => $slug,
                'canonical_path' => (string) $payload['canonical_path'],
                'canonical_url' => (string) $payload['canonical_url'],
                'status' => (string) $payload['status'],
                'published_at' => now(),
            ]
        );
    }

    private function generateUniqueToken(): string
    {
        do {
            $token = Str::lower(Str::random(22));
        } while (AgentShareLink::query()->where('token', $token)->exists());

        return $token;
    }

    /**
     * @return array<string, mixed>
     */
    private function toPayload(AgentShareLink $link, bool $reusedExisting = false): array
    {
        $platformUrl = rtrim((string) config('idx.platform_url'), '/');
        $slug = $link->agentSearch !== null ? Str::slug((string) $link->agentSearch->name) : '';
        $canonicalPath = '/shared/'.$link->token.($slug !== '' ? '/'.$slug : '');

        return [
            'id' => $link->id,
            'agent_search_id' => $link->agent_search_id,
            'token' => $link->token,
            'url' => $platformUrl.'/shared/'.$link->token,
            'canonical_url' => $platformUrl.$canonicalPath,
            'canonical_path' => $canonicalPath,
            'attribution_json' => $link->attribution_json ?? [],
            'expires_at' => optional($link->expires_at)?->toIso8601String(),
            'status' => ($link->expires_at !== null && $link->expires_at->isPast()) ? 'inactive' : 'active',
            'created_at' => optional($link->created_at)?->toIso8601String(),
            'agent_search_name' => $link->agentSearch?->name,
            'reused_existing' => $reusedExisting,
        ];
    }

    private function seoLandingListFilterForbiddenResponse(User $user, Request $request): ?JsonResponse
    {
        $templateKind = (string) $request->query('template_kind', '');
        if ($templateKind !== 'seo_landing') {
            return null;
        }
        if (app(FeatureFlagService::class)->isEnabled($user, 'seo_landing_pages')) {
            return null;
        }

        return response()->json([
            'message' => 'The seo_landing_pages feature is disabled for this account.',
            'module' => 'seo_landing_pages',
        ], 403);
    }

    /**
     * @return Collection<int, AgentShareLink>
     */
    private function filteredLinksForUser(Request $request, User $user): Collection
    {
        $templateKind = (string) $request->query('template_kind', '');
        $status = (string) $request->query('status', '');
        $queryText = trim((string) $request->query('q', ''));

        return AgentShareLink::query()
            ->where('user_id', $user->id)
            ->when($status === 'active', function ($query): void {
                $query->where(function ($inner): void {
                    $inner->whereNull('expires_at')
                        ->orWhere('expires_at', '>', now());
                });
            })
            ->when($status === 'inactive', function ($query): void {
                $query->whereNotNull('expires_at')
                    ->where('expires_at', '<=', now());
            })
            ->when($queryText !== '', function ($query) use ($queryText): void {
                $query->where('token', 'like', '%'.$queryText.'%');
            })
            ->with('agentSearch')
            ->latest('id')
            ->get()
            ->filter(function (AgentShareLink $link) use ($templateKind): bool {
                if ($templateKind === '') {
                    return true;
                }
                $kind = (string) ((is_array($link->attribution_json) ? $link->attribution_json : [])['template_kind'] ?? 'standard');

                return $kind === $templateKind;
            })
            ->values();
    }

    private function resolveWindowDays(Request $request): int
    {
        $windowDays = (int) $request->query('days', 7);
        if (! in_array($windowDays, [7, 30], true)) {
            return 7;
        }

        return $windowDays;
    }

    /**
     * @return list<array{date:string,created:int,visits:int}>
     */
    private function historyBucketsForUser(User $user, int $windowDays): array
    {
        $start = now()->startOfDay()->subDays($windowDays - 1);
        $rows = AgentShareLink::query()
            ->where('user_id', $user->id)
            ->where('created_at', '>=', $start)
            ->get();

        $buckets = [];
        for ($i = $windowDays - 1; $i >= 0; $i--) {
            $day = now()->startOfDay()->subDays($i);
            $key = $day->toDateString();
            $buckets[$key] = [
                'date' => $key,
                'created' => 0,
                'visits' => 0,
            ];
        }

        foreach ($rows as $row) {
            if ($row->created_at === null) {
                continue;
            }
            $dateKey = $row->created_at->startOfDay()->toDateString();
            if (! array_key_exists($dateKey, $buckets)) {
                continue;
            }
            $buckets[$dateKey]['created']++;
            $attribution = is_array($row->attribution_json) ? $row->attribution_json : [];
            $buckets[$dateKey]['visits'] += (int) ($attribution['visit_count'] ?? 0);
        }

        return array_values($buckets);
    }

    /**
     * @param  Collection<int, AgentShareLink>  $rows
     * @return array<string, int>
     */
    private function metricsSummary(Collection $rows, int $windowDays): array
    {
        $windowStart = now()->subDays($windowDays);
        $previousWindowStart = now()->subDays($windowDays * 2);

        $total = $rows->count();
        $activeTotal = $rows->filter(fn (AgentShareLink $link): bool => ! ($link->expires_at !== null && $link->expires_at->isPast()))->count();
        $inactiveTotal = $total - $activeTotal;

        $seoRows = $rows->filter(function (AgentShareLink $link): bool {
            $kind = (string) ((is_array($link->attribution_json) ? $link->attribution_json : [])['template_kind'] ?? 'standard');

            return $kind === 'seo_landing';
        });
        $seoTotal = $seoRows->count();
        $seoActiveTotal = $seoRows->filter(fn (AgentShareLink $link): bool => ! ($link->expires_at !== null && $link->expires_at->isPast()))->count();
        $seoInactiveTotal = $seoTotal - $seoActiveTotal;
        $createdInWindow = $rows->filter(fn (AgentShareLink $link): bool => $link->created_at !== null && $link->created_at->gte($windowStart))->count();
        $visitsInWindow = $rows
            ->filter(fn (AgentShareLink $link): bool => $link->created_at !== null && $link->created_at->gte($windowStart))
            ->sum(function (AgentShareLink $link): int {
                $attribution = is_array($link->attribution_json) ? $link->attribution_json : [];

                return (int) ($attribution['visit_count'] ?? 0);
            });
        $createdPreviousWindow = $rows
            ->filter(fn (AgentShareLink $link): bool => $link->created_at !== null && $link->created_at->gte($previousWindowStart) && $link->created_at->lt($windowStart))
            ->count();
        $visitsPreviousWindow = $rows
            ->filter(fn (AgentShareLink $link): bool => $link->created_at !== null && $link->created_at->gte($previousWindowStart) && $link->created_at->lt($windowStart))
            ->sum(function (AgentShareLink $link): int {
                $attribution = is_array($link->attribution_json) ? $link->attribution_json : [];

                return (int) ($attribution['visit_count'] ?? 0);
            });
        $createdDelta = $createdInWindow - $createdPreviousWindow;
        $visitsDelta = $visitsInWindow - $visitsPreviousWindow;

        return [
            'total' => $total,
            'active_total' => $activeTotal,
            'inactive_total' => $inactiveTotal,
            'seo_total' => $seoTotal,
            'seo_active_total' => $seoActiveTotal,
            'seo_inactive_total' => $seoInactiveTotal,
            'window_days' => $windowDays,
            'created_in_window' => $createdInWindow,
            'visits_in_window' => $visitsInWindow,
            'created_previous_window' => $createdPreviousWindow,
            'visits_previous_window' => $visitsPreviousWindow,
            'created_delta' => $createdDelta,
            'visits_delta' => $visitsDelta,
        ];
    }

    public function embedCode(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $type = (string) ($request->input('widget_type') ?? 'search');
        $searchId = (string) ($request->input('agent_search_id') ?? '');
        $maxListings = (int) ($request->input('max_listings') ?? 6);
        $theme = (string) ($request->input('theme') ?? 'light');
        $baseUrl = rtrim((string) config('app.url'), '/');

        $attrs = [
            'data-quantyra-widget="'.$type.'"',
            'data-theme="'.$theme.'"',
            'data-max-listings="'.max(1, min(50, $maxListings)).'"',
        ];
        if ($searchId !== '') {
            $attrs[] = 'data-search-id="'.$searchId.'"';
        }

        $embedCode = '<div '.implode(' ', $attrs).'></div>'."\n".'<script src="'.$baseUrl.'/widget/loader.js" async></script>';

        return response()->json([
            'data' => [
                'embed_code' => $embedCode,
                'widget_type' => $type,
                'theme' => $theme,
                'max_listings' => $maxListings,
            ],
        ]);
    }
}
