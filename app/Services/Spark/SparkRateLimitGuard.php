<?php

declare(strict_types=1);

namespace App\Services\Spark;

use Illuminate\Http\Client\Response;
use Illuminate\Support\Facades\Cache;

final class SparkRateLimitGuard
{
    private const CACHE_KEY = 'spark.sync.rate_limit_state';

    private const CACHE_TTL_SECONDS = 7200;

    public function acquire(): void
    {
        $minIntervalMs = $this->minFetchIntervalMs();
        $state = $this->state();
        $nowMs = (int) floor(microtime(true) * 1000);

        if ($minIntervalMs > 0 && isset($state['last_acquire_ms'])) {
            $elapsed = $nowMs - (int) $state['last_acquire_ms'];
            if ($elapsed < $minIntervalMs) {
                usleep(($minIntervalMs - $elapsed) * 1000);
                $nowMs = (int) floor(microtime(true) * 1000);
            }
        }

        $this->putState([
            'last_acquire_ms' => $nowMs,
            'extra_delay_ms' => 0,
        ]);
    }

    public function recordFromResponse(Response $response): void
    {
        if ($response->status() !== 429) {
            return;
        }

        $state = $this->state();
        $retryAfter = $response->header('Retry-After');
        $extraMs = is_numeric($retryAfter) ? (int) $retryAfter * 1000 : 5000;
        $state['extra_delay_ms'] = max((int) ($state['extra_delay_ms'] ?? 0), $extraMs);
        $this->putState($state);
    }

    public function delayMillisecondsForNextFetch(): int
    {
        $minIntervalMs = $this->minFetchIntervalMs();
        $delayMs = max(0, $minIntervalMs);
        $extraMs = (int) ($this->state()['extra_delay_ms'] ?? 0);
        if ($extraMs > 0) {
            $delayMs = max($delayMs, $extraMs);
        }

        return $delayMs;
    }

    private function minFetchIntervalMs(): int
    {
        $configured = max(0, (int) config('spark.sync_min_fetch_interval_ms', 500));
        $fromRps = (int) floor(1000 / max(1, (int) config('spark.sync_max_requests_per_second', 2)));

        return max($configured, $fromRps);
    }

    /**
     * @return array<string, mixed>
     */
    private function state(): array
    {
        $cached = Cache::get(self::CACHE_KEY);

        return is_array($cached) ? $cached : [];
    }

    /**
     * @param  array<string, mixed>  $state
     */
    private function putState(array $state): void
    {
        Cache::put(self::CACHE_KEY, $state, self::CACHE_TTL_SECONDS);
    }
}
