<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Nominatim User-Agent
    |--------------------------------------------------------------------------
    |
    | OpenStreetMap Nominatim requires a valid User-Agent identifying your
    | application. Browsers cannot set this header on outbound fetch(), so
    | geocoding is proxied through /agent/searches/geocode with this value.
    |
    | @see https://operations.osmfoundation.org/policies/nominatim/
    |
    */
    'nominatim_user_agent' => env(
        'GEOCODING_NOMINATIM_USER_AGENT',
        'QuantyraIDX/1.0 (+'.(parse_url((string) env('APP_URL', 'http://localhost'), PHP_URL_HOST) ?: 'localhost').')'
    ),

    'nominatim_url' => env('GEOCODING_NOMINATIM_URL', 'https://nominatim.openstreetmap.org/search'),

    'timeout_seconds' => (int) env('GEOCODING_TIMEOUT', 10),

    /*
    |--------------------------------------------------------------------------
    | Geocode response cache
    |--------------------------------------------------------------------------
    |
    | Short TTL reduces duplicate Nominatim calls (policy-friendly). Cache is
    | keyed by normalized query + limit; misses still apply per-user rate limits.
    |
    */
    'geocode_cache_ttl_seconds' => (int) env('GEOCODING_CACHE_TTL', 120),

];
