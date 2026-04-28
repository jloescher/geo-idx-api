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
                'Unlimited JS Widget usage',
                '1 domain included',
                'Additional domains: $39/mo each',
                'No monthly API call limits advertised',
                'Basic LeadConnector / GoHighLevel app',
                'Teaser gating (3 listings preview)',
                'Email OTP delivery',
                '15-minute MLS data cache',
                'Full REST API access: No',
            ],
        ],
        'smart' => [
            'key' => 'smart',
            'label' => 'Smart',
            'monthly_display' => '$79',
            'annual_display' => '$758',
            'annual_note' => '20% off vs monthly × 12',
            'best_for' => 'Teams & growing agents',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_SMART_MONTHLY'),
            'stripe_price_yearly' => env('STRIPE_PRICE_IDX_SMART_YEARLY'),
            'features' => [
                'Unlimited JS Widget usage',
                '5 domains / unlimited widgets',
                'No monthly API call limits advertised',
                'Full LeadConnector / GoHighLevel app',
                'Phone + email OTP',
                'Advanced search and lead-capture experiences',
                'Image proxy with fast SSD-backed cache',
                'Full REST API access: No',
            ],
        ],
        'ultra' => [
            'key' => 'ultra',
            'label' => 'Ultra',
            'monthly_display' => '$179',
            'annual_display' => '$1,718',
            'annual_note' => '20% off vs monthly × 12',
            'best_for' => 'Agencies & custom site builders',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_ULTRA_MONTHLY'),
            'stripe_price_yearly' => env('STRIPE_PRICE_IDX_ULTRA_YEARLY'),
            'features' => [
                'Unlimited domains',
                '2,000,000 API calls / month',
                'Developer API keys + full REST API',
                'Overage: $0.001 per additional API call',
                'Priority support queue',
                'Fair-usage + soft rate limits (backend protection)',
            ],
        ],
        'mega' => [
            'key' => 'mega',
            'label' => 'Mega',
            'monthly_display' => '$449',
            'annual_display' => '$4,310',
            'annual_note' => '20% off vs monthly × 12',
            'best_for' => 'Brokerages & high-volume users',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_MEGA_MONTHLY'),
            'stripe_price_yearly' => env('STRIPE_PRICE_IDX_MEGA_YEARLY'),
            'features' => [
                'Unlimited API calls',
                'Unlimited domains',
                'Full REST API access',
                'Comps engine access',
                'Overage: $0.001 per additional API call',
                'Fair-usage + soft rate limits (backend protection)',
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
            'monthly_display' => '$39/mo',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_ADDON_EXTRA_DOMAIN'),
        ],
        'priority_support' => [
            'label' => 'Priority support',
            'monthly_display' => '$49/mo',
            'stripe_price_monthly' => env('STRIPE_PRICE_IDX_ADDON_PRIORITY_SUPPORT'),
        ],
    ],

    'trial_days' => 0,

    'teaser_leads_cache_ttl' => 900,
];
