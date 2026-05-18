<?php

declare(strict_types=1);

namespace App\Services\Spark;

use Illuminate\Http\Client\Response;

final class SparkUpstreamRetry
{
    /**
     * @param  callable(): Response  $send
     */
    public static function run(callable $send): Response
    {
        $attempts = max(1, (int) config('spark.sync_max_http_retries', 4) + 1);
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
            usleep(random_int(0, 250) * 1000);
        }

        return $last ?? $send();
    }
}
