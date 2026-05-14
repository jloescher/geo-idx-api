<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class QuantyraLead extends Model
{
    protected $table = 'quantyra_leads';

    protected $fillable = [
        'ghl_location_id',
        'lead_type',
        'source',
        'payload',
        'listing_id',
        'quantyra_domain',
    ];

    protected function casts(): array
    {
        return [
            'payload' => 'array',
        ];
    }
}
