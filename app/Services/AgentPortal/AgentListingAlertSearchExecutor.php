<?php

namespace App\Services\AgentPortal;

use App\Models\AgentSearch;
use App\Models\AgentSearchFilter;
use App\Models\AgentSearchGeometry;
use App\Models\User;
use App\Services\AgentPortal\Contracts\MultiMlsQueryCompilerInterface;
use App\Services\AgentPortal\Contracts\SearchExecutionBrokerInterface;

/**
 * Runs a saved agent search against Bridge for scheduled listing alerts (no HTTP response cache).
 */
final class AgentListingAlertSearchExecutor
{
    public function __construct(
        private readonly MultiMlsQueryCompilerInterface $compiler,
        private readonly SearchExecutionBrokerInterface $broker,
        private readonly SubscriberFeedAccessService $feeds,
        private readonly AgentSearchGeometrySafeguardService $geometrySafeguards,
    ) {}

    /**
     * @return array{success: true, merged: array<string, mixed>}|array{success: false, reason: string, errors?: array<string, mixed>}
     */
    public function executeForOwner(User $owner, AgentSearch $search): array
    {
        $search->loadMissing(['filters', 'geometries']);

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

        if ($filters === [] && $geometries === []) {
            return ['success' => false, 'reason' => 'empty_criteria'];
        }

        $geometryErrors = $this->geometrySafeguards->validate($geometries);
        if ($geometryErrors !== []) {
            return ['success' => false, 'reason' => 'geometry', 'errors' => $geometryErrors];
        }

        $scope = $this->feeds->resolvedSearchScopesForUser($owner);
        $errors = $this->compiler->validate($filters, $scope);
        if ($errors !== []) {
            return ['success' => false, 'reason' => 'validation', 'errors' => $errors];
        }

        $compiled = $this->compiler->compile($filters, $scope);
        foreach ($compiled as $key => $query) {
            if (! is_array($query)) {
                continue;
            }
            $compiled[$key]['geometries'] = $geometries;
        }

        $raw = $this->broker->execute($compiled);
        $merged = $this->broker->merge($raw);

        return ['success' => true, 'merged' => $merged];
    }
}
