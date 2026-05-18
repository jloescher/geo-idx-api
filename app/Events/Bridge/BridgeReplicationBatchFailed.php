<?php

declare(strict_types=1);

namespace App\Events\Bridge;

use Illuminate\Foundation\Events\Dispatchable;
use Illuminate\Queue\SerializesModels;

final class BridgeReplicationBatchFailed
{
    use Dispatchable, SerializesModels;

    /**
     * @param  array<string, mixed>  $odataQuery
     */
    public function __construct(
        public string $dataset,
        public string $mode,
        public string $failureType,
        public string $message,
        public ?string $batchId = null,
        public ?int $httpStatus = null,
        public ?string $bridgeUrl = null,
        public array $odataQuery = [],
    ) {}
}
