<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

#[Fillable([
    'user_id',
    'agent_search_id',
    'token',
    'attribution_json',
    'expires_at',
])]
class AgentShareLink extends Model
{
    public function casts(): array
    {
        return [
            'attribution_json' => 'array',
            'expires_at' => 'datetime',
        ];
    }

    /**
     * @return BelongsTo<User, $this>
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    /**
     * @return BelongsTo<AgentSearch, $this>
     */
    public function agentSearch(): BelongsTo
    {
        return $this->belongsTo(AgentSearch::class);
    }
}
