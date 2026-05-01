<?php

namespace App\Services\AgentPortal;

use App\Services\AgentPortal\Contracts\LookupProviderInterface;

/**
 * Placeholder until Bridge/RESO lookups are wired; returns empty catalogs so UI can boot.
 */
final class StubLookupProvider implements LookupProviderInterface
{
    public function fetchFieldCatalog(string $mlsCode, string $datasetCode): array
    {
        return [];
    }

    public function fetchEnumValues(string $mlsCode, string $datasetCode, string $sourceFieldKey): array
    {
        return [];
    }

    public function fetchBoundaryCatalog(string $mlsCode, string $datasetCode): array
    {
        return [];
    }
}
