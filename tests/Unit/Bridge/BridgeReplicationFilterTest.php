<?php

namespace Tests\Unit\Bridge;

use App\Services\Bridge\BridgeSyncService;
use ReflectionMethod;
use Tests\TestCase;

class BridgeReplicationFilterTest extends TestCase
{
    public function test_replication_active_pending_filter_matches_mls_cache_expression(): void
    {
        $filter = $this->invokePrivate('replicationActivePendingFilter');

        $this->assertSame(
            "(StandardStatus eq 'Active' or StandardStatus eq 'Pending')",
            $filter,
        );
    }

    public function test_replication_select_list_always_includes_media(): void
    {
        config(['bridge.sync_include_media' => false]);

        $select = $this->invokePrivate('replicationSelectList', ['stellar']);

        $this->assertStringContainsString('Media', $select);
    }

    private function invokePrivate(string $method, array $args = []): mixed
    {
        $reflection = new ReflectionMethod(BridgeSyncService::class, $method);
        $service = app(BridgeSyncService::class);

        return $reflection->invoke($service, ...$args);
    }
}
