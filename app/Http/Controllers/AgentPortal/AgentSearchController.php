<?php

namespace App\Http\Controllers\AgentPortal;

use App\Http\Controllers\Controller;
use App\Http\Requests\AgentPortal\AgentSearchExecuteRequest;
use App\Http\Requests\AgentPortal\AgentSearchUpsertRequest;
use App\Models\AgentSearch;
use App\Models\AgentSearchFilter;
use App\Models\AgentSearchGeometry;
use App\Models\User;
use App\Services\AgentPortal\AgentActivityTracker;
use App\Services\AgentPortal\AgentSearchGeometrySafeguardService;
use App\Services\AgentPortal\Contracts\LookupProviderInterface;
use App\Services\AgentPortal\Contracts\MultiMlsQueryCompilerInterface;
use App\Services\AgentPortal\Contracts\SearchExecutionBrokerInterface;
use App\Services\AgentPortal\FieldCatalogService;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\RateLimiter;

class AgentSearchController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $rows = AgentSearch::query()
            ->where('user_id', $user->id)
            ->with(['filters', 'geometries'])
            ->latest('id')
            ->get();

        return response()->json(['data' => $rows]);
    }

    public function show(Request $request, int $searchId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $search = AgentSearch::query()
            ->where('id', $searchId)
            ->where('user_id', $user->id)
            ->with(['filters', 'geometries'])
            ->firstOrFail();

        return response()->json(['data' => $search]);
    }

    public function store(AgentSearchUpsertRequest $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $payload = $request->validated();
        $search = DB::transaction(function () use ($user, $payload): AgentSearch {
            $search = AgentSearch::query()->create([
                'user_id' => $user->id,
                'name' => (string) $payload['name'],
                'search_state_json' => $payload['search_state_json'] ?? [],
                'mls_scope_json' => $payload['mls_scope_json'] ?? [],
                'is_template' => (bool) ($payload['is_template'] ?? false),
                'source' => (string) ($payload['source'] ?? 'manual'),
            ]);

            $this->syncNestedRows($search, $payload);

            return $search->load(['filters', 'geometries']);
        });

        app(AgentActivityTracker::class)->record(
            $user,
            'save_search',
            "Saved search: {$search->name}",
        );

        return response()->json(['data' => $search], 201);
    }

    public function update(AgentSearchUpsertRequest $request, int $searchId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $search = AgentSearch::query()
            ->where('id', $searchId)
            ->where('user_id', $user->id)
            ->firstOrFail();

        $payload = $request->validated();
        DB::transaction(function () use ($search, $payload): void {
            $search->update([
                'name' => (string) $payload['name'],
                'search_state_json' => $payload['search_state_json'] ?? [],
                'mls_scope_json' => $payload['mls_scope_json'] ?? [],
                'is_template' => (bool) ($payload['is_template'] ?? false),
                'source' => (string) ($payload['source'] ?? 'manual'),
            ]);

            $this->syncNestedRows($search, $payload);
        });

        return response()->json(['data' => $search->load(['filters', 'geometries'])]);
    }

    public function destroy(Request $request, int $searchId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        AgentSearch::query()
            ->where('id', $searchId)
            ->where('user_id', $user->id)
            ->firstOrFail();
        AgentSearch::query()->where('id', $searchId)->where('user_id', $user->id)->delete();

        return response()->json([], 204);
    }

    public function execute(
        AgentSearchExecuteRequest $request,
        MultiMlsQueryCompilerInterface $compiler,
        SearchExecutionBrokerInterface $broker,
        SubscriberFeedAccessService $feeds
    ): JsonResponse {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $validated = $request->validated();
        $telemetryTrigger = isset($validated['telemetry']['trigger'])
            ? (string) $validated['telemetry']['trigger']
            : null;
        $search = null;
        $agentSearchId = null;
        if (isset($validated['agent_search_id'])) {
            $agentSearchId = (int) $validated['agent_search_id'];
            $search = AgentSearch::query()
                ->where('id', $agentSearchId)
                ->where('user_id', $user->id)
                ->with(['filters', 'geometries'])
                ->firstOrFail();
        }

        $scope = $feeds->resolvedSearchScopesForUser($user);
        $filters = $search !== null
            ? $search->filters->map(fn (AgentSearchFilter $filter): array => [
                'field' => (string) $filter->canonical_field_key,
                'operator' => (string) $filter->operator,
                'value' => $filter->value_json,
            ])->values()->all()
            : (array) ($validated['filters'] ?? []);
        $geometries = $search !== null
            ? $search->geometries->map(fn (AgentSearchGeometry $geometry): array => [
                'geometry_type' => (string) $geometry->geometry_type,
                'mode' => (string) $geometry->mode,
                'geojson' => (array) ($geometry->geojson ?? []),
            ])->values()->all()
            : (array) ($validated['geometries'] ?? []);

        $geometryErrors = app(AgentSearchGeometrySafeguardService::class)->validate($geometries);
        if ($geometryErrors !== []) {
            return response()->json(['errors' => $geometryErrors], 422);
        }

        $errors = $compiler->validate($filters, $scope);
        if ($errors !== []) {
            return response()->json(['errors' => $errors], 422);
        }

        $compiled = $compiler->compile($filters, $scope);
        foreach ($compiled as $key => $query) {
            if (! is_array($query)) {
                continue;
            }
            $compiled[$key]['geometries'] = $geometries;
        }
        $cacheKey = $this->queryPlanCacheKey($user->id, $filters, $scope, $geometries);
        $cacheHit = Cache::has($cacheKey);
        $merged = Cache::remember($cacheKey, now()->addSeconds(90), function () use ($broker, $compiled): array {
            $raw = $broker->execute($compiled);

            return $broker->merge($raw);
        });

        $resolvedTrigger = $telemetryTrigger
            ?? ($agentSearchId !== null ? 'saved_loaded' : $this->inferSearchExecuteTrigger($filters));
        $this->recordSearchExecutedActivity(
            $user,
            $merged,
            $filters,
            $geometries,
            $cacheHit,
            count($compiled),
            count($scope),
            $resolvedTrigger,
            'workspace',
            $agentSearchId,
        );

        return response()->json([
            'data' => $merged,
            'meta' => [
                'compiled_query_count' => count($compiled),
                'scope_count' => count($scope),
                'cache_hit' => $cacheHit,
            ],
        ]);
    }

    public function lookupOptions(
        Request $request,
        SubscriberFeedAccessService $feeds,
        LookupProviderInterface $lookups
    ): JsonResponse {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $scopes = $feeds->resolvedSearchScopesForUser($user);
        $fieldKey = (string) $request->query('field', '');
        $rows = [];
        foreach ($scopes as $scope) {
            $mls = (string) ($scope['mls_code'] ?? '');
            $dataset = (string) ($scope['dataset_code'] ?? '');
            if ($mls === '' || $dataset === '') {
                continue;
            }

            $values = $fieldKey !== ''
                ? $lookups->fetchEnumValues($mls, $dataset, $fieldKey)
                : $lookups->fetchFieldCatalog($mls, $dataset);

            $rows[] = [
                'mls_code' => $mls,
                'dataset_code' => $dataset,
                'values' => $values,
            ];
        }

        return response()->json(['data' => $rows]);
    }

    /**
     * @param  array<string, mixed>  $payload
     */
    private function syncNestedRows(AgentSearch $search, array $payload): void
    {
        $search->filters()->delete();
        $search->geometries()->delete();

        foreach ((array) ($payload['filters'] ?? []) as $filter) {
            if (! is_array($filter)) {
                continue;
            }
            AgentSearchFilter::query()->create([
                'agent_search_id' => $search->id,
                'canonical_field_key' => (string) ($filter['canonical_field_key'] ?? ''),
                'operator' => (string) ($filter['operator'] ?? ''),
                'value_json' => $filter['value_json'] ?? null,
                'applies_to_mls_json' => $filter['applies_to_mls_json'] ?? null,
            ]);
        }

        foreach ((array) ($payload['geometries'] ?? []) as $geometry) {
            if (! is_array($geometry)) {
                continue;
            }
            AgentSearchGeometry::query()->create([
                'agent_search_id' => $search->id,
                'geometry_type' => (string) ($geometry['geometry_type'] ?? 'polygon'),
                'mode' => (string) ($geometry['mode'] ?? 'include'),
                'geojson' => (array) ($geometry['geojson'] ?? []),
                'bbox_json' => $geometry['bbox_json'] ?? null,
                'area_m2' => $geometry['area_m2'] ?? null,
            ]);
        }
    }

    public function serialize(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $filters = (array) ($request->input('filters', []));
        $geometries = (array) ($request->input('geometries', []));

        $payload = [
            'f' => $this->encodeFilters($filters),
        ];
        if ($geometries !== []) {
            $payload['g'] = $this->encodeGeometries($geometries);
        }

        $query = http_build_query($payload, '', '&', PHP_QUERY_RFC3986);
        $url = $request->getSchemeAndHttpHost().'/agent/searches/shared?'.$query;

        return response()->json([
            'data' => [
                'url' => $url,
                'params' => $payload,
            ],
        ]);
    }

    public function sharedSearch(
        Request $request,
        MultiMlsQueryCompilerInterface $compiler,
        SearchExecutionBrokerInterface $broker,
        SubscriberFeedAccessService $feeds
    ): JsonResponse {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $encodedFilters = (string) ($request->query('f', ''));
        $encodedGeometries = (string) ($request->query('g', ''));

        $filters = $this->decodeFilters($encodedFilters);
        $geometries = $this->decodeGeometries($encodedGeometries);
        $geometryErrors = app(AgentSearchGeometrySafeguardService::class)->validate($geometries);
        if ($geometryErrors !== []) {
            return response()->json(['errors' => $geometryErrors], 422);
        }

        $scope = $feeds->resolvedSearchScopesForUser($user);
        $errors = $compiler->validate($filters, $scope);
        if ($errors !== []) {
            return response()->json(['errors' => $errors], 422);
        }

        $compiled = $compiler->compile($filters, $scope);
        foreach ($compiled as $key => $query) {
            if (! is_array($query)) {
                continue;
            }
            $compiled[$key]['geometries'] = $geometries;
        }
        $cacheKey = $this->queryPlanCacheKey($user->id, $filters, $scope, $geometries, 'shared');
        $cacheHit = Cache::has($cacheKey);
        $merged = Cache::remember($cacheKey, now()->addSeconds(90), function () use ($broker, $compiled): array {
            $raw = $broker->execute($compiled);

            return $broker->merge($raw);
        });

        $resolvedTrigger = $this->inferSearchExecuteTrigger($filters);
        $this->recordSearchExecutedActivity(
            $user,
            $merged,
            $filters,
            $geometries,
            $cacheHit,
            count($compiled),
            count($scope),
            $resolvedTrigger,
            'shared',
            null,
        );

        return response()->json([
            'data' => $merged,
            'meta' => [
                'compiled_query_count' => count($compiled),
                'scope_count' => count($scope),
                'shared' => true,
                'cache_hit' => $cacheHit,
            ],
        ]);
    }

    /**
     * @param  list<array<string, mixed>>  $filters
     */
    private function encodeFilters(array $filters): string
    {
        $parts = [];
        foreach ($filters as $filter) {
            if (! is_array($filter)) {
                continue;
            }
            $field = (string) ($filter['field'] ?? '');
            $op = (string) ($filter['operator'] ?? 'eq');
            $value = $filter['value'] ?? '';
            if ($field === '') {
                continue;
            }
            $parts[] = $field.'|'.$op.'|'.str_replace('|', '\\|', (string) $value);
        }

        return base64_encode(implode(';', $parts));
    }

    /**
     * @return list<array<string, mixed>>
     */
    private function decodeFilters(string $encoded): array
    {
        $decoded = base64_decode($encoded, true);
        if ($decoded === false || $decoded === '') {
            return [];
        }

        $filters = [];
        foreach (explode(';', $decoded) as $segment) {
            if ($segment === '') {
                continue;
            }
            $segments = explode('|', $segment, 3);
            if (count($segments) < 3) {
                continue;
            }
            $field = $segments[0];
            $op = $segments[1];
            $value = str_replace('\\|', '|', $segments[2]);

            if (is_numeric($value)) {
                $value = strpos($value, '.') !== false ? (float) $value : (int) $value;
            }

            $filters[] = ['field' => $field, 'operator' => $op, 'value' => $value];
        }

        return $filters;
    }

    /**
     * @param  list<array<string, mixed>>  $geometries
     */
    private function encodeGeometries(array $geometries): string
    {
        return base64_encode(json_encode($geometries, JSON_THROW_ON_ERROR) ?: '[]');
    }

    /**
     * Proxy geocoding to Nominatim with a server-side User-Agent (browser fetch cannot set it).
     *
     * @see https://operations.osmfoundation.org/policies/nominatim/
     */
    public function geocode(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $q = trim((string) $request->query('q', ''));
        if ($q === '' || strlen($q) > 200) {
            return response()->json([
                'message' => 'Geocode query is required (max 200 characters).',
                'errors' => [
                    'q' => ['Provide a non-empty query up to 200 characters.'],
                ],
            ], 422);
        }

        $limit = (int) $request->query('limit', 5);
        $limit = min(10, max(1, $limit));

        $cacheTtl = max(0, (int) config('geocoding.geocode_cache_ttl_seconds', 120));
        $cacheKey = 'idx:agent-geocode:v1:'.sha1(strtolower($q).'|'.$limit);

        if ($cacheTtl > 0) {
            /** @var mixed $cached */
            $cached = Cache::get($cacheKey);
            if (is_array($cached)) {
                return response()->json([
                    'data' => $cached,
                    'meta' => ['cached' => true],
                ]);
            }
        }

        $rateKey = 'agent-geocode:user:'.$user->id;
        if (RateLimiter::tooManyAttempts($rateKey, 30)) {
            return response()->json([
                'message' => 'Too many geocode requests. Try again in a minute.',
            ], 429);
        }

        RateLimiter::hit($rateKey, 60);

        $url = (string) config('geocoding.nominatim_url');
        $userAgent = (string) config('geocoding.nominatim_user_agent');
        $timeout = (int) config('geocoding.timeout_seconds', 10);

        try {
            $response = Http::timeout($timeout)
                ->withHeaders([
                    'User-Agent' => $userAgent,
                    'Accept' => 'application/json',
                ])
                ->get($url, [
                    'format' => 'json',
                    'q' => $q,
                    'limit' => $limit,
                ]);
        } catch (\Throwable) {
            return response()->json(['message' => 'Geocode service unavailable.'], 503);
        }

        if (! $response->successful()) {
            return response()->json(['message' => 'Upstream geocoder rejected the request.'], 502);
        }

        /** @var mixed $payload */
        $payload = $response->json();
        $data = is_array($payload) ? $payload : [];

        if ($cacheTtl > 0) {
            Cache::put($cacheKey, $data, now()->addSeconds($cacheTtl));
        }

        return response()->json([
            'data' => $data,
            'meta' => ['cached' => false],
        ]);
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function fieldCatalog(
        Request $request,
        FieldCatalogService $catalogService,
        SubscriberFeedAccessService $feeds
    ): JsonResponse {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $scope = $feeds->resolvedSearchScopesForUser($user);
        $category = (string) ($request->query('category', ''));

        $fields = $catalogService->getFieldsForScope($scope, $category !== '' ? $category : null);
        $categories = $catalogService->getCategories();

        return response()->json([
            'data' => [
                'fields' => $fields,
                'categories' => $categories,
            ],
        ]);
    }

    private function decodeGeometries(string $encoded): array
    {
        if ($encoded === '') {
            return [];
        }
        $decoded = base64_decode($encoded, true);
        if ($decoded === false) {
            return [];
        }

        $data = json_decode($decoded, true, 512, JSON_THROW_ON_ERROR);

        return is_array($data) ? $data : [];
    }

    /**
     * @param  list<array<string, mixed>>  $filters
     */
    private function inferSearchExecuteTrigger(array $filters): string
    {
        foreach ($filters as $filter) {
            if (! is_array($filter)) {
                continue;
            }
            if (($filter['field'] ?? '') === 'location.viewport' && ($filter['operator'] ?? '') === 'bbox') {
                return 'viewport';
            }
        }

        return 'manual';
    }

    /**
     * @param  list<array<string, mixed>>  $filters
     * @param  list<array<string, mixed>>  $geometries
     * @param  array<string, mixed>  $merged
     */
    private function recordSearchExecutedActivity(
        User $user,
        array $merged,
        array $filters,
        array $geometries,
        bool $cacheHit,
        int $compiledQueryCount,
        int $scopeCount,
        string $trigger,
        string $surface,
        ?int $agentSearchId,
    ): void {
        $meta = is_array($merged['meta'] ?? null) ? $merged['meta'] : [];
        $total = (int) ($meta['total_items'] ?? count($merged['items'] ?? []));
        $sources = (int) ($meta['sources'] ?? 0);

        app(AgentActivityTracker::class)->record(
            $user,
            'search_execute',
            sprintf(
                'Search executed: %d listing%s, %d source%s',
                $total,
                $total === 1 ? '' : 's',
                $sources,
                $sources === 1 ? '' : 's',
            ),
            'completed',
            [
                'trigger' => $trigger,
                'surface' => $surface,
                'total_items' => $total,
                'sources' => $sources,
                'geometry_count' => count($geometries),
                'filter_count' => count($filters),
                'cache_hit' => $cacheHit,
                'compiled_query_count' => $compiledQueryCount,
                'scope_count' => $scopeCount,
                'agent_search_id' => $agentSearchId,
            ],
        );
    }

    /**
     * @param  array<string, mixed>  $scope
     * @param  list<array<string, mixed>>  $filters
     * @param  list<array<string, mixed>>  $geometries
     */
    private function queryPlanCacheKey(
        int $userId,
        array $filters,
        array $scope,
        array $geometries,
        string $prefix = 'execute'
    ): string {
        $serialized = json_encode([
            'filters' => $filters,
            'scope' => $scope,
            'geometries' => $geometries,
        ], JSON_THROW_ON_ERROR);

        return sprintf('idx:query-plan:%s:%d:%s', $prefix, $userId, sha1($serialized));
    }
}
