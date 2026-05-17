<?php

declare(strict_types=1);

namespace App\Services\Bridge;

use Carbon\CarbonImmutable;

final readonly class BridgeReplicaCursorPatch
{
    public function __construct(
        public bool $applyReplicationState = false,
        public ?string $replicationNextUrl = null,
        public ?bool $replicationInProgress = null,
        public ?CarbonImmutable $maxBridgeTs = null,
        public bool $markSyncFinished = false,
    ) {}
}
