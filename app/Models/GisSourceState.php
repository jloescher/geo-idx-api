<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;

#[Fillable([
    'source_key',
    'fingerprint',
    'generation',
    'last_checked_at',
    'last_changed_at',
])]
class GisSourceState extends Model
{
    /**
     * Revenue impact: cheap ArcGIS layer fingerprints invalidate only when hosts
     * republish, so long-lived origin cache stays honest without polling every bbox.
     */
    protected $table = 'gis_source_states';

    protected $primaryKey = 'source_key';

    public $incrementing = false;

    protected $keyType = 'string';

    public function casts(): array
    {
        return [
            'last_checked_at' => 'datetime',
            'last_changed_at' => 'datetime',
        ];
    }
}
