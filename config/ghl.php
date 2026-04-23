<?php

// Revenue Impact: GHL Marketplace install → OAuth → widgets → leads → CRM = subscription velocity

$idx = require __DIR__.'/idx_urls.php';

$defaultScopes = implode(' ', [
    'contacts.readonly', 'contacts.write',
    'opportunities.readonly', 'opportunities.write',
    'locations.readonly', 'locations/tags.write',
    'users.readonly',
    'oauth.readonly', 'oauth.write',
    'conversations.readonly', 'conversations/message.write',
    'workflows.readonly',
    'calendars/events.write',
    'businesses.readonly',
    'forms.readonly',
    'links.write',
    'campaigns.readonly',
]);

return [
    'urls' => [
        'idx_platform' => $idx['platform_url'],
        'api_public' => $idx['api_public_url'],
        'images' => $idx['images_public_url'],
    ],

    'oauth' => [
        'client_id' => env('GHL_CLIENT_ID'),
        'client_secret' => env('GHL_CLIENT_SECRET'),
        'redirect_uri' => env('GHL_REDIRECT_URI', $idx['api_public_url'].'/oauth/leadconnector/callback'),
        'authorize_url' => env('GHL_AUTHORIZE_URL', 'https://marketplace.gohighlevel.com/oauth/chooselocation'),
        'token_url' => env('GHL_TOKEN_URL', 'https://services.leadconnectorhq.com/oauth/token'),
        'location_token_url' => env('GHL_LOCATION_TOKEN_URL', 'https://services.leadconnectorhq.com/oauth/locationToken'),
        'scopes' => env('GHL_SCOPES', $defaultScopes),
        'default_user_type' => env('GHL_DEFAULT_USER_TYPE', 'Location'),
    ],

    'api' => [
        'base_url' => env('GHL_API_BASE_URL', 'https://services.leadconnectorhq.com'),
        'version' => env('GHL_API_VERSION', '2021-07-28'),
        'timeout' => (int) env('GHL_API_TIMEOUT', 30),
        'max_retries' => (int) env('GHL_API_MAX_RETRIES', 3),
    ],

    'webhooks' => [
        'secret' => env('GHL_WEBHOOK_SECRET', env('GHL_CLIENT_SECRET')),
        'signature_header' => env('GHL_WEBHOOK_SIGNATURE_HEADER', 'X-GHL-Signature'),
        'require_signature' => env('GHL_WEBHOOK_REQUIRE_SIGNATURE', true),
    ],

    'sync' => [
        'queues' => [
            'sync' => env('GHL_QUEUE_SYNC', 'default'),
            'webhooks' => env('GHL_QUEUE_WEBHOOKS', 'default'),
            'maintenance' => env('GHL_QUEUE_MAINTENANCE', 'default'),
        ],
        'subscription_tags' => [
            'active' => env('GHL_SUBSCRIPTION_TAG_ACTIVE', 'quantyra-active'),
            'past_due' => env('GHL_SUBSCRIPTION_TAG_PAST_DUE', 'quantyra-past-due'),
            'cancelled' => env('GHL_SUBSCRIPTION_TAG_CANCELLED', 'quantyra-cancelled'),
            'trial' => env('GHL_SUBSCRIPTION_TAG_TRIAL', 'quantyra-trial'),
        ],
    ],

    'widgets' => [
        'rate_limit_per_minute' => (int) env('GHL_WIDGET_RATE_LIMIT', 120),
    ],

    'audit' => [
        'enabled' => env('GHL_AUDIT_LOG_ENABLED', true),
        'file' => env('GHL_AUDIT_LOG_PATH', storage_path('logs/ghl_audit.log')),
    ],

    'admin' => [
        'refresh_token' => env('GHL_ADMIN_REFRESH_TOKEN'),
    ],
];
