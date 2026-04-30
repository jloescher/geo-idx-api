<?php

return [

    'provider' => env('GEOCODING_PROVIDER', 'google'),

    'google_api_key' => env('GOOGLE_MAPS_GEOCODING_API_KEY'),

    'cache_ttl_seconds' => (int) env('GEOCODING_CACHE_TTL', 2592000),

    'timeout_seconds' => (int) env('GEOCODING_TIMEOUT', 5),

];
