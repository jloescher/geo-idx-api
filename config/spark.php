<?php

$idx = require __DIR__.'/idx_urls.php';

$replicationHost = rtrim((string) env('SPARK_REPLICATION_HOST', 'https://replication.sparkapi.com'), '/');
$replicationRoot = trim((string) env('SPARK_REPLICATION_RESO_ROOT', 'Reso/OData'), '/');
$legacyResoOverride = rtrim((string) env('SPARK_RESO_BASE_URL', ''), '/');

if ($legacyResoOverride !== '') {
    $replicationResoBase = $legacyResoOverride;
} else {
    $replicationResoBase = $replicationRoot !== '' ? "{$replicationHost}/{$replicationRoot}" : $replicationHost;
}

$apiHost = rtrim((string) env('SPARK_API_HOST', 'https://sparkapi.com'), '/');
$apiVersion = trim((string) env('SPARK_API_VERSION', 'v1'), '/');
$liveRoot = trim((string) env('SPARK_LIVE_RESO_ROOT', 'Reso/OData'), '/');
$liveResoBase = $liveRoot !== ''
    ? "{$apiHost}/{$apiVersion}/{$liveRoot}"
    : "{$apiHost}/{$apiVersion}";

return [

    /*
    |--------------------------------------------------------------------------
    | Spark Platform (Beaches MLS) RESO OData
    |--------------------------------------------------------------------------
    |
    | Replication keys must use replication.sparkapi.com (sync jobs only).
    | Live IDX proxy uses sparkapi.com. See docs/spark/README.md.
    |
    */

    /** @deprecated Use replication_reso_base_url or live_reso_base_url */
    'reso_base_url' => $replicationResoBase,

    'replication_host' => $replicationHost,

    'replication_reso_root' => $replicationRoot,

    'replication_reso_base_url' => $replicationResoBase,

    'api_host' => $apiHost,

    'api_version' => $apiVersion,

    'live_reso_root' => $liveRoot,

    'live_reso_base_url' => $liveResoBase,

    'access_token' => env('SPARK_ACCESS_TOKEN', env('SPARK_API_KEY')),

    'api_feed_id' => env('SPARK_API_FEED_ID'),

    'timeout_seconds' => (int) env('SPARK_TIMEOUT', 30),

    'datasets' => array_values(array_keys(array_filter(
        config('mls.datasets', []),
        static fn (mixed $def): bool => is_array($def)
            && ($def['provider'] ?? '') === 'spark'
            && ($def['enabled'] ?? true) !== false,
    ))),

    'images_public_base' => $idx['images_public_url'],

    'image_rewrite_hosts' => array_values(array_filter(array_map(
        trim(...),
        explode(',', (string) env('SPARK_IMAGE_REWRITE_HOSTS', 'cdn.photos.sparkplatform.com'))
    ))),

    'sync_fetch_queue' => (string) env('SPARK_SYNC_FETCH_QUEUE', 'spark-sync-fetch'),

    'sync_persist_queue' => (string) env('SPARK_SYNC_PERSIST_QUEUE', 'spark-sync-persist'),

    'sync_replication_top' => min(1000, max(1, (int) env('SPARK_SYNC_REPLICATION_TOP', 1000))),

    'sync_incremental_top' => min(1000, max(1, (int) env('SPARK_SYNC_INCREMENTAL_TOP', 1000))),

    'sync_incremental_poll_minutes' => max(1, (int) env('SPARK_SYNC_INCREMENTAL_POLL_MINUTES', env('MLS_STEADY_INCREMENTAL_POLL_MINUTES', 15))),

    /** OData $expand for replication (Media, Unit, Room, OpenHouse). */
    'sync_expand' => (string) env('SPARK_SYNC_EXPAND', 'Media,Unit,Room,OpenHouse'),

    'sync_max_chained_fetch_pages' => max(0, (int) env('SPARK_SYNC_MAX_CHAINED_FETCH_PAGES', 0)),

    'sync_max_requests_per_second' => min(10, max(1, (int) env('SPARK_SYNC_MAX_REQUESTS_PER_SECOND', 2))),

    'sync_min_fetch_interval_ms' => max(0, (int) env(
        'SPARK_SYNC_MIN_FETCH_INTERVAL_MS',
        (int) floor(1000 / max(1, (int) env('SPARK_SYNC_MAX_REQUESTS_PER_SECOND', 2)))
    )),

    'sync_max_http_retries' => max(0, (int) env('SPARK_SYNC_MAX_HTTP_RETRIES', 4)),

    'sync_upsert_chunk_size' => min(500, max(25, (int) env('SPARK_SYNC_UPSERT_CHUNK', 250))),

    'sync_persist_job_chunk_size' => min(250, max(25, (int) env('SPARK_SYNC_PERSIST_JOB_CHUNK', 25))),

    'replica_page_retention_hours' => max(1, (int) env('SPARK_REPLICA_PAGE_RETENTION_HOURS', 24)),

    'replica_page_failed_retention_days' => max(1, (int) env('SPARK_REPLICA_FAILED_RETENTION_DAYS', 7)),

    'local_mirror_rolling_months' => min(36, max(1, (int) env('SPARK_LOCAL_MIRROR_ROLLING_MONTHS', 12))),

];
