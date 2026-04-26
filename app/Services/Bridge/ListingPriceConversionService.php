<?php

namespace App\Services\Bridge;

use App\Services\CoinGeckoPricingService;

class ListingPriceConversionService
{
    public function __construct(
        private readonly CoinGeckoPricingService $pricing,
    ) {}

    public function enrichJson(string $json): string
    {
        try {
            $payload = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (\JsonException) {
            return $json;
        }

        if (! is_array($payload)) {
            return $json;
        }

        $matrix = $this->pricing->latest();
        if (! is_array($matrix) || ! is_array($matrix['quotes'] ?? null)) {
            $payload['pricing'] = [
                'status' => 'unavailable',
                'quotes' => [],
                'as_of' => null,
            ];

            return (string) json_encode($payload, JSON_UNESCAPED_SLASHES | JSON_THROW_ON_ERROR);
        }

        $quotes = $matrix['quotes'];
        $payload['pricing'] = [
            'status' => 'ok',
            'quotes' => $quotes,
            'as_of' => is_string($matrix['as_of'] ?? null) ? $matrix['as_of'] : null,
        ];

        foreach (['value', 'bundle', 'd', 'listings'] as $key) {
            if (! isset($payload[$key]) || ! is_array($payload[$key])) {
                continue;
            }

            foreach ($payload[$key] as $index => $listing) {
                if (! is_array($listing)) {
                    continue;
                }

                $listPrice = $this->extractListPriceUsd($listing);
                if ($listPrice === null) {
                    continue;
                }

                $payload[$key][$index]['pricing_converted'] = [
                    'base_currency' => 'usd',
                    'fiat' => $this->convertFiatFromUsd($listPrice, $quotes),
                    'digital_assets' => $this->convertDigitalAssetsFromUsd($listPrice, $quotes),
                ];
            }
        }

        return (string) json_encode($payload, JSON_UNESCAPED_SLASHES | JSON_THROW_ON_ERROR);
    }

    /**
     * @param  array<string, mixed>  $listing
     */
    private function extractListPriceUsd(array $listing): ?float
    {
        $candidates = [
            $listing['ListPrice'] ?? null,
            $listing['listPrice'] ?? null,
        ];

        foreach ($candidates as $candidate) {
            if (is_numeric($candidate)) {
                return (float) $candidate;
            }
        }

        return null;
    }

    /**
     * @param  array<string, array<string, float>>  $quotes
     * @return array<string, float>
     */
    private function convertFiatFromUsd(float $listPriceUsd, array $quotes): array
    {
        $btcQuotes = $quotes['btc'] ?? [];
        if (! is_array($btcQuotes) || ! isset($btcQuotes['usd']) || ! is_numeric($btcQuotes['usd']) || (float) $btcQuotes['usd'] <= 0) {
            return ['usd' => round($listPriceUsd, 2)];
        }

        $fiatValues = [];
        foreach ($btcQuotes as $currency => $btcCurrencyPrice) {
            if (! is_string($currency) || ! is_numeric($btcCurrencyPrice)) {
                continue;
            }

            $fiatValues[$currency] = round($listPriceUsd * ((float) $btcCurrencyPrice / (float) $btcQuotes['usd']), 2);
        }

        if (! isset($fiatValues['usd'])) {
            $fiatValues['usd'] = round($listPriceUsd, 2);
        }

        return $fiatValues;
    }

    /**
     * @param  array<string, array<string, float>>  $quotes
     * @return array<string, float>
     */
    private function convertDigitalAssetsFromUsd(float $listPriceUsd, array $quotes): array
    {
        $converted = [];
        foreach ($quotes as $assetId => $currencyQuotes) {
            if (! is_array($currencyQuotes) || ! isset($currencyQuotes['usd']) || ! is_numeric($currencyQuotes['usd'])) {
                continue;
            }

            $usdPrice = (float) $currencyQuotes['usd'];
            if ($usdPrice <= 0) {
                continue;
            }

            $converted[$assetId] = round($listPriceUsd / $usdPrice, 8);
        }

        return $converted;
    }
}
