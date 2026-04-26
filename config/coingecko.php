<?php

return [
    /*
    |--------------------------------------------------------------------------
    | CoinGecko API
    |--------------------------------------------------------------------------
    |
    | Revenue impact: centralized quote refresh allows listing pages to surface
    | multi-currency purchasing power without adding per-request third-party
    | latency or burning through free-tier API credits.
    |
    */
    'base_url' => rtrim((string) env('COINGECKO_BASE_URL', 'https://api.coingecko.com/api/v3'), '/'),

    'api_key' => env('COINGECKO_API_KEY'),

    'asset_ids' => array_values(array_filter(array_map(
        trim(...),
        explode(',', (string) env('COINGECKO_ASSET_IDS', 'btc,eth,sol,xrp,ada'))
    ))),

    'vs_currencies' => array_values(array_filter(array_map(
        trim(...),
        explode(',', (string) env('COINGECKO_VS_CURRENCIES', 'usd,cad,eur,gbp,mxn'))
    ))),

    'cache_key' => (string) env('COINGECKO_CACHE_KEY', 'coingecko.pricing.matrix'),

    'cache_ttl_seconds' => (int) env('COINGECKO_CACHE_TTL_SECONDS', 1200),

    'queue' => (string) env('COINGECKO_QUEUE', 'default'),

    'http_timeout_seconds' => (int) env('COINGECKO_HTTP_TIMEOUT', 12),

    'http_connect_timeout_seconds' => (int) env('COINGECKO_HTTP_CONNECT_TIMEOUT', 3),
];
