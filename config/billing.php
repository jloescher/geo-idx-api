<?php

/**
 * IDX subscription catalog — maps marketing tiers to Stripe Price IDs.
 *
 * Revenue Impact: Centralized price keys prevent checkout drift and keep “Add to cart” aligned with Stripe.
 */
return [

    'plans' => [
        'pro' => [
            'key' => 'pro',
            'label' => 'Pro',
            'monthly_display' => '$39',
            'annual_display' => '$374',
            'annual_note' => '20% off vs monthly × 12',
            'best_for' => 'Solo agents',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_PRO_MONTHLY'),
            'stripe_price_yearly' => env('STRIPE_PRICE_IDX_PRO_YEARLY'),
            'features' => [
                '1 IDX-authorized domain',
                '10,000 API calls / month included',
                '3 JS embed widgets (search, map, cards)',
                'Basic LeadConnector / GoHighLevel app',
                'Teaser gating (3 listings preview)',
                'Email OTP via TextGrid',
                'PostgreSQL-backed 15-minute MLS cache',
            ],
        ],
        'smart' => [
            'key' => 'smart',
            'label' => 'Smart',
            'monthly_display' => '$79',
            'annual_display' => '$758',
            'annual_note' => '20% off vs monthly × 12',
            'best_for' => 'Teams & power users',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_SMART_MONTHLY'),
            'stripe_price_yearly' => env('STRIPE_PRICE_IDX_SMART_YEARLY'),
            'features' => [
                '5 IDX-authorized domains',
                '50,000 API calls / month included',
                'Unlimited JS widgets',
                'Full LeadConnector / GoHighLevel app',
                'Phone + email OTP (TextGrid)',
                'Advanced search API for programmatic SEO',
                'Image proxy on NVMe-backed cache',
            ],
        ],
        'ultra' => [
            'key' => 'ultra',
            'label' => 'Ultra',
            'monthly_display' => '$179',
            'annual_display' => '$1,718',
            'annual_note' => '20% off vs monthly × 12',
            'best_for' => 'Agencies & custom builders',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_ULTRA_MONTHLY'),
            'stripe_price_yearly' => env('STRIPE_PRICE_IDX_ULTRA_YEARLY'),
            'features' => [
                'Unlimited domains',
                '250,000 API calls / month included',
                'Developer API keys & white-label widgets',
                'AI SEO helpers for neighborhood pages',
                'Priority support queue',
                'Leaflet + OpenStreetMap + REST depth',
            ],
        ],
        'mega' => [
            'key' => 'mega',
            'label' => 'Mega',
            'monthly_display' => '$449',
            'annual_display' => '$4,310',
            'annual_note' => '20% off vs monthly × 12',
            'best_for' => 'Brokerages & high-volume',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_MEGA_MONTHLY'),
            'stripe_price_yearly' => env('STRIPE_PRICE_IDX_MEGA_YEARLY'),
            'features' => [
                'Everything in Ultra',
                'Dedicated cache lane',
                'Unlimited API calls (fair use)',
                'Custom branding & revenue-share lead program',
                'SLA-backed uptime targets',
            ],
        ],
    ],

    'overage' => [
        'label' => '$0.001 per additional API call',
        'stripe_metered_price' => env('STRIPE_PRICE_IDX_API_OVERAGE_METERED'),
    ],

    'addons' => [
        'extra_domain' => [
            'label' => 'Extra domain',
            'monthly_display' => '$19/mo',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_ADDON_EXTRA_DOMAIN'),
        ],
        'priority_support' => [
            'label' => 'Priority support',
            'monthly_display' => '$49/mo',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_ADDON_PRIORITY_SUPPORT'),
        ],
    ],

    'trial_days' => 14,

    'teaser_leads_cache_ttl' => 900,
];
