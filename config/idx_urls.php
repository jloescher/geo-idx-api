<?php

/**
 * Shared Quantyra IDX host defaults (required from idx.php, ghl.php, bridge.php).
 *
 * Do not call config() from other config files to read these values — that breaks
 * `php artisan config:cache` when those files load before `idx`.
 *
 * | Surface    | Env / default host              | Role |
 * |------------|---------------------------------|------|
 * | IDX App    | IDX_PLATFORM_URL → idx…        | GHL marketplace UX + independent dashboard |
 * | IDX API    | IDX_API_PUBLIC_URL or APP_URL  | REST + JS embed widgets (OAuth, webhooks) |
 * | IDX images | IDX_IMAGES_PUBLIC_URL → idx-images… | MLS image rewrite proxy in front of Bridge |
 */
return [
    'platform_url' => rtrim((string) env('IDX_PLATFORM_URL', 'https://idx.quantyralabs.cc'), '/'),
    'api_public_url' => rtrim((string) env('IDX_API_PUBLIC_URL', env('APP_URL', 'https://idx-api.quantyralabs.cc')), '/'),
    'images_public_url' => rtrim((string) env('IDX_IMAGES_PUBLIC_URL', 'https://idx-images.quantyralabs.cc'), '/'),
];
