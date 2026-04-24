<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;

#[Fillable([
    'query_hash',
    'geojson',
    'county',
    'expires_at',
    'source_used',
    'source_generation',
])]
class GisCache extends Model
{
    /**
     * Revenue impact: persistent GIS rows amortize ArcGIS latency across thousands
     * of map pans, shrinking infra spend per engaged lead.
     */
    protected $table = 'gis_cache';

    public function casts(): array
    {
        return [
            'expires_at' => 'datetime',
        ];
    }
}
