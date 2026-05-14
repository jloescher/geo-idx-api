<?php

namespace App\Services\Mls;

use App\Enums\MlsProvider;
use App\Models\Domain;
use Illuminate\Http\Request;
use Symfony\Component\HttpKernel\Exception\HttpException;

/**
 * Revenue impact: one resolver prevents duplicate Bridge calls from mismatched dataset vs. feed codes.
 *
 * Compliance: MLS GRID IDX + participant rules require explicit allowlist checks before any MLS egress.
 */
final class MlsFeedResolver
{
    /**
     * Full feed map: internal catalog key => definition (Bridge `bridge_{dataset}` keys only).
     *
     * @return array<string, array<string, mixed>>
     */
    public function feedsMap(): array
    {
        $feeds = [];
        $bridgeDatasets = config('bridge.datasets', ['stellar']);
        if (! is_array($bridgeDatasets) || $bridgeDatasets === []) {
            $bridgeDatasets = ['stellar'];
        }
        foreach ($bridgeDatasets as $dataset) {
            if (! is_string($dataset)) {
                continue;
            }
            $dataset = trim($dataset);
            if ($dataset === '') {
                continue;
            }
            $internalKey = 'bridge_'.$dataset;
            $feeds[$internalKey] = [
                'provider' => MlsProvider::STELLAR->value,
                'bridge_dataset_id' => $dataset,
            ];
        }

        return $feeds;
    }

    /**
     * Values accepted on dashboard forms and legacy rows: internal keys plus unprefixed Bridge dataset slugs.
     *
     * @return list<string>
     */
    public function validFormFeedInputs(): array
    {
        $keys = [];
        foreach ($this->feedsMap() as $catalogKey => $def) {
            $keys[] = $catalogKey;
            if (($def['provider'] ?? '') === MlsProvider::STELLAR->value && isset($def['bridge_dataset_id'])) {
                $keys[] = (string) $def['bridge_dataset_id'];
            }
        }

        return array_values(array_unique($keys));
    }

    /**
     * Map a wire value (`stellar`, `bridge_stellar`) to the internal catalog key.
     */
    public function normalizeWireDatasetToCatalogKey(string $wire): string
    {
        $wire = trim($wire);
        if ($wire === '') {
            throw new HttpException(400, 'MLS feed code is empty.');
        }

        $map = $this->feedsMap();
        if (isset($map[$wire])) {
            return $wire;
        }

        $prefixed = 'bridge_'.$wire;
        if (isset($map[$prefixed])) {
            return $prefixed;
        }

        throw new HttpException(400, "MLS feed '{$wire}' is not available. Available feeds: ".implode(', ', array_keys($map)));
    }

    /**
     * @return list<string>
     */
    public function catalogFeedCodes(): array
    {
        return array_values(array_keys($this->feedsMap()));
    }

    public function feedDefinition(string $feedCode): ?array
    {
        $map = $this->feedsMap();

        return $map[$feedCode] ?? null;
    }

    /**
     * Intersection of global catalog and domain (or token) restrictions.
     *
     * @return list<string>
     */
    public function enabledFeedsForRequest(Request $request): array
    {
        $catalog = $this->catalogFeedCodes();
        $allowed = $catalog;
        $domain = $request->attributes->get('bridge.domain');
        if ($domain instanceof Domain) {
            $restriction = $domain->getAllowedMlsDatasets();
            if ($restriction !== null) {
                $normalized = [];
                foreach ($restriction as $item) {
                    if (! is_string($item) || trim($item) === '') {
                        continue;
                    }
                    try {
                        $normalized[] = $this->normalizeWireDatasetToCatalogKey(trim($item));
                    } catch (HttpException) {
                        continue;
                    }
                }
                $allowed = array_values(array_intersect($normalized, $catalog));
            }
        }

        $allowedForUser = $request->attributes->get('bridge.allowed_datasets');
        if (is_array($allowedForUser) && $allowedForUser !== []) {
            $normalizedUser = [];
            foreach ($allowedForUser as $item) {
                if (! is_string($item) || trim($item) === '') {
                    continue;
                }
                try {
                    $normalizedUser[] = $this->normalizeWireDatasetToCatalogKey(trim($item));
                } catch (HttpException) {
                    continue;
                }
            }
            $allowed = array_values(array_intersect($allowed, $normalizedUser));
        }

        return $allowed;
    }

