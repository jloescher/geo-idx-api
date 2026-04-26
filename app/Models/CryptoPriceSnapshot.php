<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;

#[Fillable([
    'asset_id',
    'vs_currency',
    'price',
    'as_of',
    'payload',
])]
class CryptoPriceSnapshot extends Model
{
    public function casts(): array
    {
        return [
            'price' => 'decimal:8',
            'as_of' => 'datetime',
            'payload' => 'array',
        ];
    }
}
