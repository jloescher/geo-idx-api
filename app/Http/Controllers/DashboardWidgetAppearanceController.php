<?php

namespace App\Http\Controllers;

use App\Billing\SubscriptionCatalog;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Schema;
use Illuminate\Support\Facades\Validator;

/**
 * Persists subscriber-level widget color palette for all embed surfaces.
 */
class DashboardWidgetAppearanceController extends Controller
{
    public function __construct(
        private readonly SubscriptionCatalog $catalog,
    ) {}

    public function __invoke(Request $request): RedirectResponse
    {
        $user = $request->user();
        if ($user === null) {
            abort(403);
        }

        $planKey = $this->catalog->planKeyForUser($user) ?? '';
        if (! in_array($planKey, ['pro', 'smart', 'ultra', 'mega'], true)) {
            abort(403, 'Widget appearance is available on Pro and higher plans.');
        }

        if (! Schema::hasColumn('users', 'widget_palette')) {
            return redirect(route('dashboard.index', [], false))
                ->withErrors([
                    'widget_palette' => 'Widget appearance is temporarily unavailable while a database migration is pending. Please run the latest migrations and try again.',
                ]);
        }

        $validator = Validator::make($request->all(), [
            'primary' => ['required', 'string', 'regex:/^#?[0-9A-Fa-f]{6}$/i'],
            'secondary' => ['required', 'string', 'regex:/^#?[0-9A-Fa-f]{6}$/i'],
            'accent' => ['nullable', 'string', 'regex:/^#?[0-9A-Fa-f]{6}$/i'],
            'text' => ['required', 'string', 'regex:/^#?[0-9A-Fa-f]{6}$/i'],
            'background' => ['required', 'string', 'regex:/^#?[0-9A-Fa-f]{6}$/i'],
            'theme' => ['required', 'string', 'in:light,dark'],
        ]);
        if ($validator->fails()) {
            return redirect(route('dashboard.index', [], false))
                ->withErrors($validator)
                ->withInput();
        }
        $validated = $validator->validated();

        $norm = static function (string $hex): string {
            $h = trim($hex);

            return str_starts_with($h, '#') ? $h : '#'.$h;
        };

        $palette = [
            'primary' => $norm($validated['primary']),
            'secondary' => $norm($validated['secondary']),
            'text' => $norm($validated['text']),
            'background' => $norm($validated['background']),
            'theme' => $validated['theme'],
        ];
        if (! empty($validated['accent'])) {
            $palette['accent'] = $norm((string) $validated['accent']);
        }

        $user->forceFill(['widget_palette' => $palette])->save();

        return redirect(route('dashboard.index', [], false))
            ->with('dashboard_status', 'Widget appearance saved. Embeds pick up colors on the next config load.');
    }
}
