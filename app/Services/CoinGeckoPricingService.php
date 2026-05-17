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
        $coingeckoIds = $this->coingeckoApiIds($assetIds);
        $asOf = CarbonImmutable::now();

        $query = [
            'ids' => implode(',', array_values(array_unique($coingeckoIds))),
            'vs_currencies' => implode(',', $vsCurrencies),
        ];

        $request = Http::baseUrl((string) config('coingecko.base_url'))
            ->connectTimeout((int) config('coingecko.http_connect_timeout_seconds'))
            ->timeout((int) config('coingecko.http_timeout_seconds'))
            ->acceptJson()
            ->retry([150, 300], throw: false);

        $apiKey = (string) config('coingecko.api_key');
        if ($apiKey !== '') {
            $request = $request->withHeaders($this->apiKeyHeaders($apiKey));
        }

        $response = $request->get('/simple/price', $query);
        if (! $response->successful()) {
            throw new RuntimeException($this->httpFailureMessage($response->status(), $response->body()));
        }

        $decoded = $response->json();
        if (! is_array($decoded)) {
            throw new RuntimeException('CoinGecko pricing payload is invalid.');
        }

        $quotes = $this->normalizeQuotes($decoded, $assetIds, $coingeckoIds, $vsCurrencies);
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
     * @param  array<string, string>  $coingeckoIds  asset id => CoinGecko API id
     * @param  list<string>  $vsCurrencies
     * @return array<string, array<string, float>>
     */
    private function normalizeQuotes(array $decoded, array $assetIds, array $coingeckoIds, array $vsCurrencies): array
    {
        $quotes = [];
        foreach ($assetIds as $assetId) {
            $coingeckoId = $coingeckoIds[$assetId] ?? $assetId;
            $row = $decoded[$coingeckoId] ?? null;
            if (! is_array($row)) {
                throw new RuntimeException(
                    'CoinGecko payload missing asset row: '.$assetId.' (api id: '.$coingeckoId.')'
                );
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
     * @param  list<string>  $assetIds
     * @return array<string, string> internal asset id => CoinGecko API id
     */
    private function coingeckoApiIds(array $assetIds): array
    {
        /** @var array<string, string> $map */
        $map = (array) config('coingecko.coingecko_id_map', []);

        $resolved = [];
        foreach ($assetIds as $assetId) {
            $resolved[$assetId] = $map[$assetId] ?? $assetId;
        }

        return $resolved;
    }

    /**
     * @return array<string, string>
     */
    private function apiKeyHeaders(string $apiKey): array
    {
        $header = (string) config('coingecko.api_key_header', '');
        if ($header === '') {
            $baseUrl = (string) config('coingecko.base_url');
            $header = str_contains($baseUrl, 'pro-api.coingecko.com')
                ? 'x-cg-pro-api-key'
                : 'x-cg-demo-api-key';
        }

        return [$header => $apiKey];
    }

    private function httpFailureMessage(int $status, string $body): string
    {
        $snippet = strlen($body) > 240 ? substr($body, 0, 240).'…' : $body;
        $hint = $status === 429
            ? ' Set COINGECKO_API_KEY (demo or pro) for higher limits.'
            : '';

        return 'CoinGecko pricing request failed with status '.$status.': '.$snippet.$hint;
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
