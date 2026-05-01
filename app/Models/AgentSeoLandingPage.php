<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

#[Fillable([
    'user_id',
    'agent_search_id',
    'agent_share_link_id',
    'slug',
    'canonical_path',
    'canonical_url',
    'status',
    'published_at',
])]
class AgentSeoLandingPage extends Model
{
    public function casts(): array
    {
        return [
            'published_at' => 'datetime',
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
    public function search(): BelongsTo
    {
        return $this->belongsTo(AgentSearch::class, 'agent_search_id');
    }

    /**
     * @return BelongsTo<AgentShareLink, $this>
     */
    public function shareLink(): BelongsTo
    {
        return $this->belongsTo(AgentShareLink::class, 'agent_share_link_id');
    }
}
