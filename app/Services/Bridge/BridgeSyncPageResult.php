<?php

declare(strict_types=1);

namespace App\Services\Bridge;

use Carbon\CarbonImmutable;

/**
 * @phpstan-type Row array<string, mixed>
 */
final readonly class BridgeSyncPageResult
{
    /**
     * @param  list<Row>  $rows
     */
    public function __construct(
        public array $rows,
        public ?string $nextReplicationUrl,
        public bool $replicationComplete,
        public bool $incrementalHasMore,
        public int $nextIncrementalSkip,
        public ?CarbonImmutable $maxBridgeTs,
        public bool $replicationStarting = false,
        public bool $forbidden = false,
        public bool $httpError = false,
    ) {}

    public static function forbidden(): self
    {
        return new self(
            rows: [],
            nextReplicationUrl: null,
            replicationComplete: true,
            incrementalHasMore: false,
            nextIncrementalSkip: 0,
            maxBridgeTs: null,
            forbidden: true,
        );
    }

    public static function httpError(): self
    {
        return new self(
            rows: [],
            nextReplicationUrl: null,
            replicationComplete: true,
            incrementalHasMore: false,
            nextIncrementalSkip: 0,
            maxBridgeTs: null,
            httpError: true,
        );
    }
}
