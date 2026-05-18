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
     * API suspension; pipeline fetch jobs chain until the cursor clears.
     */
    /** @deprecated Use sync_fetch_queue — kept for backward-compatible env overrides. */
    'sync_queue' => (string) env('BRIDGE_SYNC_QUEUE', env('BRIDGE_SYNC_FETCH_QUEUE', 'bridge-sync-fetch')),

    /** Rate-limited Bridge HTTP fetch jobs (kickoff + replication/incremental pages). */
    'sync_fetch_queue' => (string) env('BRIDGE_SYNC_FETCH_QUEUE', env('BRIDGE_SYNC_QUEUE', 'bridge-sync-fetch')),

    /** Parallel Postgres persist chunk jobs (no Bridge HTTP throttling). */
    'sync_persist_queue' => (string) env('BRIDGE_SYNC_PERSIST_QUEUE', 'bridge-sync-persist'),

    'sync_replication_top' => min(2000, max(1, (int) env('BRIDGE_SYNC_REPLICATION_TOP', 2000))),
    'sync_incremental_top' => min(200, max(1, (int) env('BRIDGE_SYNC_INCREMENTAL_TOP', 200))),

    /** Optional safety cap on chained fetch jobs per kickoff (0 = unlimited). */
    'sync_max_chained_fetch_pages' => max(0, (int) env('BRIDGE_SYNC_MAX_CHAINED_FETCH_PAGES', 0)),

    /** @deprecated Monolithic job page caps; kept for rollback. Pipeline ignores these. */
    'sync_max_replication_pages_per_job' => max(1, (int) env('BRIDGE_SYNC_MAX_REPLICATION_PAGES', 12)),
    'sync_max_incremental_pages_per_job' => max(1, (int) env('BRIDGE_SYNC_MAX_INCREMENTAL_PAGES', 40)),

    /** Max Bridge GETs per second during replication/incremental fetch (persist jobs excluded). */
    'sync_max_requests_per_second' => min(10, max(1, (int) env('BRIDGE_SYNC_MAX_REQUESTS_PER_SECOND', 2))),

    'sync_max_requests_per_minute' => min(334, max(1, (int) env('BRIDGE_SYNC_MAX_REQUESTS_PER_MINUTE', 120))),

    'sync_max_requests_per_hour' => min(5000, max(1, (int) env('BRIDGE_SYNC_MAX_REQUESTS_PER_HOUR', 4800))),

    'sync_min_fetch_interval_ms' => max(0, (int) env(
        'BRIDGE_SYNC_MIN_FETCH_INTERVAL_MS',
        (int) floor(1000 / max(1, (int) env('BRIDGE_SYNC_MAX_REQUESTS_PER_SECOND', 2)))
    )),

    'sync_include_media' => filter_var(env('BRIDGE_SYNC_INCLUDE_MEDIA', false), FILTER_VALIDATE_BOOL),
    /*
     * Max retries after HTTP 429 / 503 from Bridge (shared: listing sync, proxy, hybrid search).
     * Applies only to outbound Bridge requests — not application rate limits for domain/API-key traffic.
     */
    'sync_max_http_retries' => max(0, (int) env('BRIDGE_SYNC_MAX_HTTP_RETRIES', 4)),

    /*
     * Rolling mirror window — rows older than this (by MLS ModificationTimestamp) are purged
     * nightly; PostGIS searches also constrain to this window for parity with mirror scope.
     */
    'local_mirror_rolling_months' => min(36, max(1, (int) env('BRIDGE_LOCAL_MIRROR_ROLLING_MONTHS', 12))),

    'sync_upsert_chunk_size' => min(500, max(25, (int) env('BRIDGE_SYNC_UPSERT_CHUNK', 250))),

    /** Rows per queue persist job (limits jobs.payload size and worker peak RAM). */
    'sync_persist_job_chunk_size' => min(250, max(25, (int) env('BRIDGE_SYNC_PERSIST_JOB_CHUNK', 100))),

    /** Purge completed staging pages older than this many hours (failed rows use failed retention days). */
    'replica_page_retention_hours' => max(1, (int) env('BRIDGE_REPLICA_PAGE_RETENTION_HOURS', 24)),

    'replica_page_failed_retention_days' => max(1, (int) env('BRIDGE_REPLICA_FAILED_RETENTION_DAYS', 7)),
];
