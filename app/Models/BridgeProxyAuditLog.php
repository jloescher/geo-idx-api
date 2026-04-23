<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class BridgeProxyAuditLog extends Model
{
    public $timestamps = false;

    protected $fillable = [
        'logged_at',
        'domain_slug',
        'token_name',
        'request_type',
        'listing_count',
        'ip_address',
        'user_id',
    ];

    /**
     * Revenue impact: immutable audit trail supports Stellar MLS compliance
     * investigations without service downtime (reduces legal/revocation risk).
     */
    protected function casts(): array
    {
        return [
            'logged_at' => 'datetime',
        ];
    }

    /**
     * @return BelongsTo<User, BridgeProxyAuditLog>
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
