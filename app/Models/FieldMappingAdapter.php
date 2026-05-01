<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;

#[Fillable([
    'mls_code',
    'dataset_code',
    'canonical_field_key',
    'source_field_key',
    'transform_in_json',
    'transform_out_json',
])]
class FieldMappingAdapter extends Model
{
    protected $table = 'field_mapping_adapters';

    public function casts(): array
    {
        return [
            'transform_in_json' => 'array',
            'transform_out_json' => 'array',
        ];
    }
}
