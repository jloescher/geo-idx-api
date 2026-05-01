<?php

namespace App\Services\AgentPortal\Contracts;

interface SearchExecutionBrokerInterface
{
    /**
     * @param  array<string, mixed>  $compiledQueries
     * @return array<int, array<string, mixed>>
     */
    public function execute(array $compiledQueries): array;

    /**
     * @param  array<int, array<string, mixed>>  $mlsResults
     * @return array<string, mixed>
     */
    public function merge(array $mlsResults): array;
}
