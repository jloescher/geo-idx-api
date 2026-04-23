<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

#[Fillable(['domain_slug', 'is_active'])]
class Domain extends Model
{
    /**
     * Revenue impact: `is_active` gates which hostnames may consume paid MLS
     * bandwidth; inactive domains stop accruing Bridge usage immediately.
     */
    public function casts(): array
    {
        return [
            'is_active' => 'boolean',
        ];
    }

    /**
     * @param  Builder<Domain>  $query
     * @return Builder<Domain>
     */
    public function scopeActive(Builder $query): Builder
    {
        return $query->where('is_active', true);
    }
}
