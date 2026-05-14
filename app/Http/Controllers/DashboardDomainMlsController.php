<?php

namespace App\Http\Controllers;

use App\Models\Domain;
use App\Services\Mls\MlsFeedResolver;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;

class DashboardDomainMlsController extends Controller
{
    public function __construct(
        private readonly MlsFeedResolver $mlsFeeds,
    ) {}

    public function update(Request $request, Domain $domain): RedirectResponse
    {
        $user = $request->user();
        if ($user === null || $domain->user_id !== $user->id) {
            abort(403);
        }

        $accepted = $this->mlsFeeds->validFormFeedInputs();

        $validated = $request->validate([
            'allowed_mls_datasets' => ['required', 'array', 'min:1'],
            'allowed_mls_datasets.*' => ['string', Rule::in($accepted)],
            'mls_dataset' => ['nullable', 'string', Rule::in($accepted)],
        ]);

        $allowed = array_values(array_unique(array_map(
            fn (mixed $v): string => $this->mlsFeeds->normalizeWireDatasetToCatalogKey(is_string($v) ? trim($v) : ''),
            $validated['allowed_mls_datasets'],
        )));

        $default = isset($validated['mls_dataset']) && is_string($validated['mls_dataset']) && $validated['mls_dataset'] !== ''
            ? $this->mlsFeeds->normalizeWireDatasetToCatalogKey(trim($validated['mls_dataset']))
            : $allowed[0];

        if (! in_array($default, $allowed, true)) {
            $default = $allowed[0];
        }

        $domain->forceFill([
            'allowed_mls_datasets' => $allowed,
            'mls_dataset' => $default,
        ])->save();

        return redirect('/dashboard?panel=domains')
            ->with('dashboard_status', 'MLS feed access updated for '.$domain->domain_slug.'.');
    }
}
