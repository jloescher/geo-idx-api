<?php

declare(strict_types=1);

namespace App\Services\Replication;

use App\Models\Listing;
use App\Models\ListingSyncCursor;
use Carbon\CarbonImmutable;

/**
 * Revenue impact: shared cursor and persist hooks keep Bridge and Spark mirrors aligned on
 * Active/Pending scope without duplicating batch upsert logic.
 */
abstract class MlsReplicationService
{
    public function cursorForDataset(string $dataset): ListingSyncCursor
    {
        /** @var ListingSyncCursor $cursor */
        $cursor = ListingSyncCursor::query()->firstOrCreate(
            ['dataset_slug' => $dataset],
            ['replication_in_progress' => false],
        );

        return $cursor;
    }

    public function shouldRunReplication(ListingSyncCursor $cursor): bool
    {
        if ($cursor->replication_next_url !== null && $cursor->replication_next_url !== '') {
            return true;
        }

        if ($cursor->replication_in_progress) {
            return true;
        }

        return ! Listing::query()->where('dataset_slug', $cursor->dataset_slug)->exists();
    }

    public function shouldRunIncremental(ListingSyncCursor $cursor): bool
    {
        if ($cursor->replication_in_progress) {
            return false;
        }

        return $cursor->last_bridge_modification_timestamp !== null;
    }

    protected function advanceCursorTimestamp(ListingSyncCursor $cursor, ?CarbonImmutable $maxBridgeTs): void
    {
        if ($maxBridgeTs === null) {
            return;
        }
        $existing = $cursor->last_bridge_modification_timestamp;
        if (! $existing instanceof \DateTimeInterface || CarbonImmutable::parse($existing->format(\DateTimeInterface::ATOM))->lt($maxBridgeTs)) {
            $cursor->last_bridge_modification_timestamp = $maxBridgeTs;
        }
    }

    public function applyCursorPatch(string $dataset, ReplicationCursorPatch $patch): void
    {
        $cursor = $this->cursorForDataset($dataset);

        if ($patch->applyReplicationState) {
            $cursor->replication_next_url = $patch->replicationNextUrl;
            if ($patch->replicationInProgress !== null) {
                $cursor->replication_in_progress = $patch->replicationInProgress;
            }
        }

        if ($patch->incrementalWindowEnd !== null) {
            $cursor->incremental_window_end = $patch->incrementalWindowEnd;
        }

        if ($patch->maxBridgeTs !== null) {
            $this->advanceCursorTimestamp($cursor, $patch->maxBridgeTs);
        }

        if ($patch->markSyncFinished) {
            $cursor->last_sync_finished_at = now();
            $this->onSyncFinished($cursor, $patch);
        }

        $cursor->save();
    }

    protected function onSyncFinished(ListingSyncCursor $cursor, ReplicationCursorPatch $patch): void
    {
        // Provider-specific hooks (Spark advances window timestamp).
    }
}
