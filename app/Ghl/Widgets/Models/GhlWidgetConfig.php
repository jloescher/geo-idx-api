<?php

namespace App\Ghl\Widgets\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class GhlWidgetConfig extends Model
{
    protected $table = 'ghl_widget_configs';

    protected $fillable = [
        'ghl_location_id',
        'ghl_registered_url_id',
        'widget_theme',
        'primary_color',
        'secondary_color',
        'font_family',
        'default_widget_type',
        'listings_per_page',
        'map_enabled',
        'show_lead_form',
        'default_search_area',
        'property_types',
        'min_price',
        'max_price',
        'gate_after_views',
        'require_otp',
    ];

    protected function casts(): array
    {
        return [
            'property_types' => 'array',
            'map_enabled' => 'boolean',
            'show_lead_form' => 'boolean',
            'require_otp' => 'boolean',
        ];
    }

    public function registeredUrl(): BelongsTo
    {
        return $this->belongsTo(GhlRegisteredUrl::class, 'ghl_registered_url_id');
    }
}
