<?php

namespace App\Services\Bridge;

use App\Models\Domain;
use Illuminate\Http\Request;
use Symfony\Component\HttpKernel\Exception\HttpException;

final class MlsDatasetResolver
{
    /**
     * Resolve the MLS dataset for the current request.
     *
     * Priority: query param > authenticated domain default > global config default.
     * Dataset must appear in {@see getAvailableDatasets()} and in the domain's enabled set
     * when {@see Domain::getAllowedMlsDatasets()} restricts feeds.
     */
    public function resolveDataset(Request $request): string
    {
        $domain = $request->attributes->get('bridge.domain');
        $allowed = $this->datasetsEnabledForDomain($domain);
        $allowedForUser = $request->attributes->get('bridge.allowed_datasets');
        if (is_array($allowedForUser) && $allowedForUser !== []) {
            $allowed = array_values(array_intersect($allowed, $allowedForUser));
        }

        if ($allowed === []) {
            throw new HttpException(503, 'No MLS dataset is enabled for this site.');
        }

        $queryDataset = $request->query('dataset');
        if (is_string($queryDataset) && trim($queryDataset) !== '') {
            $dataset = trim($queryDataset);
            $this->validateDatasetForSite($dataset, $allowed);

            return $dataset;
        }

        if ($domain instanceof Domain) {
            $domainDataset = $domain->getMlsDataset();
            if ($domainDataset !== null) {
                $this->validateDatasetForSite($domainDataset, $allowed);

                return $domainDataset;
            }
        }

        $default = (string) config('bridge.dataset', 'stellar');
        if (in_array($default, $allowed, true)) {
            return $default;
        }

        return $allowed[0];
    }

    /**
     * @return list<string>
     */
    public function getAvailableDatasets(): array
    {
        $datasets = config('bridge.datasets', ['stellar']);

        return is_array($datasets) ? array_values(array_filter(array_map('trim', $datasets))) : ['stellar'];
    }

    /**
     * Datasets this domain may query (intersection of global catalog and optional domain allowlist).
     *
     * @return list<string>
     */
    public function datasetsEnabledForDomain(?Domain $domain): array
    {
        $global = $this->getAvailableDatasets();
        if (! $domain instanceof Domain) {
            return $global;
        }

        $restriction = $domain->getAllowedMlsDatasets();
        if ($restriction === null) {
            return $global;
        }

        return array_values(array_intersect($restriction, $global));
    }

    public function validateDataset(string $dataset): void
    {
        $this->validateDatasetForSite($dataset, $this->getAvailableDatasets());
    }

    /**
     * @param  list<string>  $enabledForSite
     */
    public function validateDatasetForSite(string $dataset, array $enabledForSite): void
    {
        $available = $this->getAvailableDatasets();
        if (! in_array($dataset, $available, true)) {
            throw new HttpException(400, "Dataset '{$dataset}' is not available. Available datasets: ".implode(', ', $available));
        }

        if (! in_array($dataset, $enabledForSite, true)) {
            throw new HttpException(403, "Dataset '{$dataset}' is not enabled for this site.");
        }
    }
}
