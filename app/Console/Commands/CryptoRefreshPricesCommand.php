<?php

namespace App\Console\Commands;

use App\Services\CoinGeckoPricingService;
use Illuminate\Console\Command;

class CryptoRefreshPricesCommand extends Command
{
    protected $signature = 'crypto:refresh-prices';

    protected $description = 'Refresh cached CoinGecko quotes for supported digital assets and fiat currencies';

    public function handle(CoinGeckoPricingService $pricing): int
    {
        try {
            $snapshot = $pricing->refresh();
            $assetCount = is_array($snapshot['quotes'] ?? null) ? count($snapshot['quotes']) : 0;
            $this->info("CoinGecko pricing refresh completed for {$assetCount} assets.");

            return self::SUCCESS;
        } catch (\Throwable $exception) {
            report($exception);
            $this->error('CoinGecko pricing refresh failed: '.$exception->getMessage());

            return self::FAILURE;
        }
    }
}
