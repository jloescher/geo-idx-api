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

];
