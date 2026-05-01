<?php

namespace App\Services\AgentPortal;

use App\Services\AgentPortal\Contracts\MultiMlsQueryCompilerInterface;

/**
 * Revenue impact: canonical filter compilation is the choke point for teaser vs full listing exposure downstream.
 */
final class DefaultMultiMlsQueryCompiler implements MultiMlsQueryCompilerInterface
{
    public function compile(array $canonicalFilters, array $mlsScope): array
    {
        $out = [];
        foreach ($mlsScope as $row) {
            $mls = is_array($row) ? (string) ($row['mls_code'] ?? '') : '';
            $dataset = is_array($row) ? (string) ($row['dataset_code'] ?? '') : '';
            if ($mls === '' || $dataset === '') {
                continue;
            }
            $key = $mls.'@'.$dataset;
            $out[$key] = [
                'mls_code' => $mls,
                'dataset_code' => $dataset,
                'filters' => $canonicalFilters,
            ];
        }

        return $out;
    }

    public function validate(array $canonicalFilters, array $mlsScope): array
    {
        $errors = [];
        if ($mlsScope === []) {
            $errors['mls_scope'][] = 'At least one MLS dataset scope is required.';
        }

        foreach ($canonicalFilters as $i => $filter) {
            if (! is_array($filter) || ! isset($filter['field'], $filter['operator'])) {
                $errors['filters'][] = "Filter index {$i} must include field and operator.";
            }
        }

        return $errors;
    }
}
