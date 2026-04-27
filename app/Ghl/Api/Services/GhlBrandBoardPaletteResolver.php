<?php

namespace App\Ghl\Api\Services;

use App\Ghl\Api\Clients\GhlApiClient;
use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use Illuminate\Support\Facades\Cache;
use Throwable;

/**
 * Fetches HighLevel Brand Board design kit colors/fonts for a location and maps them to widget CSS tokens.
 *
 * List response (GET /brand-boards/:locationId) omits full colors; this service loads detail via
 * GET /brand-boards/:locationId/:id. Scope: brand-boards/design-kit.readonly.
 *
 * @see https://marketplace.gohighlevel.com/docs/ghl/brand-boards/get-brand-boards-by-location
 */
final class GhlBrandBoardPaletteResolver
{
    public function __construct(
        private readonly GhlApiClient $client,
    ) {}

    /**
     * Cached palette for widget merge, or null if unavailable / no colors.
     *
     * @return array<string, string>|null
     */
    public function resolveForRegisteredUrl(GhlRegisteredUrl $row): ?array
    {
        $token = $row->oauthToken;
        if ($token === null || ! $token->isActive()) {
            return null;
        }

        $locationId = $row->ghl_location_id;
        $ttl = max(60, (int) config('ghl.brand_boards.cache_ttl_seconds', 3600));
        $cacheKey = 'ghl:brand-board-palette:v1:'.$locationId;
        $failKey = $cacheKey.':empty';

        $stored = Cache::get($cacheKey);
        if (is_array($stored) && array_key_exists('palette', $stored)) {
            return $stored['palette'];
        }

        if (Cache::get($failKey)) {
            return null;
        }

        $palette = $this->fetchAndMapPalette($token, $locationId);
        if ($palette !== null) {
            Cache::put($cacheKey, ['palette' => $palette], $ttl);
        } else {
            Cache::put($failKey, true, min(300, $ttl));
        }

        return $palette;
    }

    /**
     * @return array<string, string>|null
     */
    private function fetchAndMapPalette(GhlOAuthToken $token, string $locationId): ?array
    {
        try {
            $list = $this->client->request($token, 'GET', 'brand-boards/'.$locationId);
        } catch (Throwable) {
            return null;
        }
        if (! $list->successful()) {
            return null;
        }

        $body = $list->json();
        $boards = $body['brandBoards'] ?? null;
        if (! is_array($boards) || $boards === []) {
            return null;
        }

        $selected = null;
        foreach ($boards as $board) {
            if (! is_array($board)) {
                continue;
            }
            if (($board['default'] ?? $board['isDefault'] ?? false) === true) {
                $selected = $board;
                break;
            }
        }
        $selected ??= is_array($boards[0] ?? null) ? $boards[0] : null;
        if ($selected === null) {
            return null;
        }

        $boardId = (string) ($selected['_id'] ?? $selected['id'] ?? $selected['brandBoardId'] ?? '');
        if ($boardId === '') {
            return null;
        }

        try {
            $detail = $this->client->request($token, 'GET', 'brand-boards/'.$locationId.'/'.$boardId);
        } catch (Throwable) {
            return null;
        }
        if (! $detail->successful()) {
            return null;
        }

        $detailBody = $detail->json();
        if (! is_array($detailBody)) {
            return null;
        }

        if (($detailBody['deleted'] ?? false) === true) {
            return null;
        }

        $colors = $detailBody['colors'] ?? null;
        if (! is_array($colors) || $colors === []) {
            return null;
        }

        return $this->mapBrandColorsToPalette($colors, $detailBody['fonts'] ?? null);
    }

    /**
     * @param  list<mixed>  $colors
     * @return array<string, string>|null
     */
    private function mapBrandColorsToPalette(array $colors, mixed $fonts): ?array
    {
        $byLabel = [];
        $ordered = [];
        foreach ($colors as $entry) {
            if (! is_array($entry)) {
                continue;
            }
            $hex = $entry['hex'] ?? $entry['Hex'] ?? null;
            if (! is_string($hex) || trim($hex) === '') {
                continue;
            }
            $normalizedHex = $this->normalizeHex6((string) $hex);
            if ($normalizedHex === null) {
                continue;
            }
            $ordered[] = $normalizedHex;
            $label = isset($entry['label']) && is_string($entry['label'])
                ? mb_strtolower(trim($entry['label']))
                : '';
            if ($label !== '') {
                $byLabel[$label] = $normalizedHex;
            }
        }

        if ($ordered === []) {
            return null;
        }

        $find = static function (array $patterns) use ($byLabel): ?string {
            foreach ($byLabel as $label => $hex) {
                foreach ($patterns as $p) {
                    if ($label !== '' && str_contains($label, $p)) {
                        return $hex;
                    }
                }
            }

            return null;
        };

        $primary = $find(['primary', 'brand', 'main'])
            ?? ($ordered[0] ?? null);
        $secondary = $find(['secondary'])
            ?? ($ordered[1] ?? $primary);
        $accent = $find(['accent', 'highlight', 'cta', 'action'])
            ?? ($ordered[2] ?? $secondary);
        $text = $find(['text', 'body', 'foreground', 'on ', 'copy'])
            ?? ($ordered[3] ?? null);
        $background = $find(['background', 'canvas', 'base', 'surface'])
            ?? ($ordered[4] ?? null);

        $out = [];
        if ($primary !== null) {
            $out['primary_color'] = $primary;
        }
        if ($secondary !== null) {
            $out['secondary_color'] = $secondary;
        }
        if ($accent !== null) {
            $out['accent_color'] = $accent;
        }
        if ($text !== null) {
            $out['text_color'] = $text;
        }
        if ($background !== null) {
            $out['background_color'] = $background;
        }

        if (is_array($fonts) && $fonts !== []) {
            foreach ($fonts as $font) {
                if (! is_array($font)) {
                    continue;
                }
                $family = $font['font'] ?? null;
                if (! is_string($family) || trim($family) === '') {
                    continue;
                }
                $fallback = isset($font['fallback']) && is_string($font['fallback']) && trim($font['fallback']) !== ''
                    ? trim($font['fallback'])
                    : 'system-ui, sans-serif';
                $out['font_family'] = trim($family).', '.$fallback;
                break;
            }
        }

        return $out === [] ? null : $out;
    }

    private function normalizeHex6(string $value): ?string
    {
        $v = trim($value);
        if ($v === '') {
            return null;
        }
        if ($v[0] !== '#') {
            $v = '#'.$v;
        }
        // #RRGGBBAA or #RRGGBB
        if (preg_match('/^#([0-9A-Fa-f]{8})$/', $v, $m) === 1) {
            $v = '#'.substr($m[1], 0, 6);
        }
        if (preg_match('/^#([0-9A-Fa-f]{3})$/', $v, $m) === 1) {
            $v = '#'.str_repeat($m[1][0], 2).str_repeat($m[1][1], 2).str_repeat($m[1][2], 2);
        }
        if (preg_match('/^#([0-9A-Fa-f]{6})$/', $v, $m) === 1) {
            return '#'.strtolower($m[1]);
        }

        return null;
    }
}
