<?php

namespace App\Http\Responses\Search;

final readonly class SearchResult
{
    /**
     * @param  list<ListingResult>  $results
     */
    public function __construct(
        public int $totalCount,
        public array $results,
        public bool $hasMore,
        public ?int $nextSkip,
        public ?SearchStats $stats,
    ) {}

    public function toArray(): array
    {
        return [
            'total_count' => $this->totalCount,
            'results' => array_map(static fn (ListingResult $r) => $r->toArray(), $this->results),
            'has_more' => $this->hasMore,
            'next_skip' => $this->nextSkip,
            'stats' => $this->stats?->toArray(),
        ];
    }
}
