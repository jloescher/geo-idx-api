<?php

declare(strict_types=1);

namespace App\Support;

/**
 * Revenue impact: misnamed queue env (legacy bridge-sync) stranded kickoff jobs off worker WORKER_QUEUES.
 */
final class BridgeSyncQueueNames
{
    /** Pre-pipeline monolithic queue; workers listen on bridge-sync-fetch / bridge-sync-persist. */
    public const LEGACY_MONOLITHIC = 'bridge-sync';

    public const DEFAULT_FETCH = 'bridge-sync-fetch';

    public const DEFAULT_PERSIST = 'bridge-sync-persist';

    public static function fetchQueue(): string
    {
        $explicit = env('BRIDGE_SYNC_FETCH_QUEUE');
        if (is_string($explicit) && trim($explicit) !== '') {
            return trim($explicit);
        }

        $legacy = env('BRIDGE_SYNC_QUEUE');
        if (is_string($legacy) && trim($legacy) !== '') {
            $legacy = trim($legacy);

            return $legacy === self::LEGACY_MONOLITHIC ? self::DEFAULT_FETCH : $legacy;
        }

        return self::DEFAULT_FETCH;
    }

    public static function persistQueue(): string
    {
        $explicit = env('BRIDGE_SYNC_PERSIST_QUEUE');
        if (is_string($explicit) && trim($explicit) !== '') {
            return trim($explicit);
        }

        return self::DEFAULT_PERSIST;
    }

    /**
     * @return list<string>
     */
    public static function legacyFetchQueueNames(): array
    {
        return [self::LEGACY_MONOLITHIC];
    }
}
