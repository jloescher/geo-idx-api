<?php

namespace App\Ghl\Webhooks\Models;

use Illuminate\Database\Eloquent\Model;

class GhlWebhookEvent extends Model
{
    protected $table = 'ghl_webhook_events';

    protected $fillable = [
        'webhook_id',
        'event_type',
        'ghl_app_id',
        'ghl_company_id',
        'ghl_location_id',
        'ghl_user_id',
        'payload',
        'processing_status',
        'processed_at',
        'processing_error',
        'handler_class',
        'received_at',
    ];

    protected function casts(): array
    {
        return [
            'payload' => 'array',
            'processed_at' => 'datetime',
            'received_at' => 'datetime',
        ];
    }
}
