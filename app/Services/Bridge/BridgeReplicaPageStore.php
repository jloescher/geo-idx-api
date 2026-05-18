<?php

declare(strict_types=1);

namespace App\Services\Bridge;

use App\Models\BridgeReplicaPage;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;
use JsonException;

/**
 * Revenue impact: staging Bridge pages in Postgres keeps queue payloads small and lets
 * persist workers scale independently while replication catch-up runs at a steady 2 req/s.
 */
final class BridgeReplicaPageStore
{
    /**
     * @param  list<array<string, mixed>>  $rows
     */
    public function storePage(
        string $datasetSlug,
        string $mode,
        array $rows,
        ?string $bridgeUrl,
        ?array $odataQuery,
        string $provider = 'bridge',
    ): int {
        $now = now();

        $id = DB::table('bridge_replica_pages')->insertGetId([
            'provider' => $provider,
            'dataset_slug' => $datasetSlug,
            'mode' => $mode,
            'status' => BridgeReplicaPage::STATUS_PENDING,
            'compressed_payload' => base64_encode($this->compressRows($rows)),
            'row_count' => count($rows),
            'bridge_url' => $bridgeUrl,
            'upstream_url' => $bridgeUrl,
            'odata_query' => $odataQuery !== null ? json_encode($odataQuery, JSON_THROW_ON_ERROR) : null,
            'fetched_at' => $now,
            'created_at' => $now,
            'updated_at' => $now,
        ]);

        return (int) $id;
    }

    public function hasActivePage(string $datasetSlug, string $provider = 'bridge'): bool
    {
        return BridgeReplicaPage::query()
            ->where('provider', $provider)
            ->where('dataset_slug', $datasetSlug)
            ->whereIn('status', [
                BridgeReplicaPage::STATUS_PENDING,
                BridgeReplicaPage::STATUS_PROCESSING,
            ])
            ->exists();
    }

    public function markProcessing(int $pageId, ?string $batchId = null): void
    {
        $values = ['status' => BridgeReplicaPage::STATUS_PROCESSING];
        if ($batchId !== null) {
            $values['batch_id'] = $batchId;
        }

        BridgeReplicaPage::query()->whereKey($pageId)->update($values);
    }

    public function markCompleted(int $pageId): void
    {
        BridgeReplicaPage::query()->whereKey($pageId)->update([
            'status' => BridgeReplicaPage::STATUS_COMPLETED,
            'processed_at' => now(),
        ]);
    }

    public function markFailed(int $pageId): void
    {
        BridgeReplicaPage::query()->whereKey($pageId)->update([
            'status' => BridgeReplicaPage::STATUS_FAILED,
            'processed_at' => now(),
        ]);
    }

    public function deletePage(int $pageId): void
    {
        BridgeReplicaPage::query()->whereKey($pageId)->delete();
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function rowsForChunk(int $pageId, int $chunkIndex, int $chunkTotal): array
    {
        $rows = $this->loadRows($pageId);
        if ($rows === [] || $chunkTotal < 1 || $chunkIndex < 1) {
            return [];
        }

        $chunkSize = (int) max(1, (int) ceil(count($rows) / $chunkTotal));
        $offset = ($chunkIndex - 1) * $chunkSize;

        return array_slice($rows, $offset, $chunkSize);
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function loadRows(int $pageId): array
    {
        $page = DB::table('bridge_replica_pages')->where('id', $pageId)->first();
        if ($page === null || $page->compressed_payload === null) {
            return [];
        }

        $payload = $page->compressed_payload;
        if (is_resource($payload)) {
            $payload = stream_get_contents($payload);
        }

        $decoded = base64_decode((string) $payload, true);
        if ($decoded === false) {
            return [];
        }

        return $this->decompressRows($decoded);
    }

    public function purgeEligibleRows(): int
    {
        $completedBefore = now()->subHours((int) config('bridge.replica_page_retention_hours', 24));
        $failedBefore = now()->subDays((int) config('bridge.replica_page_failed_retention_days', 7));
        $abandonedBefore = now()->subHours((int) config('bridge.replica_page_retention_hours', 24));

        $deleted = 0;

        $deleted += BridgeReplicaPage::query()
            ->where('status', BridgeReplicaPage::STATUS_COMPLETED)
            ->where('processed_at', '<', $completedBefore)
            ->delete();

        $deleted += BridgeReplicaPage::query()
            ->where('status', BridgeReplicaPage::STATUS_FAILED)
            ->where('processed_at', '<', $failedBefore)
            ->delete();

        $deleted += BridgeReplicaPage::query()
            ->where('status', BridgeReplicaPage::STATUS_PENDING)
            ->whereNull('processed_at')
            ->where('fetched_at', '<', $abandonedBefore)
            ->delete();

        if ($deleted > 0) {
            Log::info('bridge.replication.staging_purged', ['deleted_count' => $deleted]);
        }

        return $deleted;
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     */
    private function compressRows(array $rows): string
    {
        $encoded = json_encode($rows, JSON_THROW_ON_ERROR | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $compressed = gzencode($encoded, 9);
        if ($compressed === false) {
            throw new JsonException('Failed to gzip-compress Bridge replica page payload.');
        }

        return $compressed;
    }

    /**
     * @return list<array<string, mixed>>
     */
    private function decompressRows(string $compressed): array
    {
        $decoded = @gzdecode($compressed);
        if (! is_string($decoded) || $decoded === '') {
            return [];
        }

        $parsed = json_decode($decoded, true);
        if (! is_array($parsed)) {
            return [];
        }

        $rows = [];
        foreach ($parsed as $row) {
            if (is_array($row)) {
                $rows[] = $row;
            }
        }

        return $rows;
    }
}
