<?php

namespace App\Services\AgentPortal\Contracts;

interface MultiMlsQueryCompilerInterface
{
    /**
     * @param  array<int, array{field: string, operator: string, value: mixed}>  $canonicalFilters
     * @param  array<int, array{mls_code: string, dataset_code: string}>  $mlsScope
     * @return array<string, array<int, array<string, mixed>>>
     */
    public function compile(array $canonicalFilters, array $mlsScope): array;

    /**
     * @param  array<int, array{field: string, operator: string, value: mixed}>  $canonicalFilters
     * @param  array<int, array{mls_code: string, dataset_code: string}>  $mlsScope
     * @return array<string, list<string>>
     */
    public function validate(array $canonicalFilters, array $mlsScope): array;
}
