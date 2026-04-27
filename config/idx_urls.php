<?php

/**
 * Shared Quantyra IDX host defaults (required from idx.php, ghl.php, bridge.php).
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

$defaultPreviewHosts = implode(',', array_values(array_unique(array_filter([
    $appHost,
    str_contains($appHost, '-api.') ? str_replace('-api.', '.', $appHost) : null,
    'localhost',
    '127.0.0.1',
]))));

return [
    'platform_url' => rtrim((string) env('IDX_PLATFORM_URL', $defaultPlatformUrl), '/'),
    'api_public_url' => rtrim((string) env('IDX_API_PUBLIC_URL', env('API_URL', $defaultApiUrl)), '/'),
    'images_public_url' => rtrim((string) env('IDX_IMAGES_PUBLIC_URL', env('IMAGE_URL', $defaultImagesUrl)), '/'),
    /**
     * Lowercase hostnames allowed for authenticated dashboard widget previews
     * (token still required; avoids whitelisting the IDX app in every GHL URL row).
     */
    'widget_preview_hostnames' => array_values(array_unique(array_filter(array_map(
        static function (string $host): string {
            return strtolower(rtrim(trim($host), '.'));
        },
        array_merge(
            explode(',', (string) env(
                'IDX_PLATFORM_HOSTS',
                $defaultPreviewHosts
            )),
            array_filter([(string) parse_url((string) env('IDX_PLATFORM_URL', $defaultPlatformUrl), PHP_URL_HOST)]),
            array_filter([(string) parse_url((string) env('IDX_API_PUBLIC_URL', env('API_URL', '')), PHP_URL_HOST)])
        )
    )))),
];
