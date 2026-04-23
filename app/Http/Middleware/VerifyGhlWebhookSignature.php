<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Revenue Impact: Verified webhooks → trusted install/uninstall → uninterrupted lead flow.
 */
class VerifyGhlWebhookSignature
{
    public function handle(Request $request, Closure $next): Response
    {
        if (! config('ghl.webhooks.require_signature')) {
            return $next($request);
        }

        $secret = (string) config('ghl.webhooks.secret');
        if ($secret === '') {
            abort(503, 'GHL webhook secret not configured');
        }

        $payload = $request->getContent();
        $header = (string) config('ghl.webhooks.signature_header');
        $signature = (string) $request->header($header, '');

        if ($signature === '') {
            abort(401, 'Missing webhook signature');
        }

        $expected = hash_hmac('sha256', $payload, $secret);

        if (! hash_equals($expected, $signature)) {
            abort(401, 'Invalid webhook signature');
        }

        return $next($request);
    }
}
