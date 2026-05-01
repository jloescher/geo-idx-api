<?php

namespace App\Services\AgentPortal\Support;

final class CacheKeyFactory
{
    public function lookups(string $mlsCode, string $datasetCode, string $scope, string $version): string
    {
        return sprintf(
            'idx:lookups:%s:%s:%s:%s',
            trim($mlsCode),
            trim($datasetCode),
            trim($scope),
            trim($version),
        );
    }

    public function queryPlan(int $userId, string $searchHash, string $mlsCode, string $datasetCode): string
    {
        return sprintf(
            'idx:query-plan:%d:%s:%s:%s',
            $userId,
            trim($searchHash),
            trim($mlsCode),
            trim($datasetCode),
        );
    }
}
