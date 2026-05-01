<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

#[Fillable([
    'agent_search_id',
    'canonical_field_key',
    'operator',
    'value_json',
    'applies_to_mls_json',
])]
class AgentSearchFilter extends Model
{
    public function casts(): array
    {
        return [
            'value_json' => 'array',
            'applies_to_mls_json' => 'array',
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
