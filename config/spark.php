<?php

$idx = require __DIR__.'/idx_urls.php';

$defaultResoBase = rtrim((string) env('SPARK_RESO_BASE_URL', ''), '/');
if ($defaultResoBase === '') {
    $host = rtrim((string) env('SPARK_HOST', 'https://replication.sparkapi.com'), '/');
    $root = trim((string) env('SPARK_RESO_ROOT', 'Reso/OData'), '/');
    $defaultResoBase = $root !== '' ? "{$host}/{$root}" : $host;
}

return [

    /*
    |--------------------------------------------------------------------------
    | Spark Platform (Beaches MLS) RESO OData
    |--------------------------------------------------------------------------
    |
    | Compliance: Bearer token remains server-side; replication host only for keys
    | with replication permission. See docs/spark-api-documentation.md.
    |
    */

    'reso_base_url' => $defaultResoBase,

    'access_token' => env('SPARK_ACCESS_TOKEN', env('SPARK_API_KEY')),

    'api_feed_id' => env('SPARK_API_FEED_ID'),

    'timeout_seconds' => (int) env('SPARK_TIMEOUT', 30),

    'datasets' => array_values(array_filter(array_map(
        trim(...),
        explode(',', (string) env('SPARK_DATASETS', 'beaches'))
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

    'sync_incremental_poll_minutes' => max(1, (int) env('SPARK_SYNC_INCREMENTAL_POLL_MINUTES', 10)),

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

    'sync_persist_job_chunk_size' => min(250, max(25, (int) env('SPARK_SYNC_PERSIST_JOB_CHUNK', 50))),

    'replica_page_retention_hours' => max(1, (int) env('SPARK_REPLICA_PAGE_RETENTION_HOURS', 24)),

    'replica_page_failed_retention_days' => max(1, (int) env('SPARK_REPLICA_FAILED_RETENTION_DAYS', 7)),

    'local_mirror_rolling_months' => min(36, max(1, (int) env('SPARK_LOCAL_MIRROR_ROLLING_MONTHS', 12))),

];
