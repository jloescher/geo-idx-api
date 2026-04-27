<?php

namespace App\Ghl\Widgets\Support;

class OriginMatcher
{
    public static function normalizeOrigin(?string $origin): ?string
    {
        if (! is_string($origin) || trim($origin) === '') {
            return null;
        }

        $parts = parse_url(trim($origin));
        if (! is_array($parts)) {
            return null;
        }

        $scheme = strtolower((string) ($parts['scheme'] ?? ''));
        $host = strtolower((string) ($parts['host'] ?? ''));
        if ($scheme === '' || $host === '') {
            return null;
        }

        $port = isset($parts['port']) ? (int) $parts['port'] : null;
        $isDefaultPort = ($scheme === 'https' && $port === 443) || ($scheme === 'http' && $port === 80);
        $portSuffix = $port !== null && ! $isDefaultPort ? ':'.$port : '';

        return $scheme.'://'.$host.$portSuffix;
    }

    public static function allowedOrigin(?string $origin, array $allowedOrigins): ?string
    {
        $normalizedOrigin = self::normalizeOrigin($origin);
        if ($normalizedOrigin === null) {
            return null;
        }

        foreach ($allowedOrigins as $allowedOrigin) {
            if (self::normalizeOrigin((string) $allowedOrigin) === $normalizedOrigin) {
                return $normalizedOrigin;
            }
        }

        return null;
    }
}
