<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

#[Fillable(['domain_slug', 'is_active', 'mls_dataset', 'allowed_mls_datasets'])]
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
            'allowed_mls_datasets' => 'array',
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

    /**
     * Returns the MLS dataset for this domain, or null to use global default.
     */
    public function getMlsDataset(): ?string
    {
        if (is_string($this->mls_dataset) && trim($this->mls_dataset) !== '') {
            return trim($this->mls_dataset);
        }

        return null;
    }

    /**
     * Optional allowlist of Bridge dataset keys (e.g. stellar, miami). Null or empty means
     * any dataset from the global IDX catalog may be used for this domain.
     *
     * @return list<string>|null
     */
    public function getAllowedMlsDatasets(): ?array
    {
        $raw = $this->allowed_mls_datasets;
        if (! is_array($raw) || $raw === []) {
            return null;
        }

        $normalized = array_values(array_filter(array_map(
            static fn (mixed $v): string => is_string($v) ? trim($v) : '',
            $raw,
        ), static fn (string $v): bool => $v !== ''));

        return $normalized === [] ? null : $normalized;
    }
}
