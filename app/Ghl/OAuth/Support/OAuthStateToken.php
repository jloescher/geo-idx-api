<?php

namespace App\Ghl\OAuth\Support;

use Illuminate\Contracts\Encryption\DecryptException;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Crypt;

/**
 * Encrypted OAuth `state` so the callback can validate without relying on session cookies
 * surviving cross-site redirects (Octane, proxies, cookie edge cases).
 */
final class OAuthStateToken
{
    private const MAX_AGE_SECONDS = 900;

    public static function encode(string $userType): string
    {
        $payload = json_encode([
            'ut' => self::normalizeUserType($userType),
            'iat' => time(),
        ], JSON_THROW_ON_ERROR);

        return Crypt::encryptString($payload);
    }

    /**
     * @return array{user_type: string}
     */
    public static function decode(string $state): array
    {
        if ($state === '') {
            throw new \InvalidArgumentException('Empty OAuth state');
        }

        $decoded = json_decode(Crypt::decryptString($state), true, 512, JSON_THROW_ON_ERROR);
        if (! is_array($decoded) || ! isset($decoded['ut'], $decoded['iat'])) {
            throw new \InvalidArgumentException('Malformed OAuth state');
        }

        $iat = (int) $decoded['iat'];
        if ($iat - time() > 120) {
            throw new \InvalidArgumentException('OAuth state not yet valid');
        }
        if (time() - $iat > self::MAX_AGE_SECONDS) {
            throw new \InvalidArgumentException('OAuth state expired');
        }

        return ['user_type' => self::normalizeUserType((string) $decoded['ut'])];
    }

    /**
     * @return array{user_type: string}|null
     */
    public static function tryResolveFromRequest(Request $request): ?array
    {
        $state = (string) $request->query('state', '');
        $code = (string) $request->query('code', '');

        if ($state === '' || $code === '') {
            return null;
        }

        try {
            return self::decode($state);
        } catch (\JsonException|DecryptException|\InvalidArgumentException) {
            $sessionState = (string) session('ghl_oauth_state', '');
            if ($sessionState !== '' && hash_equals($sessionState, $state)) {
                return [
                    'user_type' => self::normalizeUserType(
                        (string) session('ghl_oauth_user_type', config('ghl.oauth.default_user_type')),
                    ),
                ];
            }

            return null;
        }
    }

    private static function normalizeUserType(string $raw): string
    {
        $t = trim($raw);

        return strcasecmp($t, 'Company') === 0 ? 'Company' : 'Location';
    }
}
