<?php

declare(strict_types=1);

namespace App\Events\Bridge;

use App\Services\Bridge\BridgeReplicaPersistStats;
use Illuminate\Foundation\Events\Dispatchable;
use Illuminate\Queue\SerializesModels;

final class BridgeReplicationPagePersisted
{
    use Dispatchable, SerializesModels;

    public function __construct(
        public string $dataset,
        public BridgeReplicaPersistStats $stats,
        public ?int $chunkIndex = null,
        public ?int $chunkTotal = null,
    ) {}
}
