<?php

namespace App\Ghl\Sync\Models;

use Illuminate\Database\Eloquent\Model;

class GhlLeadMapping extends Model
{
    protected $table = 'ghl_lead_mappings';

    protected $fillable = [
        'quantyra_lead_type',
        'creates_contact',
        'creates_opportunity',
        'opportunity_pipeline',
        'opportunity_stage',
        'default_tags',
        'domain_tag_prefix',
        'custom_field_mappings',
        'is_high_value',
    ];

    protected function casts(): array
    {
        return [
            'creates_contact' => 'boolean',
            'creates_opportunity' => 'boolean',
            'domain_tag_prefix' => 'boolean',
            'is_high_value' => 'boolean',
            'default_tags' => 'array',
            'custom_field_mappings' => 'array',
        ];
    }
}
