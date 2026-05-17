<?php

namespace App\Jobs;

use App\Services\CoinGeckoPricingService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\Attributes\Backoff;
use Illuminate\Queue\Attributes\Tries;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Throwable;

#[Tries(3)]
#[Backoff([30, 120, 300])]
class RefreshCryptoPricingJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    /**
     * Revenue impact: queueing refreshes keeps request workers focused on serving
     * listings while still maintaining fresh conversion rates in cache and DB.
     */
    public function handle(CoinGeckoPricingService $pricing): void
    {
        $pricing->refresh();
    }

    public function failed(?Throwable $exception): void
    {
        if ($exception !== null) {
            report($exception);
        }
    }
}
