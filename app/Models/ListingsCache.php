<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class ListingsCache extends Model
{
    protected $table = 'listings_cache';

    protected $primaryKey = 'domain_slug';

    public $incrementing = false;

    protected $keyType = 'string';

    public $timestamps = false;

    protected $fillable = [
        'domain_slug',
        'compressed_data',
        'last_updated',
        'etag',
    ];

    /**
     * Revenue impact: gzip bytea reduces Postgres IO + network egress from DB
     * replicas, lowering infra cost per qualified lead session.
     */
    protected function casts(): array
    {
        return [
            'last_updated' => 'datetime',
        ];
    }
}
