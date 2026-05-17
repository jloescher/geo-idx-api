<?php

declare(strict_types=1);

namespace App\Services\Bridge;

use Illuminate\Http\Client\Response;
use Illuminate\Support\Facades\Cache;

/**
 * Revenue impact: proactive Bridge request budgeting avoids 429 suspensions during replication
 * catch-up while still maximizing throughput under the 334/min burst and 5k/hour quotas.
 */
final class BridgeRateLimitGuard
{
    private const CACHE_KEY = 'bridge.sync.rate_limit_state';

    private const CACHE_TTL_SECONDS = 120;

    public function acquire(): void
    {
        $maxPerMinute = max(1, (int) config('bridge.sync_max_requests_per_minute', 280));
        $minIntervalMs = max(0, (int) config('bridge.sync_min_fetch_interval_ms', 200));

        $state = $this->state();
        $nowMs = (int) floor(microtime(true) * 1000);

        if ($minIntervalMs > 0 && isset($state['last_acquire_ms'])) {
            $elapsed = $nowMs - (int) $state['last_acquire_ms'];
            if ($elapsed < $minIntervalMs) {
                usleep(($minIntervalMs - $elapsed) * 1000);
                $nowMs = (int) floor(microtime(true) * 1000);
            }
        }

        $bucket = $this->minuteBucket();
        $count = (int) (($state['minute_bucket'] ?? '') === $bucket ? ($state['request_count'] ?? 0) : 0);

        if ($count >= $maxPerMinute) {
            $sleepSeconds = 60 - (int) date('s');
            if ($sleepSeconds > 0) {
                sleep($sleepSeconds);
            }
            $bucket = $this->minuteBucket();
            $count = 0;
            $nowMs = (int) floor(microtime(true) * 1000);
        }

        $extraMs = (int) ($state['extra_delay_ms'] ?? 0);
        if ($extraMs > 0) {
            usleep(min(30_000_000, $extraMs * 1000));
            $nowMs = (int) floor(microtime(true) * 1000);
        }

        $this->putState([
            'minute_bucket' => $bucket,
            'request_count' => $count + 1,
            'last_acquire_ms' => $nowMs,
            'extra_delay_ms' => 0,
        ]);
    }

    public function recordFromResponse(Response $response): void
    {
        $state = $this->state();
        $extraMs = 0;

        $burstRemaining = $this->headerInt($response, 'Burst-RateLimit-Remaining');
        if ($burstRemaining !== null && $burstRemaining < 30) {
            $extraMs = max($extraMs, 2000);
        }

        $appRemaining = $this->headerInt($response, 'Application-RateLimit-Remaining');
        if ($appRemaining !== null && $appRemaining < 100) {
            $extraMs = max($extraMs, 1000);
        }

        if ($response->status() === 429) {
            $retryAfter = $response->header('Retry-After');
            if (is_numeric($retryAfter)) {
                $extraMs = max($extraMs, (int) $retryAfter * 1000);
            } else {
                $extraMs = max($extraMs, 5000);
            }
        }

        if ($extraMs > 0) {
            $state['extra_delay_ms'] = max((int) ($state['extra_delay_ms'] ?? 0), $extraMs);
            $this->putState($state);
        }
    }

    public function delaySecondsForNextFetch(): int
    {
        $minIntervalMs = max(0, (int) config('bridge.sync_min_fetch_interval_ms', 200));

        return (int) max(1, (int) ceil($minIntervalMs / 1000));
    }

    /**
     * @return array{minute_bucket?: string, request_count?: int, last_acquire_ms?: int, extra_delay_ms?: int}
     */
    private function state(): array
    {
        $cached = Cache::get(self::CACHE_KEY);

        return is_array($cached) ? $cached : [];
    }

    /**
     * @param  array{minute_bucket?: string, request_count?: int, last_acquire_ms?: int, extra_delay_ms?: int}  $state
     */
    private function putState(array $state): void
    {
        Cache::put(self::CACHE_KEY, $state, self::CACHE_TTL_SECONDS);
    }

    private function minuteBucket(): string
    {
        return date('Y-m-d-H-i');
    }

    private function headerInt(Response $response, string $name): ?int
    {
        $value = $response->header($name);
        if ($value === null || $value === '') {
            return null;
        }

        return is_numeric($value) ? (int) $value : null;
    }
}
