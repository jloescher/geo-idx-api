<?php

namespace Tests\Unit\Bridge;

use App\Services\Bridge\BridgeSyncService;
use ReflectionMethod;
use Tests\TestCase;

class BridgeSyncSelectListTest extends TestCase
{
    public function test_sync_select_omits_media_when_include_media_is_false(): void
    {
        config(['bridge.sync_include_media' => false]);

        $select = $this->syncSelectList('stellar');

        $this->assertStringNotContainsString('Media', $select);
        $this->assertStringNotContainsString('DockYN', $select);
    }

    public function test_sync_select_includes_media_when_include_media_is_true(): void
    {
        config(['bridge.sync_include_media' => true]);

        $select = $this->syncSelectList('stellar');

        $this->assertStringContainsString('Media', $select);
    }

    private function syncSelectList(string $dataset): string
    {
        $method = new ReflectionMethod(BridgeSyncService::class, 'syncSelectList');
        $service = app(BridgeSyncService::class);

        return $method->invoke($service, $dataset);
    }
}
