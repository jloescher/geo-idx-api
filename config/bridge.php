<?php

$idx = require __DIR__.'/idx_urls.php';

return [

    /*
    |--------------------------------------------------------------------------
    | Bridge Data Output
    |--------------------------------------------------------------------------
    |
    | Revenue impact: correct host + path keeps MLS data flowing while
    | avoiding broken IDX pages (faster lead capture, fewer bounce exits).
    |
    | Paths follow docs/bridge-api-documentation.md (dataset: stellar).
    | If Bridge changes routing, override via env without code deploys.
    |
    */

    /*
     * Default host matches docs/bridge-api-documentation.md (Base URL).
     * Many deployments require https://api.bridgedataoutput.com — set BRIDGE_HOST.
     */
    'host' => rtrim(env('BRIDGE_HOST', 'https://bridgedataoutput.com'), '/'),

    'dataset' => env('BRIDGE_DATASET', 'stellar'),

    'server_token' => env('BRIDGE_API_KEY'),

    /*
     * Optional prefix before /{dataset}/... when your Bridge account uses API v2 routing.
     * Example: /api/v2 → final URL {host}/api/v2/{dataset}/listings
     */
    'path_prefix' => trim(env('BRIDGE_PATH_PREFIX', ''), '/'),

    /*
     * Example: reso/odata → {host}/reso/odata/{dataset}/Property
     * Leave empty to use doc-style {host}/{dataset}/Property.
     */
    'reso_root' => trim((string) env('BRIDGE_RESO_ROOT', ''), '/'),

    'timeout_seconds' => (int) env('BRIDGE_TIMEOUT', 30),

    'listings_cache_ttl_seconds' => (int) env('LISTINGS_CACHE_TTL', 900),

    'lookups_cache_ttl_seconds' => (int) env('BRIDGE_LOOKUPS_CACHE_TTL', 2_592_000),

    'image_cache_ttl_seconds' => (int) env('IMAGE_CACHE_TTL', 86400),

    'image_cache_path' => env('IMAGE_CACHE_PATH', storage_path('app/image_cache')),

    /*
    | Photo path relative to host (prefix applied first). Override if Bridge changes.
    */
    'listing_photo_path_template' => env(
        'BRIDGE_LISTING_PHOTO_PATH',
        '/api/v2/{dataset}/listings/{listingKey}/photos/{photoId}'
    ),

    /*
     * Available MLS datasets (comma-separated). Each dataset maps to a Bridge Data Output
     * data source. The first value is the default.
     */
    'datasets' => array_values(array_filter(array_map(
        trim(...),
        explode(',', (string) env('BRIDGE_DATASETS', 'stellar'))
    ))),

    'images_public_base' => $idx['images_public_url'],

    /*
     * Optional extra hostnames (comma-separated) whose listing photo URLs should be rewritten.
     */
    'image_rewrite_hosts' => array_values(array_filter(array_map(
        trim(...),
        explode(',', (string) env('BRIDGE_IMAGE_REWRITE_HOSTS', ''))
    ))),

    /*
     * Revenue impact: honoring Bridge $top limits (200 standard / 2000 replication) prevents
     * API suspension; bounded pages per job avoids tight retry loops under rate limits.
     */
    'sync_replication_top' => min(2000, max(1, (int) env('BRIDGE_SYNC_REPLICATION_TOP', 2000))),
    'sync_incremental_top' => min(200, max(1, (int) env('BRIDGE_SYNC_INCREMENTAL_TOP', 200))),
    'sync_max_replication_pages_per_job' => max(1, (int) env('BRIDGE_SYNC_MAX_REPLICATION_PAGES', 12)),
    'sync_max_incremental_pages_per_job' => max(1, (int) env('BRIDGE_SYNC_MAX_INCREMENTAL_PAGES', 40)),
    'sync_max_http_retries' => max(0, (int) env('BRIDGE_SYNC_MAX_HTTP_RETRIES', 4)),

    /*
     * Rolling mirror window — rows older than this (by MLS ModificationTimestamp) are purged
     * nightly; PostGIS searches also constrain to this window for parity with mirror scope.
     */
    'local_mirror_rolling_months' => min(36, max(1, (int) env('BRIDGE_LOCAL_MIRROR_ROLLING_MONTHS', 12))),

    'sync_upsert_chunk_size' => min(500, max(25, (int) env('BRIDGE_SYNC_UPSERT_CHUNK', 250))),
];
