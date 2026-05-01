<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

#[Fillable([
    'agent_alert_id',
    'status',
    'metadata_json',
    'ran_at',
])]
class AgentAlertRun extends Model
{
    public function casts(): array
    {
        return [
            'metadata_json' => 'array',
            'ran_at' => 'datetime',
        ];
    }

    /**
     * @return BelongsTo<AgentAlert, $this>
     */
    public function agentAlert(): BelongsTo
    {
        return $this->belongsTo(AgentAlert::class);
    }
}
