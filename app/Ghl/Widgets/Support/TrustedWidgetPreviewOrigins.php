<?php

namespace App\Ghl\Widgets\Support;

/**
 * Hostnames where the Quantyra subscriber dashboard may load widget JS for proxy-backed previews.
 *
 * Matches {@see config('idx_urls.widget_preview_hostnames')} used by {@see WidgetEmbedValidationService}.
 */
final class TrustedWidgetPreviewOrigins
{
    public static function originIfTrusted(?string $originOrReferer): ?string
    {
        $normalized = OriginMatcher::normalizeOrigin($originOrReferer);
        if ($normalized === null) {
            return null;
        }

        $host = strtolower((string) parse_url($normalized, PHP_URL_HOST));
        if ($host === '') {
            return null;
        }

        /** @var list<string> $trusted */
        $trusted = config('idx_urls.widget_preview_hostnames', []);

        return in_array($host, $trusted, true) ? $normalized : null;
    }
}
