<?php

namespace Tests\Unit\Support;

use App\Support\BridgeSyncQueueNames;
use PHPUnit\Framework\Attributes\DataProvider;
use Tests\TestCase;

class BridgeSyncQueueNamesTest extends TestCase
{
    protected function tearDown(): void
    {
        putenv('BRIDGE_SYNC_FETCH_QUEUE');
        putenv('BRIDGE_SYNC_QUEUE');
        putenv('BRIDGE_SYNC_PERSIST_QUEUE');

        parent::tearDown();
    }

    public function test_legacy_monolithic_queue_env_maps_to_fetch_queue(): void
    {
        putenv('BRIDGE_SYNC_QUEUE=bridge-sync');
        putenv('BRIDGE_SYNC_FETCH_QUEUE');

        $this->assertSame('bridge-sync-fetch', BridgeSyncQueueNames::fetchQueue());
    }

    public function test_explicit_fetch_queue_wins_over_legacy(): void
    {
        putenv('BRIDGE_SYNC_QUEUE=bridge-sync');
        putenv('BRIDGE_SYNC_FETCH_QUEUE=custom-fetch');

        $this->assertSame('custom-fetch', BridgeSyncQueueNames::fetchQueue());
    }

    #[DataProvider('persistQueueProvider')]
    public function test_persist_queue_resolution(?string $env, string $expected): void
    {
        if ($env === null) {
            putenv('BRIDGE_SYNC_PERSIST_QUEUE');
        } else {
            putenv('BRIDGE_SYNC_PERSIST_QUEUE='.$env);
        }

        $this->assertSame($expected, BridgeSyncQueueNames::persistQueue());
    }

    /**
     * @return array<string, array{0: ?string, 1: string}>
     */
    public static function persistQueueProvider(): array
    {
        return [
            'default' => [null, 'bridge-sync-persist'],
            'explicit' => ['custom-persist', 'custom-persist'],
        ];
    }
}
