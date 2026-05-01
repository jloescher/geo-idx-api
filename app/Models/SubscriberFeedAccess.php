<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

#[Fillable([
    'user_id',
    'mls_code',
    'feed_id',
    'dataset_code',
    'status',
    'permissions_json',
    'connected_at',
    'last_verified_at',
])]
class SubscriberFeedAccess extends Model
{
    protected $table = 'subscriber_feed_access';

    public function casts(): array
    {
        return [
            'permissions_json' => 'array',
            'connected_at' => 'datetime',
            'last_verified_at' => 'datetime',
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
