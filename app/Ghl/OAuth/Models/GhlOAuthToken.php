<?php

namespace App\Ghl\OAuth\Models;

use App\Ghl\Sync\Models\GhlInstalledLocation;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\SoftDeletes;

class GhlOAuthToken extends Model
{
    use SoftDeletes;

    protected $table = 'ghl_oauth_tokens';

    protected $fillable = [
        'ghl_company_id',
        'ghl_location_id',
        'ghl_user_id',
        'access_token',
        'refresh_token',
        'refresh_token_id',
        'user_type',
        'expires_at',
        'refresh_expires_at',
        'scopes',
        'is_bulk_install',
        'installed_at',
        'last_refreshed_at',
        'status',
        'revoked_at',
        'revoke_reason',
        'access_token_hash',
    ];

    protected function casts(): array
    {
        return [
            'expires_at' => 'datetime',
            'refresh_expires_at' => 'datetime',
            'installed_at' => 'datetime',
            'last_refreshed_at' => 'datetime',
            'revoked_at' => 'datetime',
            'is_bulk_install' => 'boolean',
            'access_token' => 'encrypted',
            'refresh_token' => 'encrypted',
        ];
    }

    public function installedLocations(): HasMany
    {
        return $this->hasMany(GhlInstalledLocation::class, 'ghl_oauth_token_id');
    }

    public function registeredUrls(): HasMany
    {
        return $this->hasMany(GhlRegisteredUrl::class, 'ghl_oauth_token_id');
    }

    public function isActive(): bool
    {
        return $this->status === 'active' && $this->expires_at?->isFuture() === true;
    }
}
