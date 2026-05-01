<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;

#[Fillable([
    'cache_key',
    'mls_code',
    'dataset_code',
    'scope',
    'payload_json',
    'checksum',
    'version_tag',
    'expires_at',
    'refreshed_at',
])]
class LookupCacheSnapshot extends Model
{
    protected $table = 'lookup_cache_snapshots';

    public function casts(): array
    {
        return [
            'payload_json' => 'array',
            'expires_at' => 'datetime',
            'refreshed_at' => 'datetime',
        ];
    }
}
