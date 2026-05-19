<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class MlsProxyAuditLog extends Model
{
    protected $table = 'mls_proxy_audit_logs';

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
     * Revenue impact: immutable audit trail supports MLS compliance
     * investigations without service downtime (reduces legal/revocation risk).
     */
    protected function casts(): array
    {
        return [
            'logged_at' => 'datetime',
        ];
    }

    /**
     * @return BelongsTo<User, MlsProxyAuditLog>
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
