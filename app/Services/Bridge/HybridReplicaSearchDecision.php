<?php

namespace App\Services\Bridge;

use App\Http\Requests\Search\SearchRequest;

/**
 * Revenue impact: a single classifier prevents accidental Bridge bypass for historically indexed queries.
 */
final class HybridReplicaSearchDecision
{
    /**
     * @param  array<string, mixed>  $validated  From {@see SearchRequest::validated()}
     */
    public function prefersPostgresReplica(array $validated): bool
    {
        if (isset($validated['price_reduced_within_days'])) {
            return false;
        }

        $statuses = $validated['statuses'] ?? [];
        if (! is_array($statuses)) {
            $statuses = [];
        }

        foreach ($statuses as $s) {
            $sl = strtolower((string) $s);
            if (! in_array($sl, ['active', 'pending'], true)) {
                return false;
            }
        }

        $activeOnly = $validated['active_only'] ?? true;
        if (! $activeOnly && $statuses === []) {
            return false;
        }

        return true;
    }

    /**
     * @param  array<string, mixed>  $validated
     */
    public function geoEmptyShouldRetryBridge(array $validated, int $localResultCount): bool
    {
        if ($localResultCount !== 0) {
            return false;
        }

        return isset($validated['geo']['distance']['radius_miles'])
            || isset($validated['geo']['bbox']['west']);
    }
}
