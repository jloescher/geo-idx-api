<?php

return [

    'site_key' => env('CLOUDFLARE_TURNSTILE_SITE_KEY'),

    'secret_key' => env('CLOUDFLARE_TURNSTILE_SECRET_KEY'),

    'verify_url' => 'https://challenges.cloudflare.com/turnstile/v0/siteverify',

];
