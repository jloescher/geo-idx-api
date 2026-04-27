<?php

namespace App\Support;

use Illuminate\Http\Request;

/**
 * Resolves Quantyra IDX platform vs API public URLs for the active HTTP request.
 *
 * When the request hits an API-style host (…-idx-api… or idx-api…), platform URLs use the
 * same prefix swap as {@see routes/web.php} (strip {@code -api.} from the hostname) so dev/staging
 * installs link to dev-idx / staging-idx instead of production defaults.
 */
final class IdxPublicUrls
{
    /**
     * True when the host follows Quantyra’s idx-api / *-idx-api hostname convention.
     */
    public static function requestHostLooksLikeIdxApi(Request $request): bool
    {
        $host = strtolower(trim($request->getHttpHost()));

        return $host !== '' && str_contains($host, '-api.');
    }

    public static function apiBaseForRequest(Request $request): string
    {
        if (self::requestHostLooksLikeIdxApi($request)) {
            return rtrim($request->getScheme().'://'.$request->getHttpHost(), '/');
        }

        return rtrim((string) config('idx_urls.api_public_url'), '/');
    }

    public static function platformBaseForRequest(Request $request): string
    {
        if (self::requestHostLooksLikeIdxApi($request)) {
            $platformHost = str_replace('-api.', '.', strtolower($request->getHttpHost()));

            return rtrim($request->getScheme().'://'.$platformHost, '/');
        }

        return rtrim((string) config('idx_urls.platform_url'), '/');
    }
}
