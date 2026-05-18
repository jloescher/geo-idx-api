<?php

declare(strict_types=1);

namespace App\Events\Bridge;

use Illuminate\Foundation\Events\Dispatchable;
use Illuminate\Queue\SerializesModels;

final class BridgeReplicationPageFetched
{
    use Dispatchable, SerializesModels;

    /**
     * @param  array<string, int>  $statusCounts
     * @param  array<string, mixed>  $odataQuery
     */
    public function __construct(
        public string $dataset,
        public string $mode,
        public string $bridgeUrl,
        public array $odataQuery,
        public int $httpStatus,
        public int $listingsDownloaded,
        public array $statusCounts,
        public bool $replicationStarting,
        public bool $hasNextPage,
        public int $chainDepth,
    ) {}
}
