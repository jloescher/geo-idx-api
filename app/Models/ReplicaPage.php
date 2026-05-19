<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;

#[Fillable([
    'provider',
    'dataset_slug',
    'mode',
    'status',
    'compressed_payload',
    'row_count',
    'bridge_url',
    'upstream_url',
    'odata_query',
    'batch_id',
    'fetched_at',
    'processed_at',
])]
class ReplicaPage extends Model
{
    public const STATUS_PENDING = 'pending';

    public const STATUS_PROCESSING = 'processing';

    public const STATUS_COMPLETED = 'completed';

    public const STATUS_FAILED = 'failed';

    protected $table = 'replica_pages';

    /**
     * @return array<string, string>
     */
    public function casts(): array
    {
        return [
            'odata_query' => 'array',
            'fetched_at' => 'immutable_datetime',
            'processed_at' => 'immutable_datetime',
            'row_count' => 'integer',
        ];
    }
}
