<?php

namespace App\Services\Bridge;

use App\Http\Requests\Search\SearchRequest;

/**
 * Revenue impact: a single classifier prevents accidental Bridge bypass for historically indexed queries
 * and enables split Active/Pending (Postgres) + Closed (Bridge) searches without duplicating MLS egress.
 */
final class HybridReplicaSearchDecision
{
    /**
     * @param  array<string, mixed>  $validated  From {@see SearchRequest::validated()}
     */
    public function routeMode(array $validated): HybridSearchRouteMode
    {
        if (isset($validated['price_reduced_within_days'])) {
            return HybridSearchRouteMode::BridgeOnly;
        }

        $statuses = $this->normalizeStatuses($validated);
        $replicaStatuses = array_values(array_intersect($statuses, ['active', 'pending']));
        $hasClosed = in_array('closed', $statuses, true);
        $hasNonReplica = $this->hasNonReplicaStatuses($statuses);

        if ($statuses === []) {
            return ($validated['active_only'] ?? true)
                ? HybridSearchRouteMode::PostgresOnly
                : HybridSearchRouteMode::BridgeOnly;
        }

        if ($hasNonReplica) {
            return HybridSearchRouteMode::BridgeOnly;
        }

        if ($hasClosed && $replicaStatuses !== []) {
            return HybridSearchRouteMode::Split;
        }

        if ($hasClosed) {
            return HybridSearchRouteMode::BridgeOnly;
        }

        return HybridSearchRouteMode::PostgresOnly;
    }

    /**
     * @param  array<string, mixed>  $validated
     */
    public function prefersPostgresReplica(array $validated): bool
    {
        return $this->routeMode($validated) === HybridSearchRouteMode::PostgresOnly;
    }

    /**
     * @param  array<string, mixed>  $validated
     * @return list<string> Lowercase RESO statuses for the local mirror leg (active, pending).
     */
    public function replicaStatusesForSplit(array $validated): array
    {
        $statuses = $this->normalizeStatuses($validated);
        $replica = array_values(array_intersect($statuses, ['active', 'pending']));

        if ($replica !== []) {
            return $replica;
        }

        if ($statuses === [] && ($validated['active_only'] ?? true)) {
            return ['active'];
        }

        return ['active', 'pending'];
    }

    /**
     * @param  array<string, mixed>  $validated
     */
    public function geoEmptyShouldRetryBridge(array $validated, int $localResultCount): bool
    {
        if ($localResultCount !== 0) {
            return false;
        }

        if ($this->routeMode($validated) === HybridSearchRouteMode::Split) {
            return false;
        }

        return isset($validated['geo']['distance']['radius_miles'])
            || isset($validated['geo']['bbox']['west']);
    }

    /**
     * @param  array<string, mixed>  $validated
     * @return list<string>
     */
    private function normalizeStatuses(array $validated): array
    {
        $statuses = $validated['statuses'] ?? [];
        if (! is_array($statuses)) {
            return [];
        }

        $normalized = [];
        foreach ($statuses as $status) {
            $slug = strtolower(trim((string) $status));
            if ($slug !== '') {
                $normalized[] = $slug;
            }
        }

        return array_values(array_unique($normalized));
    }

    /**
     * @param  list<string>  $normalizedStatuses
     */
    private function hasNonReplicaStatuses(array $normalizedStatuses): bool
    {
        foreach ($normalizedStatuses as $status) {
            if (! in_array($status, ['active', 'pending', 'closed'], true)) {
                return true;
            }
        }

        return false;
    }
}
