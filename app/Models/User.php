<?php

namespace App\Models;

// use Illuminate\Contracts\Auth\MustVerifyEmail;
use Database\Factories\UserFactory;
use Filament\Models\Contracts\FilamentUser;
use Filament\Panel;
use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Attributes\Hidden;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Foundation\Auth\User as Authenticatable;
use Illuminate\Notifications\Notifiable;
use Laravel\Sanctum\HasApiTokens;
use RuntimeException;

#[Fillable([
    'name',
    'email',
    'password',
    'widget_palette',
    'mls_id',
    'mls_email',
    'assigned_mls_datasets',
    'mls_membership_status',
    'mls_membership_verified_at',
    'mls_membership_next_reverify_at',
    'mls_membership_last_error',
])]
#[Hidden(['password', 'remember_token', 'widget_embed_site_key'])]
class User extends Authenticatable implements FilamentUser
{
    /** @use HasFactory<UserFactory> */
    use HasApiTokens, HasFactory, Notifiable;

    /**
     * Get the attributes that should be cast.
     *
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'email_verified_at' => 'datetime',
            'password' => 'hashed',
            'widget_palette' => 'array',
            'assigned_mls_datasets' => 'array',
            'mls_membership_verified_at' => 'datetime',
            'mls_membership_next_reverify_at' => 'datetime',
        ];
    }

    public function domains(): HasMany
    {
        return $this->hasMany(Domain::class);
    }

    /**
     * @return list<string>
     */
    public function canAccessPanel(Panel $panel): bool
    {
        return $panel->getId() === 'dashboard';
    }

    public function assignedDatasets(): array
    {
        $datasets = $this->assigned_mls_datasets;
        if (! is_array($datasets) || $datasets === []) {
            return ['stellar'];
        }

        $normalized = array_values(array_filter(array_map(
            static fn (mixed $v): string => is_string($v) ? trim($v) : '',
            $datasets,
        ), static fn (string $v): bool => $v !== ''));

        return $normalized === [] ? ['stellar'] : $normalized;
    }

    protected static function booted(): void
    {
        static::deleting(function (User $user): void {
            if (app()->isProduction() && ! (bool) config('debug_audit.allow_user_deletion_in_production')) {
                throw new RuntimeException(
                    'User deletion is disabled in production. Set ALLOW_USER_DELETION_IN_PRODUCTION=true only during an explicit maintenance window.'
                );
            }
        });
    }
}
