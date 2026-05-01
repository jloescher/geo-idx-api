<?php

namespace App\Http\Controllers\Marketing;

use App\Http\Controllers\Controller;
use App\Models\AgentSearch;
use App\Models\AgentSearchFilter;
use App\Models\AgentSearchGeometry;
use App\Models\AgentShareLink;
use App\Services\AgentPortal\Contracts\MultiMlsQueryCompilerInterface;
use App\Services\AgentPortal\Contracts\SearchExecutionBrokerInterface;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Str;
use Illuminate\View\View;

class SharedLinkController extends Controller
{
    public function show(Request $request, string $token, ?string $slug = null): View|RedirectResponse
    {
        $shareLink = $this->activeShareLink($token);
        $search = $shareLink->agentSearch()->with(['filters', 'geometries'])->first();
        $canonicalSlug = $this->canonicalSlug($search?->name);

        if ($canonicalSlug !== '' && $slug !== $canonicalSlug) {
            return redirect()->route('marketing.shared.show', [
                'token' => $token,
                'slug' => $canonicalSlug,
            ]);
        }

        $attribution = is_array($shareLink->attribution_json) ? $shareLink->attribution_json : [];
        foreach (['utm_source', 'utm_medium', 'utm_campaign', 'utm_term', 'utm_content'] as $utmKey) {
            $value = $request->query($utmKey);
            if (is_string($value) && trim($value) !== '') {
                $attribution[$utmKey] = trim($value);
            }
        }
        $attribution['visit_count'] = ((int) ($attribution['visit_count'] ?? 0)) + 1;
        $attribution['last_visited_at'] = now()->toIso8601String();
        $shareLink->forceFill(['attribution_json' => $attribution])->save();

        $canonicalUrl = rtrim((string) config('idx.platform_url'), '/').'/shared/'.$token.($canonicalSlug !== '' ? '/'.$canonicalSlug : '');
        $seoTitle = ($search?->name ?? 'Shared search').' | Quantyra IDX';
        $seoDescription = $this->buildSeoDescription($search);

        return view('marketing.shared-link', [
            'shareLink' => $shareLink->fresh(['agentSearch']),
            'search' => $search,
            'canonicalUrl' => $canonicalUrl,
            'seoTitle' => $seoTitle,
            'seoDescription' => $seoDescription,
            'robotsDirective' => $this->robotsDirective($request),
        ]);
    }

    public function execute(
        string $token,
        MultiMlsQueryCompilerInterface $compiler,
        SearchExecutionBrokerInterface $broker,
        SubscriberFeedAccessService $feeds
    ): JsonResponse {
        $shareLink = $this->activeShareLink($token);
        $search = $shareLink->agentSearch()->with(['filters', 'geometries'])->firstOrFail();
        $owner = $shareLink->user()->firstOrFail();

        $scope = $feeds->resolvedSearchScopesForUser($owner);
        $filters = $search->filters->map(fn (AgentSearchFilter $filter): array => [
            'field' => (string) $filter->canonical_field_key,
            'operator' => (string) $filter->operator,
            'value' => $filter->value_json,
        ])->values()->all();
        $geometries = $search->geometries->map(fn (AgentSearchGeometry $geometry): array => [
            'geometry_type' => (string) $geometry->geometry_type,
            'mode' => (string) $geometry->mode,
            'geojson' => (array) ($geometry->geojson ?? []),
        ])->values()->all();

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

        $raw = $broker->execute($compiled);
        $merged = $broker->merge($raw);

        return response()->json(['data' => $merged]);
    }

    private function activeShareLink(string $token): AgentShareLink
    {
        return AgentShareLink::query()
            ->where('token', $token)
            ->where(function ($query): void {
                $query->whereNull('expires_at')
                    ->orWhere('expires_at', '>', now());
            })
            ->firstOrFail();
    }

    private function canonicalSlug(?string $name): string
    {
        return $name !== null ? Str::slug($name) : '';
    }

    private function robotsDirective(Request $request): string
    {
        $allowedTrackingKeys = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_term', 'utm_content'];
        $queryKeys = array_keys($request->query());
        $nonTracking = array_diff($queryKeys, $allowedTrackingKeys);

        return $nonTracking === [] ? 'index,follow' : 'noindex,follow';
    }

    private function buildSeoDescription(?AgentSearch $search): string
    {
        if ($search === null) {
            return 'Browse shared listings and discover homes matched to this curated search.';
        }

        $fragments = [];
        foreach ($search->filters->take(3) as $filter) {
            if (! is_string($filter->canonical_field_key) || $filter->canonical_field_key === '') {
                continue;
            }
            $value = is_scalar($filter->value_json) ? (string) $filter->value_json : null;
            if ($value === null || $value === '') {
                continue;
            }
            $fragments[] = sprintf('%s %s %s', $filter->canonical_field_key, $filter->operator, $value);
        }

        if ($fragments === []) {
            return 'Browse shared listings and discover homes matched to this curated search.';
        }

        return 'Browse shared listings tuned to filters: '.implode('; ', $fragments).'.';
    }
}
