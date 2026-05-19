<?php

declare(strict_types=1);

namespace App\Services\Replication;

use Carbon\CarbonImmutable;

final readonly class ReplicationCursorPatch
{
    public function __construct(
        public bool $applyReplicationState = false,
        public ?string $replicationNextUrl = null,
        public ?bool $replicationInProgress = null,
        public ?CarbonImmutable $maxBridgeTs = null,
        public ?CarbonImmutable $incrementalWindowEnd = null,
        public bool $markSyncFinished = false,
    ) {}
}
