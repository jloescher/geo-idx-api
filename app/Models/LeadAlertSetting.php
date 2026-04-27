<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class LeadAlertSetting extends Model
{
    protected $fillable = [
        'user_id',
        'enabled',
        'frequency',
        'rules',
        'last_sent_at',
    ];

    protected function casts(): array
    {
        return [
            'enabled' => 'boolean',
            'rules' => 'array',
            'last_sent_at' => 'datetime',
        ];
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
