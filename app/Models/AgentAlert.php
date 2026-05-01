<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

#[Fillable([
    'user_id',
    'agent_search_id',
    'name',
    'alert_type',
    'status',
    'schedule_json',
    'next_run_at',
])]
class AgentAlert extends Model
{
    public function casts(): array
    {
        return [
            'schedule_json' => 'array',
            'next_run_at' => 'datetime',
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

    /**
     * @return HasMany<AgentAlertRun, $this>
     */
    public function runs(): HasMany
    {
        return $this->hasMany(AgentAlertRun::class);
    }
}
