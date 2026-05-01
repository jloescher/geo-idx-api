<?php

namespace Tests\Unit\Services\Bridge;

use App\Services\Bridge\HybridReplicaSearchDecision;
use PHPUnit\Framework\TestCase;

class HybridReplicaSearchDecisionTest extends TestCase
{
    public function test_price_reduced_rejects_local_replica(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertFalse($decision->prefersPostgresReplica([
            'price_reduced_within_days' => 14,
            'page' => ['limit' => 24],
        ]));
    }

    public function test_closed_status_rejects_local_replica(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertFalse($decision->prefersPostgresReplica([
            'statuses' => ['Closed'],
        ]));
    }

    public function test_active_defaults_allow_local_when_other_rules_pass(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertTrue($decision->prefersPostgresReplica([]));
        $this->assertTrue($decision->prefersPostgresReplica([
            'statuses' => ['Active', 'Pending'],
        ]));
    }

    public function test_active_only_disabled_without_statuses_requires_bridge(): void
    {
        $decision = new HybridReplicaSearchDecision;
        $this->assertFalse($decision->prefersPostgresReplica([
            'active_only' => false,
        ]));
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
}
