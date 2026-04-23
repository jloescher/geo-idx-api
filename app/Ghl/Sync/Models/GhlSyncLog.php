<?php

namespace App\Ghl\Sync\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class GhlSyncLog extends Model
{
    protected $table = 'ghl_sync_logs';

    protected $fillable = [
        'ghl_location_id',
        'quantyra_lead_id',
        'ghl_contact_id',
        'ghl_opportunity_id',
        'sync_type',
        'lead_type',
        'sync_status',
        'retry_count',
        'max_retries',
        'error_message',
        'error_code',
        'request_payload',
        'response_payload',
        'completed_at',
    ];

    protected function casts(): array
    {
        return [
            'request_payload' => 'array',
            'response_payload' => 'array',
            'completed_at' => 'datetime',
        ];
    }

    public function lead(): BelongsTo
    {
        return $this->belongsTo(QuantyraLead::class, 'quantyra_lead_id');
    }
}
