<?php

namespace App\Http\Controllers;

use App\Billing\SubscriptionCatalog;
use App\Models\Domain;
use App\Services\DomainOwnershipVerifier;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;

class DashboardDomainController extends Controller
{
    public function __construct(
        private readonly SubscriptionCatalog $catalog,
        private readonly DomainOwnershipVerifier $ownershipVerifier,
    ) {}

    public function store(Request $request): RedirectResponse
    {
        $user = $request->user();
        if ($user === null) {
            return redirect('/dashboard');
        }

        $domainLimit = $this->catalog->domainLimitForUser($user);
        $activeDomainCount = Domain::query()
            ->where('user_id', $user->id)
            ->where('is_active', true)
            ->count();

        if (is_int($domainLimit) && $activeDomainCount >= $domainLimit) {
            return redirect('/dashboard')
                ->withErrors([
                    'domain_slug' => "You've reached your plan limit of {$domainLimit} domains. Remove a domain or upgrade to add more.",
                ])
                ->withInput();
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

        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => mb_strtolower((string) $validated['domain_slug']),
            'is_active' => true,
            'mls_dataset' => 'stellar',
            'allowed_mls_datasets' => ['stellar'],
            'verification_status' => 'pending',
        ]);

        $domain = Domain::query()
            ->where('user_id', $user->id)
            ->where('domain_slug', mb_strtolower((string) $validated['domain_slug']))
            ->latest('id')
            ->first();
        if ($domain !== null) {
            $this->ownershipVerifier->issueTxtChallenge($domain);
        }

        return redirect('/dashboard')
            ->with('dashboard_status', 'Domain added. Publish the TXT record (or connect GHL) to enable widget access.');
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
                ? 'TXT verification succeeded. Widget access for this domain is now enabled.'
                : 'TXT verification not detected yet. DNS propagation can take a few minutes.'
        );
    }

    public function verifyGhl(Request $request, Domain $domain): RedirectResponse
    {
        $user = $request->user();
        if ($user === null || $domain->user_id !== $user->id) {
            abort(403);
        }

        $verified = $this->ownershipVerifier->verifyGhlAttachment($domain, $user);

        return redirect('/dashboard')->with(
            'dashboard_status',
            $verified
                ? 'LeadConnector/GHL verification succeeded for this domain.'
                : 'No matching LeadConnector/GHL site attachment found for this domain yet.'
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
}