    /**
     * Enabled internal catalog feeds for a domain (no HTTP request).
     *
     * @return list<string>
     */
    public function enabledFeedsForDomain(Domain $domain, ?array $tokenAllowedDatasets = null): array
    {
        $catalog = $this->catalogFeedCodes();
        $allowed = $catalog;
        $restriction = $domain->getAllowedMlsDatasets();
        if ($restriction !== null) {
            $normalized = [];
            foreach ($restriction as $item) {
                if (! is_string($item) || trim($item) === '') {
                    continue;
                }
                try {
                    $normalized[] = $this->normalizeWireDatasetToCatalogKey(trim($item));
                } catch (HttpException) {
                    continue;
                }
            }
            $allowed = array_values(array_intersect($normalized, $catalog));
        }

        if ($tokenAllowedDatasets !== null && $tokenAllowedDatasets !== []) {
            $normalizedToken = [];
            foreach ($tokenAllowedDatasets as $item) {
                if (! is_string($item) || trim($item) === '') {
                    continue;
                }
                try {
                    $normalizedToken[] = $this->normalizeWireDatasetToCatalogKey(trim($item));
                } catch (HttpException) {
                    continue;
                }
            }
            $allowed = array_values(array_intersect($allowed, $normalizedToken));
        }

        return $allowed;
    }

    /**
     * Default internal catalog feed for cache refresh and domain defaults.
     *
     * @throws HttpException
     */
    public function resolveDefaultCatalogFeedForDomain(Domain $domain): string
    {
        $enabled = $this->enabledFeedsForDomain($domain);
        if ($enabled === []) {
            throw new HttpException(503, 'No MLS feed is enabled for this domain.');
        }

        $preferred = $domain->getMlsDataset();
        if (is_string($preferred) && trim($preferred) !== '') {
            $norm = $this->normalizeWireDatasetToCatalogKey(trim($preferred));
            if (in_array($norm, $enabled, true)) {
                return $norm;
            }
        }

        $defaultInternal = $this->normalizeWireDatasetToCatalogKey(trim((string) config('bridge.dataset', 'stellar')));
        if (in_array($defaultInternal, $enabled, true)) {
            return $defaultInternal;
        }

        return $enabled[0];
    }

    /**
     * Resolve the MLS feed code for the current request (internal catalog key).
     */
    public function resolveFeedCode(Request $request): string
    {
        $enabled = $this->enabledFeedsForRequest($request);
        if ($enabled === []) {
            throw new HttpException(503, 'No MLS feed is enabled for this site.');
        }

        $queryFeed = $request->query('dataset');
        if (is_string($queryFeed) && trim($queryFeed) !== '') {
            $code = $this->normalizeWireDatasetToCatalogKey(trim($queryFeed));
            $this->validateFeedForSite($code, $enabled);

            return $code;
        }

        $domain = $request->attributes->get('bridge.domain');
        if ($domain instanceof Domain) {
            $defaultDataset = $domain->getMlsDataset();
            if ($defaultDataset !== null) {
                $code = $this->normalizeWireDatasetToCatalogKey(trim($defaultDataset));
                $this->validateFeedForSite($code, $enabled);

                return $code;
            }
        }

        $default = trim((string) config('bridge.dataset', 'stellar'));
        $defaultInternal = $this->normalizeWireDatasetToCatalogKey($default);
        if (in_array($defaultInternal, $enabled, true)) {
            return $defaultInternal;
        }

        return $enabled[0];
    }

    public function validateFeedCode(string $feedCode): void
    {
        $normalized = $this->normalizeWireDatasetToCatalogKey($feedCode);
        $this->validateFeedForSite($normalized, $this->catalogFeedCodes());
    }

    /**
     * @param  list<string>  $enabledForSite
     */
    public function validateFeedForSite(string $feedCode, array $enabledForSite): void
    {
        $normalized = $this->normalizeWireDatasetToCatalogKey($feedCode);
        $catalog = $this->catalogFeedCodes();
        if (! in_array($normalized, $catalog, true)) {
            throw new HttpException(400, "MLS feed '{$feedCode}' is not available. Available feeds: ".implode(', ', $catalog));
        }

        if (! in_array($normalized, $enabledForSite, true)) {
            throw new HttpException(403, "MLS feed '{$feedCode}' is not enabled for this site.");
        }
    }
}
