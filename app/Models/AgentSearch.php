<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

#[Fillable([
    'user_id',
    'name',
    'search_state_json',
    'mls_scope_json',
    'is_template',
    'source',
])]
class AgentSearch extends Model
{
    public function casts(): array
    {
        return [
            'search_state_json' => 'array',
            'mls_scope_json' => 'array',
            'is_template' => 'boolean',
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
     * @return HasMany<AgentSearchFilter, $this>
     */
    public function filters(): HasMany
    {
        return $this->hasMany(AgentSearchFilter::class);
    }

    /**
     * @return HasMany<AgentSearchGeometry, $this>
     */
    public function geometries(): HasMany
    {
        return $this->hasMany(AgentSearchGeometry::class);
    }
}
