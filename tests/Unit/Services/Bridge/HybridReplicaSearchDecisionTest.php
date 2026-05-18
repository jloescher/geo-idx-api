<?php

namespace Tests\Unit\Services\Bridge;

use App\Services\Bridge\HybridReplicaSearchDecision;
use App\Services\Bridge\HybridSearchRouteMode;
use PHPUnit\Framework\TestCase;

class HybridReplicaSearchDecisionTest extends TestCase
{
    public function test_price_reduced_routes_to_bridge_only(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertSame(
            HybridSearchRouteMode::BridgeOnly,
            $decision->routeMode(['price_reduced_within_days' => 14, 'page' => ['limit' => 24]]),
        );
    }

    public function test_closed_only_routes_to_bridge(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertSame(
            HybridSearchRouteMode::BridgeOnly,
            $decision->routeMode(['statuses' => ['Closed']]),
        );
    }

    public function test_active_pending_routes_to_postgres_only(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertTrue($decision->prefersPostgresReplica([]));
        $this->assertSame(
            HybridSearchRouteMode::PostgresOnly,
            $decision->routeMode(['statuses' => ['Active', 'Pending']]),
        );
    }

    public function test_mixed_active_pending_closed_routes_to_split(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertSame(
            HybridSearchRouteMode::Split,
            $decision->routeMode(['statuses' => ['Active', 'Pending', 'Closed']]),
        );
    }

    public function test_non_replica_status_routes_to_bridge_only(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertSame(
            HybridSearchRouteMode::BridgeOnly,
            $decision->routeMode(['statuses' => ['Active', 'Coming Soon']]),
        );
    }

    public function test_active_only_disabled_without_statuses_requires_bridge(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertSame(
            HybridSearchRouteMode::BridgeOnly,
            $decision->routeMode(['active_only' => false]),
        );
    }

    public function test_geo_retry_when_local_empty_has_geo_radius(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertTrue($decision->geoEmptyShouldRetryBridge([
            'geo' => ['distance' => ['lat' => 1, 'lng' => 2, 'radius_miles' => 3]],
        ], 0));
        $this->assertFalse($decision->geoEmptyShouldRetryBridge([
            'geo' => ['distance' => ['lat' => 1, 'lng' => 2, 'radius_miles' => 3]],
        ], 2));
        $this->assertFalse($decision->geoEmptyShouldRetryBridge([], 0));
    }

    public function test_geo_retry_disabled_for_split_mode(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertFalse($decision->geoEmptyShouldRetryBridge([
            'statuses' => ['Active', 'Closed'],
            'geo' => ['distance' => ['lat' => 1, 'lng' => 2, 'radius_miles' => 3]],
        ], 0));
    }
}
