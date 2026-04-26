<?php

namespace Tests\Unit\Services\Bridge;

use App\Services\Bridge\ListingsCacheService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class ListingsCacheServiceSearchTest extends TestCase
{
    use RefreshDatabase;

    public function test_remember_search_invokes_supplier_once_per_partition_and_fingerprint(): void
    {
        $svc = app(ListingsCacheService::class);
        $calls = 0;
        $fingerprint = str_repeat('a', 64);

        $first = $svc->rememberSearchResult('part-key', $fingerprint, function () use (&$calls): array {
            $calls++;

            return ['value' => [['ListingKey' => 'stellar:1']], 'count' => 1, 'nextLink' => null];
        });
        $second = $svc->rememberSearchResult('part-key', $fingerprint, function () use (&$calls): array {
            $calls++;

            return ['value' => [], 'count' => 0, 'nextLink' => null];
        });

        $this->assertSame(1, $calls);
        $this->assertSame('stellar:1', $first['value'][0]['ListingKey'] ?? null);
        $this->assertSame('stellar:1', $second['value'][0]['ListingKey'] ?? null);
    }
}
