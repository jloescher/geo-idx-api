<?php

/**
 * Shared Quantyra IDX host defaults (required from idx.php, bridge.php, and related config).
 *
 * Do not call config() from other config files to read these values — that breaks
 * `php artisan config:cache` when those files load before `idx`.
 *
 * APP_URL is the canonical base URL fallback for all absolute URL generation.
 */
$appUrl = rtrim((string) env('APP_URL', 'http://localhost'), '/');
$appHost = (string) parse_url($appUrl, PHP_URL_HOST);

$defaultPlatformUrl = str_contains($appHost, '-api.')
    ? str_replace('-api.', '.', $appUrl)
    : $appUrl;

$defaultApiUrl = rtrim((string) env('API_URL', $appUrl), '/');
$defaultImagesUrl = rtrim((string) env('IMAGE_URL', $appUrl), '/');

return [
    'platform_url' => rtrim((string) env('IDX_PLATFORM_URL', $defaultPlatformUrl), '/'),
    'api_public_url' => rtrim((string) env('IDX_API_PUBLIC_URL', env('API_URL', $defaultApiUrl)), '/'),
    'images_public_url' => rtrim((string) env('IDX_IMAGES_PUBLIC_URL', env('IMAGE_URL', $defaultImagesUrl)), '/'),
];
