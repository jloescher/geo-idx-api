<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Model;

#[Fillable([
    'mls_code',
    'dataset_code',
    'source_field_key',
    'canonical_field_key',
    'display_label',
    'field_type',
    'category',
    'operators_json',
    'enum_values_json',
    'is_reso_standard',
    'is_custom_mls_field',
    'compatibility_tags_json',
    'lookup_version',
])]
class MlsFieldCatalog extends Model
{
    protected $table = 'mls_field_catalog';

    public function casts(): array
    {
        return [
            'operators_json' => 'array',
            'enum_values_json' => 'array',
            'is_reso_standard' => 'boolean',
            'is_custom_mls_field' => 'boolean',
            'compatibility_tags_json' => 'array',
        ];
    }
}
