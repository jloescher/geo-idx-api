<?php

declare(strict_types=1);

namespace App\Services\Bridge;

use Illuminate\Http\Client\Response;

/**
 * Revenue impact: shared 429/503 handling keeps Bridge proxy + hybrid search resilient without
 * duplicating backoff math; only upstream Bridge responses trigger waits (not idx-api clients).
 */
final class BridgeUpstreamRetry
{
    /**
     * Retry on HTTP 429 / 503 with exponential backoff, optional Retry-After (seconds), and jitter.
     * Uses {@see config('bridge.sync_max_http_retries')} for all outbound Bridge HTTP (sync + proxy).
     *
     * @param  callable(): Response  $send
     */
    public static function run(callable $send): Response
    {
        $attempts = max(1, (int) config('bridge.sync_max_http_retries', 4) + 1);
        $delayMs = 500;
        $last = null;
        for ($i = 0; $i < $attempts; $i++) {
            $last = $send();
            if ($last->status() !== 429 && $last->status() !== 503) {
                return $last;
            }
            $retryAfter = $last->header('Retry-After');
            $sleep = is_numeric($retryAfter) ? (int) $retryAfter * 1000 : $delayMs;
            usleep(min(30_000_000, $sleep * 1000));
            $delayMs = min(8000, (int) ($delayMs * 1.8));
            $jitter = random_int(0, 250);
            usleep($jitter * 1000);
        }

        return $last ?? $send();
    }
}
