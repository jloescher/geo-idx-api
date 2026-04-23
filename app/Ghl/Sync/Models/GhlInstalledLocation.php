<?php

namespace App\Ghl\Sync\Models;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class GhlInstalledLocation extends Model
{
    protected $table = 'ghl_installed_locations';

    protected $fillable = [
        'ghl_oauth_token_id',
        'ghl_company_id',
        'ghl_location_id',
        'location_name',
        'location_address',
        'location_timezone',
        'is_whitelabel',
        'whitelabel_domain',
        'whitelabel_logo_url',
        'subscription_status',
        'subscription_id',
        'subscription_updated_at',
        'mls_request_count',
        'lead_count',
        'last_activity_at',
        'installed_at',
        'uninstalled_at',
        'status',
    ];

    protected function casts(): array
    {
        return [
            'is_whitelabel' => 'boolean',
            'subscription_updated_at' => 'datetime',
            'last_activity_at' => 'datetime',
            'installed_at' => 'datetime',
            'uninstalled_at' => 'datetime',
        ];
    }

    public function oauthToken(): BelongsTo
    {
        return $this->belongsTo(GhlOAuthToken::class, 'ghl_oauth_token_id');
    }
}
