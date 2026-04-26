<?php

namespace Tests\Feature;

use App\Jobs\RefreshCryptoPricingJob;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Queue;
use Tests\TestCase;

class CryptoPricingScheduleTest extends TestCase
{
    use RefreshDatabase;

    public function test_schedule_run_dispatches_crypto_pricing_refresh_job(): void
    {
        config(['coingecko.queue' => 'default']);
        Queue::fake();

        Carbon::setTestNow(Carbon::create(2026, 4, 27, 12, 10, 0, 'UTC'));

        $this->artisan('schedule:run')->assertSuccessful();

        Queue::assertPushed(RefreshCryptoPricingJob::class);
        Queue::assertPushedOn('default', RefreshCryptoPricingJob::class);

        Carbon::setTestNow();
    }
}
