<?php

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
    | Listings row cache (PostgreSQL)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: 15-minute bulk refresh caps Bridge spend while keeping SERP-first paint fast.
    |
    | Compliance: cache stores only Active/Pending snapshots; closed/history is live API only (MLS GRID IDX posture).
    |
    */
    'listings_row_retention_days' => max(1, min(366, (int) env('MLS_LISTINGS_CACHE_RETENTION_DAYS', 365))),

    /*
    |--------------------------------------------------------------------------
    | Listings sync pagination (Active + Pending full pull)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: bounded page counts prevent runaway MLS egress on misconfigured upstreams.
    |
    */
    'listings_sync_page_size' => max(50, min(200, (int) env('MLS_LISTINGS_SYNC_PAGE_SIZE', 200))),
    'listings_sync_max_pages' => max(1, min(5000, (int) env('MLS_LISTINGS_SYNC_MAX_PAGES', 500))),
    'listings_sync_max_rows' => max(1000, min(500000, (int) env('MLS_LISTINGS_SYNC_MAX_ROWS', 100000))),
];
