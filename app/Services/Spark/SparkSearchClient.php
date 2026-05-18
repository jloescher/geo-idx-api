<?php

declare(strict_types=1);

namespace App\Services\Spark;

/**
 * Live Spark RESO Property search for closed / non-replica queries.
 */
final readonly class SparkSearchClient
{
    public function __construct(
        private SparkHttpService $spark,
    ) {}

    /**
     * @return array{value: list<array<string, mixed>>, count: int, nextLink: ?string}
     */
    public function search(
        string $filter,
        string $orderby,
        int $top,
        int $skip,
    ): array {
        $queryParams = [
            '$top' => $top,
            '$skip' => $skip,
        ];

        if ($filter !== '') {
            $queryParams['$filter'] = $filter;
        }
        if ($orderby !== '') {
            $queryParams['$orderby'] = $orderby;
        }

        $url = $this->spark->propertyCollectionUrl();
        $response = $this->spark->serverJsonGet($url, $queryParams);

        if (! $response->successful()) {
            return [
                'value' => [],
                'count' => 0,
                'nextLink' => null,
            ];
        }

        $body = $response->json();
        $value = is_array($body['value'] ?? null) ? $body['value'] : [];
        $nextLink = is_array($body) && isset($body['@odata.nextLink']) && is_string($body['@odata.nextLink'])
            ? $body['@odata.nextLink']
            : null;

        return [
            'value' => $value,
            'count' => count($value),
            'nextLink' => $nextLink,
        ];
    }
}
