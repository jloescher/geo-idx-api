<?php

use App\Models\User;
use Illuminate\Foundation\Auth\Access\AuthorizesRequests;
use Illuminate\Support\Str;
use Laravel\Sanctum\PersonalAccessToken;
use Livewire\Volt\Component;

new class extends Component {
    use AuthorizesRequests;

    public string $tokenName = '';

    public ?string $newToken = null;

    /** @var list<array{id: int, name: string, abilities: list<string>, last_used_at: ?string, created_at: ?string}> */
    public array $tokens = [];

    public function mount(): void
    {
        /** @var User $user */
        $user = auth()->user();
        $this->loadTokens($user);
    }

    public function createToken(): void
    {
        $validated = $this->validate([
            'tokenName' => ['required', 'string', 'max:60'],
        ]);

        /** @var User $user */
        $user = auth()->user();

        $token = $user->createToken(Str::of($validated['tokenName'])->trim()->value(), ['idx:full']);

        $this->newToken = $token->plainTextToken;
        $this->tokenName = '';

        $this->loadTokens($user);
        $this->dispatch('token-created');
    }

    public function revokeToken(int $tokenId): void
    {
        /** @var User $user */
        $user = auth()->user();

        $token = PersonalAccessToken::find($tokenId);

        abort_if(
            $token === null
            || $token->tokenable_id !== $user->id
            || $token->tokenable_type !== $user::class,
            403,
        );

        $token->delete();
        $this->loadTokens($user);
        $this->dispatch('token-revoked');
    }

    /** @return array{abilitiesLabels: array<string, string>} */
    public function with(): array
    {
        /** @var User $user */
        $user = auth()->user();

        $abilitiesLabels = ['idx:full' => 'Full API access'];

        return ['abilitiesLabels' => $abilitiesLabels];
    }

    private function loadTokens(User $user): void
    {
        $this->tokens = $user->tokens()
            ->orderByDesc('id')
            ->limit(8)
            ->get()
            ->map(fn (PersonalAccessToken $token) => [
                'id' => $token->id,
                'name' => $token->name,
                'abilities' => $token->abilities ?? [],
                'last_used_at' => $token->last_used_at?->toDayDateTimeString(),
                'created_at' => $token->created_at?->toDayDateTimeString(),
            ])
            ->all();
    }
}; ?>

<div class="mt-8 rounded-2xl border border-white/10 bg-slate-900/70 p-6">
    <h2 class="text-xl font-semibold text-white">API Keys</h2>
    <p class="mt-1 text-sm text-slate-300">
        Manage tokens for the Bridge and GIS JSON API. Send <span class="font-mono text-xs">Authorization: Bearer …</span> together with
        <span class="font-mono text-xs">X-Domain-Slug: your-verified-domain.com</span> on every request (or <span class="font-mono text-xs">?domain=</span>).
    </p>

    {{-- Token creation form --}}
    <form wire:submit="createToken" class="mt-5 flex flex-col gap-3 sm:flex-row">
        <input
            type="text"
            wire:model.lazy="tokenName"
            placeholder="Token name (e.g. IDX Production Key)"
            maxlength="60"
            class="min-h-11 w-full rounded-lg border border-white/15 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:border-cyan-400/50 focus:outline-none focus:ring-1 focus:ring-cyan-400/30"
        />
        <button
            type="submit"
            wire:loading.attr="disabled"
            class="inline-flex min-h-11 items-center justify-center rounded-lg bg-emerald-500 px-4 py-2 text-sm font-semibold text-slate-950 shadow transition hover:bg-emerald-400 disabled:cursor-not-allowed disabled:opacity-60"
        >
            <span wire:loading.remove wire:target="createToken">Generate API token</span>
            <span wire:loading wire:target="createToken">
                <svg class="size-4 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                </svg>
                <span class="ml-2">Creating…</span>
            </span>
        </button>
    </form>

    @error('tokenName')
        <p class="mt-2 text-xs text-rose-400">{{ $message }}</p>
    @enderror

    {{-- One-time new token alert --}}
    @if ($newToken)
        <div class="mt-4 rounded-xl border border-amber-400/40 bg-amber-900/20 p-4" x-data="{ copied: false }">
            <p class="text-sm font-semibold text-amber-100">New API token created — shown once only</p>
            <p class="mt-2 break-all font-mono text-xs text-amber-50/90">{{ $newToken }}</p>
            <button
                type="button"
                @click="
                    navigator.clipboard.writeText('{{ $newToken }}');
                    copied = true;
                    setTimeout(() => copied = false, 2000);
                "
                class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-amber-400/30 px-3 py-1.5 text-xs font-semibold text-amber-200 transition hover:bg-amber-500/10"
            >
                <svg x-show="!copied" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"></path>
                </svg>
                <svg x-show="copied" x-cloak class="size-3.5 text-emerald-400" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"></path>
                </svg>
                <span x-text="copied ? 'Copied!' : 'Copy'"></span>
            </button>
        </div>
    @endif

    {{-- Token list --}}
    <div class="mt-5 space-y-3">
        @forelse ($tokens as $t)
            <div class="rounded-lg border border-white/10 bg-slate-950/70 p-3" x-data="{ visible: false }">
                <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                    <div class="min-w-0">
                        <p class="truncate text-sm font-semibold text-slate-100">{{ $t['name'] }}</p>
                        <p class="text-xs text-slate-400">
                            Last used: {{ $t['last_used_at'] ?? 'Never' }}
                            · Created: {{ $t['created_at'] ?? '—' }}
                        </p>
                        <div class="mt-1 flex flex-wrap items-center gap-1">
                            @foreach ($t['abilities'] as $ability)
                                <span class="inline-flex items-center rounded-full bg-slate-800 px-2 py-0.5 text-[10px] font-medium text-slate-300 ring-1 ring-white/10">
                                    {{ $abilitiesLabels[$ability] ?? $ability }}
                                </span>
                            @endforeach
                        </div>
                    </div>

                    <div class="flex items-center gap-2">
                        {{-- Revoke token --}}
                        <button
                            type="button"
                            wire:click="revokeToken({{ $t['id'] }})"
                            wire:confirm="Are you sure you want to revoke this token?"
                            wire:loading.attr="disabled"
                            class="inline-flex min-h-10 items-center rounded-md border border-rose-400/40 px-3 py-2 text-xs font-semibold text-rose-200 transition hover:bg-rose-500/10 disabled:cursor-not-allowed disabled:opacity-60"
                        >
                            <span wire:loading.remove wire:target="revokeToken({{ $t['id'] }})">Revoke</span>
                            <span wire:loading wire:target="revokeToken({{ $t['id'] }})">
                                <svg class="size-3.5 animate-spin" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                                </svg>
                            </span>
                        </button>
                    </div>
                </div>
            </div>
        @empty
            <p class="text-sm text-slate-400">No API keys created yet. Generate one above to get started.</p>
        @endforelse
    </div>
</div>
