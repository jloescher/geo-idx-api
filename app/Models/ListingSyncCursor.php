<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;

#[Fillable([
    'dataset_slug',
    'last_bridge_modification_timestamp',
    'incremental_window_end',
    'replication_next_url',
    'replication_in_progress',
    'last_sync_finished_at',
])]
class ListingSyncCursor extends Model
{
    protected $table = 'listing_sync_cursors';

    protected $primaryKey = 'dataset_slug';

    public $incrementing = false;

    protected $keyType = 'string';

    /**
     * @return array<string, string>
     */
    public function casts(): array
    {
        return [
            'last_bridge_modification_timestamp' => 'immutable_datetime',
            'incremental_window_end' => 'immutable_datetime',
            'replication_in_progress' => 'boolean',
            'last_sync_finished_at' => 'immutable_datetime',
        ];
    }
}
