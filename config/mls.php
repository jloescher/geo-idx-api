<?php

use App\Enums\MlsProvider;

$sparkFeeds = [];

if (is_string(env('MLS_SPARK_SPACE_COAST_CLIENT_ID')) && trim((string) env('MLS_SPARK_SPACE_COAST_CLIENT_ID')) !== '') {
    $sparkFeeds['space_coast'] = [
        'provider' => MlsProvider::Spark->value,
        'client_id' => env('MLS_SPARK_SPACE_COAST_CLIENT_ID'),
        'client_secret' => env('MLS_SPARK_SPACE_COAST_CLIENT_SECRET'),
        'token_url' => env('MLS_SPARK_SPACE_COAST_TOKEN_URL'),
        'reso_base_url' => rtrim((string) env('MLS_SPARK_SPACE_COAST_RESO_BASE_URL', ''), '/'),
        'oauth_scope' => env('MLS_SPARK_SPACE_COAST_SCOPE', 'api'),
    ];
}

if (is_string(env('MLS_SPARK_BEACHES_CLIENT_ID')) && trim((string) env('MLS_SPARK_BEACHES_CLIENT_ID')) !== '') {
    $sparkFeeds['beaches'] = [
        'provider' => MlsProvider::Spark->value,
        'client_id' => env('MLS_SPARK_BEACHES_CLIENT_ID'),
        'client_secret' => env('MLS_SPARK_BEACHES_CLIENT_SECRET'),
        'token_url' => env('MLS_SPARK_BEACHES_TOKEN_URL'),
        'reso_base_url' => rtrim((string) env('MLS_SPARK_BEACHES_RESO_BASE_URL', ''), '/'),
        'oauth_scope' => env('MLS_SPARK_BEACHES_SCOPE', 'api'),
    ];
}

return [

    /*
    |--------------------------------------------------------------------------
    | Bridge shared credentials (Bridge Interactive)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: one server-side key pair amortizes Stellar MLS egress across all Bridge-backed feeds.
    |
    | Compliance: credentials must never be forwarded to browsers; RESO proxy only.
    |
    */
    'bridge' => [
        'api_key' => env('BRIDGE_API_KEY'),
        'api_secret' => env('BRIDGE_API_SECRET'),
    ],

    /*
    |--------------------------------------------------------------------------
    | Spark OAuth token cache (NVMe-backed in prod when CACHE_STORE=file on fast disk)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: token reuse avoids Spark rate limits and keeps p95 search latency stable.
    |
    | Compliance: OAuth tokens are subscriber secrets; cache store must not be world-readable.
    |
    */
    'spark_token_cache_store' => env('MLS_SPARK_TOKEN_CACHE_STORE', 'file'),
    'spark_token_cache_ttl_seconds' => max(60, (int) env('MLS_SPARK_TOKEN_CACHE_TTL', 3300)),

    /*
    |--------------------------------------------------------------------------
    | Spark feeds (flat codes; merged at runtime with Bridge datasets — see MlsFeedResolver)
    |--------------------------------------------------------------------------
    */
    'spark_feeds' => $sparkFeeds,

    /*
    |--------------------------------------------------------------------------
    | Listings row cache (PostgreSQL)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: 15-minute bulk refresh caps Bridge/Spark spend while keeping SERP-first paint fast.
    |
    | Compliance: cache stores only Active/Pending snapshots; closed/history is live API only (MLS GRID IDX posture).
    |
    */
    'listings_row_retention_days' => max(1, min(366, (int) env('MLS_LISTINGS_CACHE_RETENTION_DAYS', 365))),
];
