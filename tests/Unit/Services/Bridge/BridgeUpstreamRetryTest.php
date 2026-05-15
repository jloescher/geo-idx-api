<?php

declare(strict_types=1);

namespace Tests\Unit\Services\Bridge;

use App\Services\Bridge\BridgeUpstreamRetry;
use GuzzleHttp\Psr7\Response as Psr7Response;
use Illuminate\Http\Client\Response;
use Tests\TestCase;

class BridgeUpstreamRetryTest extends TestCase
{
    public function test_returns_immediately_on_success(): void
    {
        config(['bridge.sync_max_http_retries' => 4]);
        $calls = 0;
        $out = BridgeUpstreamRetry::run(function () use (&$calls) {
            $calls++;

            return new Response(new Psr7Response(200, [], '{}'));
        });
        $this->assertSame(200, $out->status());
        $this->assertSame(1, $calls);
    }

    public function test_retries_on_429_until_200(): void
    {
        config(['bridge.sync_max_http_retries' => 1]);
        $calls = 0;
        $out = BridgeUpstreamRetry::run(function () use (&$calls) {
            $calls++;
            if ($calls === 1) {
                return new Response(new Psr7Response(429, ['Retry-After' => '0'], ''));
            }

            return new Response(new Psr7Response(200, [], '{"ok":true}'));
        });
        $this->assertSame(200, $out->status());
        $this->assertSame(2, $calls);
    }

    public function test_retries_on_503_until_200(): void
    {
        config(['bridge.sync_max_http_retries' => 1]);
        $calls = 0;
        $out = BridgeUpstreamRetry::run(function () use (&$calls) {
            $calls++;
            if ($calls === 1) {
                return new Response(new Psr7Response(503, ['Retry-After' => '0'], ''));
            }

            return new Response(new Psr7Response(200, [], '{}'));
        });
        $this->assertSame(200, $out->status());
        $this->assertSame(2, $calls);
    }

    public function test_returns_last_response_when_attempts_exhausted(): void
    {
        config(['bridge.sync_max_http_retries' => 0]);
        $calls = 0;
        $out = BridgeUpstreamRetry::run(function () use (&$calls) {
            $calls++;

            return new Response(new Psr7Response(429, ['Retry-After' => '0'], ''));
        });
        $this->assertSame(429, $out->status());
        $this->assertSame(1, $calls);
    }
}
