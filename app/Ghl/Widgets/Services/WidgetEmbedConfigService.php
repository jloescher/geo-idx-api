<?php

namespace App\Ghl\Widgets\Services;

use App\Ghl\Api\Services\GhlBrandBoardPaletteResolver;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Ghl\Widgets\Models\GhlWidgetConfig;
use App\Models\User;
use App\Widgets\DirectSiteWidgetContext;
use Illuminate\Http\Request;

/**
 * Builds merged widget runtime config for embeds.
 *
 * Merge order (lowest to highest before client-side query overrides):
 * defaults → {@see GhlWidgetConfig} (install prefs + legacy colors) → GHL Brand Board palette (cached GET
 * brand-boards, {@see GhlBrandBoardPaletteResolver}) for marketplace installs → subscriber {@see User::$widget_palette}
 * when linked via {@see GhlRegisteredUrl::$quantyra_user_id} → loader query string (client).
 */
final class WidgetEmbedConfigService
{
    public function __construct(
        private readonly GhlBrandBoardPaletteResolver $brandBoardPalette,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function mergedConfig(Request $request): array
    {
        $ctx = $request->attributes->get('ghl_registered_url');
        $locationId = $ctx->ghl_location_id;

        $cfg = GhlWidgetConfig::query()->where('ghl_location_id', $locationId)->first();

        $paletteUser = $this->resolvePaletteUser($ctx);

        $base = [
            'location_id' => $locationId,
            'theme' => 'light',
            'primary_color' => '#2563EB',
            'secondary_color' => '#1E40AF',
            'accent_color' => null,
            'text_color' => null,
            'background_color' => null,
            'font_family' => 'Inter, system-ui, sans-serif',
            'listings_per_page' => 20,
            'map_enabled' => true,
            'default_search_area' => 'Tampa Bay',
            'property_types' => null,
            'min_price' => 0,
            'max_price' => null,
            'idx_platform' => config('ghl.urls.idx_platform'),
            'images_cdn' => config('ghl.urls.images'),
        ];

        if ($cfg !== null) {
            $this->applyGhlWidgetConfig($base, $cfg);
        }

        if ($ctx instanceof GhlRegisteredUrl) {
            $brandPalette = $this->brandBoardPalette->resolveForRegisteredUrl($ctx);
            if (is_array($brandPalette) && $brandPalette !== []) {
                $this->applyKeyedHexPalette($base, $brandPalette);
            }
        }

        if ($paletteUser instanceof User) {
            $this->applyUserPalette($base, $paletteUser);
        }

        if (($base['accent_color'] ?? null) === null || $base['accent_color'] === '') {
            $base['accent_color'] = $base['secondary_color'];
        }

        return $base;
    }

    /**
     * @param  array<string, mixed>  $base
     * @param  array<string, string>  $colors  keyed like primary_color, font_family, …
     */
    private function applyKeyedHexPalette(array &$base, array $colors): void
    {
        foreach (['primary_color', 'secondary_color', 'accent_color', 'text_color', 'background_color'] as $key) {
            if (isset($colors[$key]) && is_string($colors[$key]) && trim($colors[$key]) !== '') {
                $base[$key] = $this->normalizeHexColor((string) $colors[$key]);
            }
        }
        if (isset($colors['font_family']) && is_string($colors['font_family']) && trim($colors['font_family']) !== '') {
            $base['font_family'] = trim($colors['font_family']);
        }
    }

    /**
     * @param  array<string, mixed>  $base
     */
    private function applyUserPalette(array &$base, User $user): void
    {
        $p = $user->widget_palette;
        if (! is_array($p) || $p === []) {
            return;
        }

        if (isset($p['theme']) && is_string($p['theme']) && trim($p['theme']) !== '') {
            $base['theme'] = trim($p['theme']);
        }
        foreach (
            [
                'primary' => 'primary_color',
                'secondary' => 'secondary_color',
                'accent' => 'accent_color',
                'text' => 'text_color',
                'background' => 'background_color',
            ] as $jsonKey => $outKey
        ) {
            if (isset($p[$jsonKey]) && is_string($p[$jsonKey]) && trim($p[$jsonKey]) !== '') {
                $base[$outKey] = $this->normalizeHexColor((string) $p[$jsonKey]);
            }
        }
    }

    /**
     * @param  array<string, mixed>  $base
     */
    private function applyGhlWidgetConfig(array &$base, GhlWidgetConfig $cfg): void
    {
        $base['theme'] = $cfg->widget_theme ?: $base['theme'];
        $base['primary_color'] = $this->normalizeHexColor((string) ($cfg->primary_color ?: $base['primary_color']));
        $base['secondary_color'] = $this->normalizeHexColor((string) ($cfg->secondary_color ?: $base['secondary_color']));
        $base['font_family'] = $cfg->font_family ?: $base['font_family'];
        $base['listings_per_page'] = (int) $cfg->listings_per_page;
        $base['map_enabled'] = (bool) $cfg->map_enabled;
        $base['default_search_area'] = $cfg->default_search_area ?: $base['default_search_area'];
        $base['property_types'] = $cfg->property_types;
        $base['min_price'] = (int) $cfg->min_price;
        $base['max_price'] = $cfg->max_price !== null ? (int) $cfg->max_price : null;
    }

    private function normalizeHexColor(string $value): string
    {
        $v = trim($value);
        if ($v === '') {
            return $value;
        }
        if ($v[0] !== '#') {
            $v = '#'.$v;
        }

        return strlen($v) === 4
            ? '#'.str_repeat($v[1], 2).str_repeat($v[2], 2).str_repeat($v[3], 2)
            : $v;
    }

    private function resolvePaletteUser(mixed $ctx): ?User
    {
        if ($ctx instanceof DirectSiteWidgetContext) {
            return $ctx->user;
        }
        if ($ctx instanceof GhlRegisteredUrl && $ctx->quantyra_user_id !== null) {
            return User::query()->find($ctx->quantyra_user_id);
        }

        return null;
    }
}
