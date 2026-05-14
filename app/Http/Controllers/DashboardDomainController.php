<?php

namespace App\Http\Controllers;

use App\Models\Domain;
use App\Services\DomainOwnershipVerifier;
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
    ) {}

    public function store(Request $request): RedirectResponse
    {
        $user = $request->user();
        if ($user === null) {
            return redirect('/dashboard');
        }

        $normalizedDomain = $this->normalizeDomainInput((string) $request->input('domain_slug', ''));
        $request->merge(['domain_slug' => $normalizedDomain]);

        $validated = $request->validate([
            'domain_slug' => [
                'required',
                'string',
                'max:255',
                'regex:/^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$/',
                Rule::unique('domains', 'domain_slug'),
            ],
        ], [
            'domain_slug.regex' => 'Use a valid hostname like example.com (no protocol or path).',
        ]);

        $attributes = [
            'domain_slug' => mb_strtolower((string) $validated['domain_slug']),
            'is_active' => true,
            'mls_dataset' => 'stellar',
            'allowed_mls_datasets' => ['stellar'],
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
            ->with('dashboard_status', 'Domain added. Publish the TXT record to verify ownership and enable API access.');
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

        return redirect('/dashboard')->with(
            'dashboard_status',
            $verified
                ? 'TXT verification succeeded. You can bind this domain to API tokens with X-Domain-Slug.'
                : 'TXT verification not detected yet. DNS propagation can take a few minutes.'
        );
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
