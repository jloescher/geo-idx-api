<?php

namespace App\Ghl\Widgets\Models;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Models\User;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasOne;

class GhlRegisteredUrl extends Model
{
    protected $table = 'ghl_registered_urls';

    protected $fillable = [
        'ghl_oauth_token_id',
        'ghl_location_id',
        'ghl_company_id',
        'primary_url',
        'additional_urls',
        'urls_validated',
        'validation_errors',
        'widget_api_key',
        'widget_access_enabled',
        'integration_type',
        'mls_agreement_acknowledged',
        'mls_compliance_verified',
        'stellar_mls_approved',
        'quantyra_user_id',
    ];

    protected function casts(): array
    {
        return [
            'additional_urls' => 'array',
            'validation_errors' => 'array',
            'urls_validated' => 'boolean',
            'widget_access_enabled' => 'boolean',
            'mls_agreement_acknowledged' => 'boolean',
            'mls_compliance_verified' => 'boolean',
            'stellar_mls_approved' => 'boolean',
        ];
    }

    public function oauthToken(): BelongsTo
    {
        return $this->belongsTo(GhlOAuthToken::class, 'ghl_oauth_token_id');
    }

    public function quantyraUser(): BelongsTo
    {
        return $this->belongsTo(User::class, 'quantyra_user_id');
    }

    public function widgetConfig(): HasOne
    {
        return $this->hasOne(GhlWidgetConfig::class, 'ghl_registered_url_id');
    }

    public function allowedOrigins(): array
    {
        $origins = array_filter([$this->primary_url]);
        foreach ($this->additional_urls ?? [] as $u) {
            $origins[] = $u;
        }

        return array_values(array_unique(array_map(fn ($u) => rtrim((string) $u, '/'), $origins)));
    }
}
