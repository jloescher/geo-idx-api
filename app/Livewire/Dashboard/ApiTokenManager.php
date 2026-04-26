<?php

namespace App\Livewire\Dashboard;

use App\Billing\SubscriptionCatalog;
use App\Models\User;
use Illuminate\Contracts\View\View;
use Illuminate\Foundation\Auth\Access\AuthorizesRequests;
use Illuminate\Support\Str;
use Laravel\Sanctum\PersonalAccessToken;
use Livewire\Component;

class ApiTokenManager extends Component
{
    use AuthorizesRequests;

    public string $tokenName = '';

    /**
     * Plain text token value shown once after creation.
     */
    public ?string $newToken = null;

    /**
     * @var list<array{id: int, name: string, abilities: list<string>, last_used_at: ?string, created_at: ?string}>
     */
    public array $tokens = [];

    private SubscriptionCatalog $catalog;

    /**
     * Load user tokens and enforce plan-based access gate.
     */
    public function mount(SubscriptionCatalog $catalog): void
    {
        $this->catalog = $catalog;

        /** @var User $user */
        $user = auth()->user();
        abort_unless($this->catalog->userMayCreateIdxProxyApiTokens($user), 403);

        $this->loadTokens($user);
    }

    /**
     * Create a new Sanctum personal access token.
     */
    public function createToken(): void
    {
        $validated = $this->validate([
            'tokenName' => ['required', 'string', 'max:60'],
        ]);

        /** @var User $user */
        $user = auth()->user();

        $abilities = app(SubscriptionCatalog::class)->idxProxyAbilitiesForUser($user);
        abort_if($abilities === null, 403);

        $token = $user->createToken(Str::of($validated['tokenName'])->trim()->value(), $abilities);

        $this->newToken = $token->plainTextToken;
        $this->tokenName = '';

        $this->loadTokens($user);

        $this->dispatch('token-created');
    }

    /**
     * Revoke a token by its ID.
     */
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

    public function render(): View
    {
        /** @var User $user */
        $user = auth()->user();

        $abilitiesLabels = match (app(SubscriptionCatalog::class)->planKeyForUser($user)) {
            'mega' => ['idx:full' => 'Full API access'],
            'ultra' => ['idx:access' => 'API access (teaser)'],
            default => [],
        };

        return view('livewire.dashboard.api-token-manager', [
            'abilitiesLabels' => $abilitiesLabels,
        ]);
    }

    /**
     * Load the user's most recent tokens.
     *
     * Sanctum only stores hashed tokens, so the raw token is unavailable
     * after the initial creation response.
     */
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
}
