<?php

namespace App\Services\AgentPortal\Contracts;

interface LookupProviderInterface
{
    /**
     * @return array<int, array<string, mixed>>
     */
    public function fetchFieldCatalog(string $mlsCode, string $datasetCode): array;

    /**
     * @return array<int, string|array<string, mixed>>
     */
    public function fetchEnumValues(string $mlsCode, string $datasetCode, string $sourceFieldKey): array;

    /**
     * @return array<int, array<string, mixed>>
     */
    public function fetchBoundaryCatalog(string $mlsCode, string $datasetCode): array;
}
