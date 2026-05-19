<?php

declare(strict_types=1);

namespace App\Services\Replication;

use Carbon\CarbonImmutable;

/**
 * @phpstan-type Row array<string, mixed>
 */
final readonly class ReplicationPageResult
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
        public ?string $bridgeUrl = null,
        /** @var array<string, mixed> */
        public array $odataQuery = [],
        public int $httpStatus = 200,
    ) {}

    public static function forbidden(
        ?string $bridgeUrl = null,
        array $odataQuery = [],
        int $httpStatus = 403,
    ): self {
        return new self(
            rows: [],
            nextReplicationUrl: null,
            replicationComplete: true,
            incrementalHasMore: false,
            nextIncrementalSkip: 0,
            maxBridgeTs: null,
            forbidden: true,
            bridgeUrl: $bridgeUrl,
            odataQuery: $odataQuery,
            httpStatus: $httpStatus,
        );
    }

    public static function httpError(
        ?string $bridgeUrl = null,
        array $odataQuery = [],
        int $httpStatus = 0,
    ): self {
        return new self(
            rows: [],
            nextReplicationUrl: null,
            replicationComplete: true,
            incrementalHasMore: false,
            nextIncrementalSkip: 0,
            maxBridgeTs: null,
            httpError: true,
            bridgeUrl: $bridgeUrl,
            odataQuery: $odataQuery,
            httpStatus: $httpStatus,
        );
    }
}
