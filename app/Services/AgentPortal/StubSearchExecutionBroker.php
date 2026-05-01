<?php

namespace App\Services\AgentPortal;

use App\Services\AgentPortal\Contracts\SearchExecutionBrokerInterface;

final class StubSearchExecutionBroker implements SearchExecutionBrokerInterface
{
    public function execute(array $compiledQueries): array
    {
        return [];
    }

    public function merge(array $mlsResults): array
    {
        return [
            'items' => [],
            'meta' => [
                'sources' => count($mlsResults),
            ],
        ];
    }
}
