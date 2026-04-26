<?php

namespace App\Services;

use App\Models\CryptoPriceSnapshot;
use Carbon\CarbonImmutable;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use RuntimeException;

class CoinGeckoPricingService
{
    /**
     * Refresh CoinGecko quotes and store DB/cache snapshots.
     *
     * @return array{
     *     quotes: array<string, array<string, float>>,
     *     as_of: string,
     *     status: string
     * }
     */
    public function refresh(): array
    {
        $assetIds = $this->assetIds();
        $vsCurrencies = $this->vsCurrencies();
        $asOf = CarbonImmutable::now();

        $query = [
            'ids' => implode(',', $assetIds),
            'vs_currencies' => implode(',', $vsCurrencies),
        ];

        $request = Http::baseUrl((string) config('coingecko.base_url'))
            ->connectTimeout((int) config('coingecko.http_connect_timeout_seconds'))
            ->timeout((int) config('coingecko.http_timeout_seconds'))
            ->acceptJson()
            ->retry([150, 300], throw: false);

        $apiKey = (string) config('coingecko.api_key');
        if ($apiKey !== '') {
            $request = $request->withHeaders([
                'x-cg-demo-api-key' => $apiKey,
            ]);
        }

        $response = $request->get('/simple/price', $query);
        if (! $response->successful()) {
            throw new RuntimeException('CoinGecko pricing request failed with status '.$response->status());
        }

        $decoded = $response->json();
        if (! is_array($decoded)) {
            throw new RuntimeException('CoinGecko pricing payload is invalid.');
        }

        $quotes = $this->normalizeQuotes($decoded, $assetIds, $vsCurrencies);
        $snapshot = [
            'quotes' => $quotes,
            'as_of' => $asOf->toIso8601String(),
            'status' => 'ok',
        ];

        $this->persistQuotes($quotes, $asOf, $decoded);
        Cache::put((string) config('coingecko.cache_key'), $snapshot, now()->addSeconds((int) config('coingecko.cache_ttl_seconds')));

        return $snapshot;
    }

    /**
     * @return array{quotes: array<string, array<string, float>>, as_of: string, status: string}|null
     */
    public function latest(): ?array
    {
        $cached = Cache::get((string) config('coingecko.cache_key'));
        if (is_array($cached) && isset($cached['quotes']) && is_array($cached['quotes'])) {
            return $cached;
        }

        $rows = CryptoPriceSnapshot::query()
            ->select(['asset_id', 'vs_currency', 'price', 'as_of'])
            ->get();

        if ($rows->isEmpty()) {
            return null;
        }

        $quotes = [];
        $latestAsOf = null;
        foreach ($rows as $row) {
            $assetId = strtolower((string) $row->asset_id);
            $vsCurrency = strtolower((string) $row->vs_currency);
            $quotes[$assetId][$vsCurrency] = (float) $row->price;
            $latestAsOf = $latestAsOf === null || $row->as_of->greaterThan($latestAsOf) ? $row->as_of : $latestAsOf;
        }

        if ($latestAsOf === null) {
            return null;
        }

        $snapshot = [
            'quotes' => $quotes,
            'as_of' => $latestAsOf->toIso8601String(),
            'status' => 'ok',
        ];

        Cache::put((string) config('coingecko.cache_key'), $snapshot, now()->addSeconds((int) config('coingecko.cache_ttl_seconds')));

        return $snapshot;
    }

    /**
     * @param  array<string, mixed>  $decoded
     * @param  list<string>  $assetIds
     * @param  list<string>  $vsCurrencies
     * @return array<string, array<string, float>>
     */
    private function normalizeQuotes(array $decoded, array $assetIds, array $vsCurrencies): array
    {
        $quotes = [];
        foreach ($assetIds as $assetId) {
            $row = $decoded[$assetId] ?? null;
            if (! is_array($row)) {
                throw new RuntimeException('CoinGecko payload missing asset row: '.$assetId);
            }

            foreach ($vsCurrencies as $currency) {
                $value = $row[$currency] ?? null;
                if (! is_numeric($value)) {
                    throw new RuntimeException("CoinGecko payload missing quote for {$assetId}/{$currency}");
                }
                $quotes[$assetId][$currency] = (float) $value;
            }
        }

        return $quotes;
    }

    /**
     * @param  array<string, array<string, float>>  $quotes
     * @param  array<string, mixed>  $rawPayload
     */
    private function persistQuotes(array $quotes, CarbonImmutable $asOf, array $rawPayload): void
    {
        $rows = [];
        foreach ($quotes as $assetId => $currencyQuotes) {
            foreach ($currencyQuotes as $vsCurrency => $price) {
                $rows[] = [
                    'asset_id' => $assetId,
                    'vs_currency' => $vsCurrency,
                    'price' => $price,
                    'as_of' => $asOf,
                    'payload' => isset($rawPayload[$assetId]) ? json_encode($rawPayload[$assetId], JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) : null,
                    'updated_at' => now(),
                    'created_at' => now(),
                ];
            }
        }

        CryptoPriceSnapshot::query()->upsert(
            $rows,
            ['asset_id', 'vs_currency'],
            ['price', 'as_of', 'payload', 'updated_at'],
        );
    }

    /**
     * @return list<string>
     */
    private function assetIds(): array
    {
        return array_values(array_map(
            static fn (string $asset): string => strtolower(trim($asset)),
            (array) config('coingecko.asset_ids', [])
        ));
    }

    /**
     * @return list<string>
     */
    private function vsCurrencies(): array
    {
        return array_values(array_map(
            static fn (string $currency): string => strtolower(trim($currency)),
            (array) config('coingecko.vs_currencies', [])
        ));
    }
}
