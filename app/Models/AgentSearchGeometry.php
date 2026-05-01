<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

#[Fillable([
    'agent_search_id',
    'geometry_type',
    'mode',
    'geojson',
    'bbox_json',
    'area_m2',
])]
class AgentSearchGeometry extends Model
{
    public function casts(): array
    {
        return [
            'geojson' => 'array',
            'bbox_json' => 'array',
            'area_m2' => 'decimal:4',
        ];
    }

    /**
     * @return BelongsTo<AgentSearch, $this>
     */
    public function agentSearch(): BelongsTo
    {
        return $this->belongsTo(AgentSearch::class);
    }
}
