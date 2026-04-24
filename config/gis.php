<?php

return [

    /*
    |--------------------------------------------------------------------------
    | GIS proxy TTL & transport
    |--------------------------------------------------------------------------
    |
    | Revenue impact: 15-minute parity with listings_cache keeps GIS overlay
    | traffic off fragile government hosts while map dwell time rises (more
    | registration completions per Bridge session).
    |
    */
    'cache_ttl_seconds' => (int) env('GIS_CACHE_TTL', 900),

    'http_timeout_seconds' => (int) env('GIS_HTTP_TIMEOUT', 18),

    'http_connect_timeout_seconds' => (int) env('GIS_HTTP_CONNECT_TIMEOUT', 5),

    /*
    |--------------------------------------------------------------------------
    | Queue (database driver compatible)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: offloading NVMe backup writes avoids blocking Octane
    | workers during burst map pans so OTP-gated full-detail views stay snappy.
    |
    */
    'queue' => env('GIS_QUEUE', 'default'),

    'queue_backup_writes' => filter_var(env('GIS_QUEUE_BACKUP_WRITES', true), FILTER_VALIDATE_BOOL),

    /*
    |--------------------------------------------------------------------------
    | Source priority (public government GIS only — zero MLS payload)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: ordered public sources let geo-web ship one Leaflet layer
    | for Florida pilots without MLS compliance risk, lifting time-on-site.
    |
    */
    'sources' => [
        'primary' => [
            'key' => 'florida_statewide_cadastral',
            'label' => 'Florida DOR / FGIO statewide cadastral',
            // Revenue impact: statewide coverage maximizes one integration for all FL IDX embeds.
            'query_url' => 'https://services9.arcgis.com/Gh9awoU677aKree0/arcgis/rest/services/Florida_Statewide_Cadastral/FeatureServer/0/query',
            'out_fields' => 'OBJECTID,PARCEL_ID,CO_NO,ASMNT_YR,BAS_STRT',
            'type' => 'feature_server',
        ],
        'pinellas' => [
            'key' => 'pinellas_enterprise_parcels',
            'label' => 'Pinellas County Enterprise GIS — PublicWebGIS/Parcels',
            // Revenue impact: Pinellas is the revenue pilot; local parcels load fast when statewide throttles.
            'query_url' => 'https://egis.pinellas.gov/gis/rest/services/PublicWebGIS/Parcels/MapServer/1/query',
            'out_fields' => '*',
            'type' => 'map_server',
            'bbox_wgs84' => [-82.92, 27.60, -82.52, 28.20],
        ],
        'hillsborough' => [
            'key' => 'hillsborough_hc_parcels',
            'label' => 'Hillsborough County HC_Parcels',
            // Revenue impact: Tampa Bay corridor coverage protects conversion when Pinellas-only layers miss bbox.
            'query_url' => 'https://maps.hillsboroughcounty.org/arcgis/rest/services/InfoLayers/HC_Parcels/FeatureServer/0/query',
            'out_fields' => 'OBJECTID,FOLIO,PIN,OWNER_NAME',
            'type' => 'feature_server',
            'bbox_wgs84' => [-82.85, 27.82, -82.05, 28.28],
        ],
    ],

    /*
    |--------------------------------------------------------------------------
    | Teaser vs full (OTP / idx:full parity with Bridge proxy)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: geometry/attribute starvation pre-OTP nudges phone verify
    | while still proving map value (lower bounce than hard wall, higher lead yield).
    |
    */
    'teaser_max_features' => (int) env('GIS_TEASER_MAX_FEATURES', 40),

    'teaser_coordinate_decimals' => (int) env('GIS_TEASER_COORD_DECIMALS', 4),

    'max_features_cap' => (int) env('GIS_MAX_FEATURES', 500),

    /*
    |--------------------------------------------------------------------------
    | Bbox guard (abuse / accidental world queries)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: bounding span caps prevent malicious mega-bbox pulls that
    | would starve Octane workers and erase margin on free teaser traffic.
    |
    */
    'max_bbox_span_degrees' => (float) env('GIS_MAX_BBOX_SPAN_DEG', 0.35),

    /*
    |--------------------------------------------------------------------------
    | Florida DOR CO_NO hints (speeds statewide layer when county is known)
    |--------------------------------------------------------------------------
    */
    'dor_county_numbers' => [
        'pinellas' => 52,
        'hillsborough' => 57,
    ],

    /*
    |--------------------------------------------------------------------------
    | MLS routing (future multi-board; same GIS chain for Florida public data)
    |--------------------------------------------------------------------------
    */
    'florida_mls_codes' => array_values(array_filter(array_map(
        trim(...),
        explode(',', (string) env('GIS_FLORIDA_MLS_CODES', 'stellar'))
    ))),

    /*
    |--------------------------------------------------------------------------
    | Graceful degrade — Leaflet-friendly OSM raster template (public tiles)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: a visible basemap hint keeps sessions alive when every
    | ArcGIS tier fails, preserving retargeting and return visits.
    |
    */
    'degraded_tile_layer' => [
        'url' => env('GIS_DEGRADED_TILE_URL', 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'),
        'attribution' => '&copy; OpenStreetMap contributors',
        'max_zoom' => 19,
    ],

    /*
    |--------------------------------------------------------------------------
    | Full-access context (URLs only — fetched client-side; no MLS data)
    |--------------------------------------------------------------------------
    |
    | Revenue impact: deep context layers post-OTP increase seller trust and
    | form submissions without expanding idx-api MLS liability surface.
    |
    */
    'full_access_context_layers' => [
        [
            'id' => 'pinellas_fema_folder',
            'title' => 'Pinellas County FEMA / flood story maps (ArcGIS folder)',
            'catalog_url' => 'https://egis.pinellas.gov/gis/rest/services?f=pjson',
        ],
    ],

];
