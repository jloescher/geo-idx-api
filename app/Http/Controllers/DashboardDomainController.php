<?php

namespace App\Http\Controllers;

use App\Models\Domain;
use App\Services\DomainOwnershipVerifier;
use App\Services\Mls\MlsFeedResolver;
use Illuminate\Database\QueryException;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Schema;
use Illuminate\Validation\Rule;

class DashboardDomainController extends Controller
{
    private ?bool $domainsHaveUserIdColumn = null;

    public function __construct(
        private readonly DomainOwnershipVerifier $ownershipVerifier,
        private readonly MlsFeedResolver $mlsFeeds,
    ) {}

    public function store(Request $request): RedirectResponse
    {
        $user = $request->user();
        if ($user === null) {
            return redirect('/dashboard');
        }

        $normalizedDomain = $this->normalizeDomainInput((string) $request->input('domain_slug', ''));
        $request->merge(['domain_slug' => $normalizedDomain]);

        $accepted = $this->mlsFeeds->validFormFeedInputs();

        $validated = $request->validate([
            'domain_slug' => [
                'required',
                'string',
                'max:255',
                'regex:/^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$/',
                Rule::unique('domains', 'domain_slug'),
            ],
            'allowed_mls_datasets' => ['required', 'array', 'min:1'],
            'allowed_mls_datasets.*' => ['string', Rule::in($accepted)],
            'mls_dataset' => ['nullable', 'string', Rule::in($accepted)],
        ], [
            'domain_slug.regex' => 'Use a valid hostname like example.com (no protocol or path).',
            'allowed_mls_datasets.required' => 'Select at least one MLS feed.',
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

        $attributes = [
            'domain_slug' => mb_strtolower((string) $validated['domain_slug']),
            'is_active' => true,
            'mls_dataset' => $default,
            'allowed_mls_datasets' => $allowed,
            'verification_status' => 'pending',
        ];
        if ($this->domainsTableHasUserIdColumn()) {
            $attributes['user_id'] = $user->id;
        }

        Domain::query()->create($attributes);

        $domain = Domain::query()
            ->when(
                $this->domainsTableHasUserIdColumn(),
                fn ($query) => $query->where('user_id', $user->id)
            )
            ->where('domain_slug', mb_strtolower((string) $validated['domain_slug']))
            ->latest('id')
            ->first();
        if ($domain !== null) {
            $this->ownershipVerifier->issueTxtChallenge($domain);
        }

        return redirect('/dashboard')
            ->with('dashboard_status', 'Domain added. Publish the TXT record shown below to verify ownership, then your Production API key will be generated automatically.');
    }

    public function destroy(Request $request, Domain $domain): RedirectResponse
    {
        $user = $request->user();
        if ($user === null || $domain->user_id !== $user->id) {
            abort(403);
        }

        $domain->delete();

        return redirect('/dashboard')
            ->with('dashboard_status', 'Domain removed from approved hostnames.');
    }

    public function verifyTxt(Request $request, Domain $domain): RedirectResponse
    {
        $user = $request->user();
        if ($user === null || $domain->user_id !== $user->id) {
            abort(403);
        }

        $this->ownershipVerifier->issueTxtChallenge($domain);
        $verified = $this->ownershipVerifier->verifyTxtRecord($domain);

        if (! $verified) {
            return redirect('/dashboard')->with(
                'dashboard_status',
                'TXT verification not detected yet. Check the record at your DNS host; propagation can take up to 15 minutes.',
            );
        }

        $redirect = redirect('/dashboard')->with(
            'dashboard_status',
            'Domain verified. Your API is ready to use.',
        );

        $hasProductionToken = $user->tokens()
            ->whereRaw('LOWER(name) = ?', ['production'])
            ->exists();

        if (! $hasProductionToken) {
            $token = $user->createToken('Production', ['idx:full']);
            $redirect->with('dashboard_new_api_token', $token->plainTextToken);
        }

        return $redirect;
    }

    private function normalizeDomainInput(string $raw): string
    {
        $candidate = trim($raw);
        if ($candidate === '') {
            return '';
        }

        if (str_contains($candidate, '://')) {
            $host = parse_url($candidate, PHP_URL_HOST);
            $candidate = is_string($host) ? $host : $candidate;
        } elseif (str_contains($candidate, '/')) {
            $host = parse_url('http://'.$candidate, PHP_URL_HOST);
            $candidate = is_string($host) ? $host : $candidate;
        }

        return rtrim(mb_strtolower($candidate), '.');
    }

    private function domainsTableHasUserIdColumn(): bool
    {
        if ($this->domainsHaveUserIdColumn !== null) {
            return $this->domainsHaveUserIdColumn;
        }

        try {
            $this->domainsHaveUserIdColumn = Schema::hasColumn('domains', 'user_id');
        } catch (QueryException) {
            $this->domainsHaveUserIdColumn = false;
        }

        return $this->domainsHaveUserIdColumn;
    }
}
