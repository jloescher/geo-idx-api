<?php

return [

    /*
    |--------------------------------------------------------------------------
    | MLS datasets (single source of truth)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: add Miami or new feeds via config only — no code deploy for dataset keys.
    |
    */
    'datasets' => [
        'stellar' => [
            'provider' => 'bridge',
            'label' => 'Stellar MLS (Bridge)',
            'enabled' => filter_var(env('MLS_STELLAR_ENABLED', true), FILTER_VALIDATE_BOOL),
            'api_key' => env('MLS_STELLAR_API_KEY', env('BRIDGE_STELLAR_KEY', env('BRIDGE_API_KEY'))),
            'api_secret' => env('BRIDGE_API_SECRET'),
            'replication_top' => min(2000, max(1, (int) env('MLS_STELLAR_REPLICATION_TOP', env('BRIDGE_SYNC_REPLICATION_TOP', 2000)))),
            'incremental_top' => min(200, max(1, (int) env('MLS_STELLAR_INCREMENTAL_TOP', env('BRIDGE_SYNC_INCREMENTAL_TOP', 200)))),
            'persist_chunk_size' => min(250, max(25, (int) env('MLS_STELLAR_PERSIST_CHUNK_SIZE', env('BRIDGE_SYNC_PERSIST_JOB_CHUNK', 50)))),
            'upsert_chunk_size' => min(500, max(25, (int) env('MLS_STELLAR_UPSERT_CHUNK_SIZE', env('BRIDGE_SYNC_UPSERT_CHUNK', 250)))),
            'fetch_queue' => env('BRIDGE_SYNC_FETCH_QUEUE', 'bridge-sync-fetch'),
            'persist_queue' => env('BRIDGE_SYNC_PERSIST_QUEUE', 'bridge-sync-persist'),
            'rate_limit_rps' => min(10, max(1, (int) env('MLS_STELLAR_RATE_LIMIT_RPS', env('BRIDGE_SYNC_MAX_REQUESTS_PER_SECOND', 2)))),
            'persist_sequential' => false,
        ],
        'beaches' => [
            'provider' => 'spark',
            'label' => 'Beaches MLS (Spark)',
            'enabled' => filter_var(env('MLS_BEACHES_ENABLED', true), FILTER_VALIDATE_BOOL),
            'api_key' => env('MLS_BEACHES_API_KEY', env('SPARK_BEACHES_KEY', env('SPARK_ACCESS_TOKEN'))),
            'replication_top' => min(1000, max(1, (int) env('MLS_BEACHES_REPLICATION_TOP', env('SPARK_SYNC_REPLICATION_TOP', 1000)))),
            'incremental_top' => min(1000, max(1, (int) env('MLS_BEACHES_INCREMENTAL_TOP', env('SPARK_SYNC_INCREMENTAL_TOP', 1000)))),
            'expand' => (string) env('SPARK_SYNC_EXPAND', 'Media,Unit,Room,OpenHouse'),
            'persist_chunk_size' => min(250, max(25, (int) env('MLS_BEACHES_PERSIST_CHUNK_SIZE', env('SPARK_SYNC_PERSIST_JOB_CHUNK', 25)))),
            'upsert_chunk_size' => min(500, max(25, (int) env('MLS_BEACHES_UPSERT_CHUNK_SIZE', env('SPARK_SYNC_UPSERT_CHUNK', 250)))),
            'fetch_queue' => env('SPARK_SYNC_FETCH_QUEUE', 'spark-sync-fetch'),
            'persist_queue' => env('SPARK_SYNC_PERSIST_QUEUE', 'spark-sync-persist'),
            'rate_limit_rps' => min(10, max(1, (int) env('MLS_BEACHES_RATE_LIMIT_RPS', env('SPARK_SYNC_MAX_REQUESTS_PER_SECOND', 2)))),
            'persist_sequential' => filter_var(env('SPARK_SYNC_PERSIST_SEQUENTIAL', true), FILTER_VALIDATE_BOOL),
        ],
    ],

    /*
    |--------------------------------------------------------------------------
    | Bridge shared credentials (legacy keys — prefer per-dataset api_key above)
    |--------------------------------------------------------------------------
    */
    'bridge' => [
        'api_key' => env('BRIDGE_API_KEY'),
        'api_secret' => env('BRIDGE_API_SECRET'),
    ],

    'search_cache_table' => 'mls_search_cache',

    'listings_row_retention_days' => max(1, min(366, (int) env('MLS_LISTINGS_CACHE_RETENTION_DAYS', 365))),

    'listings_sync_page_size' => max(50, min(200, (int) env('MLS_LISTINGS_SYNC_PAGE_SIZE', 200))),
    'listings_sync_max_pages' => max(1, min(5000, (int) env('MLS_LISTINGS_SYNC_MAX_PAGES', 500))),
    'listings_sync_max_rows' => max(1000, min(500000, (int) env('MLS_LISTINGS_SYNC_MAX_ROWS', 100000))),

    /*
    |--------------------------------------------------------------------------
    | Mirror rolling window (Active/Pending retention + PostGIS search)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: staging can use a shorter window (e.g. 3 months) while production
    | keeps 12 months without provider-specific env duplication.
    |
    */
    'local_mirror_rolling_months' => min(48, max(1, (int) env(
        'MLS_LOCAL_MIRROR_ROLLING_MONTHS',
        env('BRIDGE_LOCAL_MIRROR_ROLLING_MONTHS', env('SPARK_LOCAL_MIRROR_ROLLING_MONTHS', 12))
    ))),

    /*
    |--------------------------------------------------------------------------
    | Replication freshness & scheduling
    |--------------------------------------------------------------------------
    |
    | Revenue impact: catch-up at ~2 req/s until mirror is within freshness window,
    | then 15-minute incremental polls cap MLS egress while keeping maps current.
    |
    */
    'freshness_threshold_minutes' => max(1, (int) env('MLS_REPLICATION_FRESHNESS_MINUTES', 15)),
    'catch_up_kickoff_interval_minutes' => max(1, (int) env('MLS_CATCH_UP_KICKOFF_MINUTES', 1)),
    'steady_incremental_poll_minutes' => max(1, (int) env('MLS_STEADY_INCREMENTAL_POLL_MINUTES', 15)),

    'replica_page_retention_hours' => max(1, (int) env(
        'MLS_REPLICA_PAGE_RETENTION_HOURS',
        env('BRIDGE_REPLICA_PAGE_RETENTION_HOURS', env('SPARK_REPLICA_PAGE_RETENTION_HOURS', 24))
    )),
    'replica_page_failed_retention_days' => max(1, (int) env(
        'MLS_REPLICA_PAGE_FAILED_RETENTION_DAYS',
        env('BRIDGE_REPLICA_FAILED_RETENTION_DAYS', env('SPARK_REPLICA_FAILED_RETENTION_DAYS', 7))
    )),

];
