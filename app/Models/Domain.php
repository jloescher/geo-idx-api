<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Attributes\Fillable;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

#[Fillable([
    'user_id',
    'domain_slug',
    'is_active',
    'mls_dataset',
    'allowed_mls_datasets',
    'verification_status',
    'verification_method',
    'txt_verification_name',
    'txt_verification_value',
    'txt_verified_at',
    'ghl_verified_at',
    'verification_checked_at',
    'verification_metadata',
])]
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
            'verification_metadata' => 'array',
            'txt_verified_at' => 'datetime',
            'ghl_verified_at' => 'datetime',
            'verification_checked_at' => 'datetime',
        ];
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
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
     * Revenue impact: stable default feed selection avoids mis-keyed Bridge calls that burn API quota.
     *
     * Compliance: domain-scoped defaults must align with `allowed_mls_datasets` enforced upstream.
     */
    public function resolveDefaultFeedCode(): string
    {
        $explicit = $this->getMlsDataset();
        if ($explicit !== null) {
            return $explicit;
        }

        $allowed = $this->getAllowedMlsDatasets();
        if ($allowed !== null && $allowed !== []) {
            return $allowed[0];
        }

        return (string) config('bridge.dataset', 'stellar');
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

    public function isVerifiedForWidgetAccess(): bool
    {
        return in_array((string) $this->verification_status, ['verified', 'verified_ghl'], true);
    }
}
