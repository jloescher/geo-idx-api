<?php

declare(strict_types=1);

namespace App\Services\Bridge;

final readonly class BridgeReplicaPersistStats
{
    public function __construct(
        public int $rowsReceived = 0,
        public int $upserted = 0,
        public int $deleted = 0,
        public int $skipped = 0,
        public int $durationMs = 0,
    ) {}

    /**
     * @return array<string, int>
     */
    public function toArray(): array
    {
        return [
            'rows_received' => $this->rowsReceived,
            'upserted' => $this->upserted,
            'deleted' => $this->deleted,
            'skipped' => $this->skipped,
            'duration_ms' => $this->durationMs,
        ];
    }
}
