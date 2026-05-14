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
     * Full feed map: flat code => definition (merged from `config('bridge.datasets')` + `mls.spark_feeds`).
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
            $feeds[$dataset] = [
                'provider' => MlsProvider::Bridge->value,
                'bridge_dataset_id' => $dataset,
            ];
        }

        $spark = config('mls.spark_feeds', []);
        if (is_array($spark)) {
            foreach ($spark as $code => $row) {
                if (! is_string($code) || trim($code) === '' || ! is_array($row)) {
                    continue;
                }
                $feeds[trim($code)] = $row;
            }
        }

        return $feeds;
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
        $domain = $request->attributes->get('bridge.domain');
        $allowed = $catalog;
        if ($domain instanceof Domain) {
            $restriction = $domain->getAllowedMlsDatasets();
            if ($restriction !== null) {
                $allowed = array_values(array_intersect($restriction, $catalog));
            }
        }

        $allowedForUser = $request->attributes->get('bridge.allowed_datasets');
        if (is_array($allowedForUser) && $allowedForUser !== []) {
            $allowed = array_values(array_intersect($allowed, $allowedForUser));
        }

        return $allowed;
    }

    /**
     * Resolve the MLS feed code for the current request (flat key from {@see feedsMap()}).
     */
    public function resolveFeedCode(Request $request): string
    {
        $enabled = $this->enabledFeedsForRequest($request);
        if ($enabled === []) {
            throw new HttpException(503, 'No MLS feed is enabled for this site.');
        }

        $queryFeed = $request->query('dataset');
        if (is_string($queryFeed) && trim($queryFeed) !== '') {
            $code = trim($queryFeed);
            $this->validateFeedForSite($code, $enabled);

            return $code;
        }

        $domain = $request->attributes->get('bridge.domain');
        if ($domain instanceof Domain) {
            $defaultDataset = $domain->getMlsDataset();
            if ($defaultDataset !== null) {
                $this->validateFeedForSite($defaultDataset, $enabled);

                return $defaultDataset;
            }
        }

        $default = (string) config('bridge.dataset', 'stellar');
        if (in_array($default, $enabled, true)) {
            return $default;
        }

        return $enabled[0];
    }

    public function validateFeedCode(string $feedCode): void
    {
        $this->validateFeedForSite($feedCode, $this->catalogFeedCodes());
    }

    /**
     * @param  list<string>  $enabledForSite
     */
    public function validateFeedForSite(string $feedCode, array $enabledForSite): void
    {
        $catalog = $this->catalogFeedCodes();
        if (! in_array($feedCode, $catalog, true)) {
            throw new HttpException(400, "MLS feed '{$feedCode}' is not available. Available feeds: ".implode(', ', $catalog));
        }

        if (! in_array($feedCode, $enabledForSite, true)) {
            throw new HttpException(403, "MLS feed '{$feedCode}' is not enabled for this site.");
        }
    }
}
