<?php

namespace App\Services\Bridge;

use App\Models\Domain;
use App\Services\Mls\MlsFeedResolver;
use Illuminate\Http\Request;
use Symfony\Component\HttpKernel\Exception\HttpException;

final class MlsDatasetResolver
{
    public function __construct(
        private readonly MlsFeedResolver $feeds,
    ) {}

    /**
     * Resolve the MLS dataset for the current request.
     *
     * Priority: query param > authenticated domain default > global config default.
     * Dataset must appear in {@see getAvailableDatasets()} and in the domain's enabled set
     * when {@see Domain::getAllowedMlsDatasets()} restricts feeds.
     */
    public function resolveDataset(Request $request): string
    {
        return $this->feeds->resolveFeedCode($request);
    }

    /**
     * @return list<string>
     */
    public function getAvailableDatasets(): array
    {
        return $this->feeds->catalogFeedCodes();
    }

    /**
     * Datasets this domain may query (intersection of global catalog and optional domain allowlist).
     *
     * @return list<string>
     */
    public function datasetsEnabledForDomain(?Domain $domain): array
    {
        $catalog = $this->getAvailableDatasets();
        if (! $domain instanceof Domain) {
            return $catalog;
        }

        $restriction = $domain->getAllowedMlsDatasets();
        if ($restriction === null) {
            return $catalog;
        }

        $normalized = [];
        foreach ($restriction as $item) {
            if (! is_string($item) || trim($item) === '') {
                continue;
            }
            try {
                $normalized[] = $this->feeds->normalizeWireDatasetToCatalogKey(trim($item));
            } catch (HttpException) {
                continue;
            }
        }

        return array_values(array_intersect($normalized, $catalog));
    }

    public function validateDataset(string $dataset): void
    {
        $this->feeds->validateFeedCode($dataset);
    }

    /**
     * @param  list<string>  $enabledForSite
     */
    public function validateDatasetForSite(string $dataset, array $enabledForSite): void
    {
        $this->feeds->validateFeedForSite($dataset, $enabledForSite);
    }
}
