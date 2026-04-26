<?php

namespace App\Http\Responses\Search;

final readonly class SearchStats
{
    public function __construct(
        public int $resultCount,
        public ?float $avgDom,
        public ?float $avgPrice,
        public ?float $medianPrice,
    ) {}

    public function toArray(): array
    {
        return [
            'result_count' => $this->resultCount,
            'avg_dom' => $this->avgDom,
            'avg_price' => $this->avgPrice,
            'median_price' => $this->medianPrice,
        ];
    }
}
