<?php

namespace App\Ghl\Widgets\Support;

use App\Models\Domain;
use Illuminate\Http\Request;

/**
 * Resolves an approved {@see Domain} for widget traffic from the embed Origin / Referer host.
 */
final class WidgetEmbedDomainResolver
{
    public static function hostFromRequest(Request $request): ?string
    {
        $origin = (string) $request->headers->get('Origin', '');
        if ($origin === '') {
            $origin = (string) $request->headers->get('Referer', '');
        }
        if ($origin === '') {
            return null;
        }
        $host = parse_url($origin, PHP_URL_HOST);

        return is_string($host) && $host !== '' ? mb_strtolower($host) : null;
    }

    public static function resolveActiveDomain(Request $request): ?Domain
    {
        $host = self::hostFromRequest($request);
        if ($host === null) {
            return null;
        }

        $stripped = preg_replace('/^www\./', '', $host);
        $stripped = is_string($stripped) ? $stripped : $host;

        return Domain::query()
            ->active()
            ->where(function ($q) use ($host, $stripped): void {
                $q->whereRaw('LOWER(domain_slug) = ?', [$host])
                    ->orWhereRaw('LOWER(domain_slug) = ?', [$stripped]);
            })
            ->first();
    }
}
