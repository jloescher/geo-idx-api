<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

#[Fillable([
    'user_id',
    'name',
    'template_type',
    'body_json',
    'audit_json',
    'usage_count',
    'last_used_at',
    'schedule_json',
])]
class AgentAlertTemplate extends Model
{
    public function casts(): array
    {
        return [
            'body_json' => 'array',
            'audit_json' => 'array',
            'usage_count' => 'integer',
            'last_used_at' => 'datetime',
            'schedule_json' => 'array',
        ];
    }

    /**
     * @return BelongsTo<User, $this>
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
