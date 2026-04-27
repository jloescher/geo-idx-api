<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class LeadSavedSearch extends Model
{
    protected $fillable = [
        'user_id',
        'name',
        'filters',
        'is_alert_enabled',
        'alert_frequency',
        'last_alerted_at',
    ];

    protected function casts(): array
    {
        return [
            'filters' => 'array',
            'is_alert_enabled' => 'boolean',
            'last_alerted_at' => 'datetime',
        ];
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
