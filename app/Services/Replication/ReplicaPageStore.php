<?php

declare(strict_types=1);

namespace App\Services\Replication;

use App\Models\ReplicaPage;
use App\Services\Mls\MlsDatasetRegistry;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;
use JsonException;

/**
 * Revenue impact: staging MLS pages in Postgres with per-chunk gzip parts keeps persist workers
 * under memory limits while replication catch-up runs at a steady ~2 req/s.
 */
final class ReplicaPageStore
{
    public function __construct(
        private readonly MlsDatasetRegistry $datasets,
    ) {}

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
        $chunkSize = max(1, $this->datasets->persistChunkSize($datasetSlug));
        $parts = $this->compressRowChunks($rows, $chunkSize);

        $id = DB::table('replica_pages')->insertGetId([
            'provider' => $provider,
            'dataset_slug' => $datasetSlug,
            'mode' => $mode,
            'status' => ReplicaPage::STATUS_PENDING,
            'compressed_payload' => $this->encodePayload($parts),
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
        return ReplicaPage::query()
            ->where('provider', $provider)
            ->where('dataset_slug', $datasetSlug)
            ->whereIn('status', [
                ReplicaPage::STATUS_PENDING,
                ReplicaPage::STATUS_PROCESSING,
            ])
            ->exists();
    }

    public function markProcessing(int $pageId, ?string $batchId = null): void
    {
        $values = ['status' => ReplicaPage::STATUS_PROCESSING];
        if ($batchId !== null) {
            $values['batch_id'] = $batchId;
        }

        ReplicaPage::query()->whereKey($pageId)->update($values);
    }

    public function markCompleted(int $pageId): void
    {
        ReplicaPage::query()->whereKey($pageId)->update([
            'status' => ReplicaPage::STATUS_COMPLETED,
            'processed_at' => now(),
        ]);
    }

    public function markFailed(int $pageId): void
    {
        ReplicaPage::query()->whereKey($pageId)->update([
            'status' => ReplicaPage::STATUS_FAILED,
            'processed_at' => now(),
        ]);
    }

    public function deletePage(int $pageId): void
    {
        ReplicaPage::query()->whereKey($pageId)->delete();
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function rowsForChunk(int $pageId, int $chunkIndex, int $chunkTotal): array
    {
        if ($chunkTotal < 1 || $chunkIndex < 1) {
            return [];
        }

        $page = DB::table('replica_pages')->where('id', $pageId)->first();
        if ($page === null || $page->compressed_payload === null) {
            return [];
        }

        $payload = $this->readPayloadString($page->compressed_payload);
        $v2 = $this->decodeV2Payload($payload);
        if ($v2 !== null) {
            $partIndex = $chunkIndex - 1;
            if (! isset($v2[$partIndex])) {
                return [];
            }

            return $this->decompressRows($v2[$partIndex]);
        }

        $rows = $this->decompressLegacyPayload($payload);
        if ($rows === []) {
            return [];
        }

        $chunkSize = (int) max(1, (int) ceil(count($rows) / $chunkTotal));
        $offset = ($chunkIndex - 1) * $chunkSize;

        return array_slice($rows, $offset, $chunkSize);
    }

    public function chunkCountForPage(int $pageId): int
    {
        $page = DB::table('replica_pages')->where('id', $pageId)->first();
        if ($page === null || $page->compressed_payload === null) {
            return 1;
        }

        $payload = $this->readPayloadString($page->compressed_payload);
        $v2 = $this->decodeV2Payload($payload);
        if ($v2 !== null) {
            return max(1, count($v2));
        }

        return 1;
    }

    public function purgeEligibleRows(): int
    {
        $completedBefore = now()->subHours((int) config('mls.replica_page_retention_hours', 24));
        $failedBefore = now()->subDays((int) config('mls.replica_page_failed_retention_days', 7));
        $abandonedBefore = now()->subHours((int) config('mls.replica_page_retention_hours', 24));

        $deleted = 0;

        $deleted += ReplicaPage::query()
            ->where('status', ReplicaPage::STATUS_COMPLETED)
            ->where('processed_at', '<', $completedBefore)
            ->delete();

        $deleted += ReplicaPage::query()
            ->where('status', ReplicaPage::STATUS_FAILED)
            ->where('processed_at', '<', $failedBefore)
            ->delete();

        $deleted += ReplicaPage::query()
            ->where('status', ReplicaPage::STATUS_PENDING)
            ->whereNull('processed_at')
            ->where('fetched_at', '<', $abandonedBefore)
            ->delete();

        if ($deleted > 0) {
            Log::info('replication.staging_purged', ['deleted_count' => $deleted]);
        }

        return $deleted;
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     * @return list<string>
     */
    private function compressRowChunks(array $rows, int $chunkSize): array
    {
        if ($rows === []) {
            return [];
        }

        $parts = [];
        foreach (array_chunk($rows, $chunkSize) as $chunk) {
            $parts[] = $this->compressRows($chunk);
        }

        return $parts;
    }

    private function encodePayload(array $gzipParts): string
    {
        if ($gzipParts === []) {
            return '';
        }

        $encodedParts = [];
        foreach ($gzipParts as $part) {
            $encodedParts[] = base64_encode($part);
        }

        return json_encode(['v' => 2, 'parts' => $encodedParts], JSON_THROW_ON_ERROR);
    }

    /**
     * @return list<string>|null
     */
    private function decodeV2Payload(string $payload): ?array
    {
        $parsed = json_decode($payload, true);
        if (! is_array($parsed) || ($parsed['v'] ?? null) !== 2) {
            return null;
        }

        $parts = $parsed['parts'] ?? null;
        if (! is_array($parts)) {
            return null;
        }

        $gzipParts = [];
        foreach ($parts as $part) {
            if (! is_string($part)) {
                continue;
            }
            $decoded = base64_decode($part, true);
            if ($decoded !== false) {
                $gzipParts[] = $decoded;
            }
        }

        return $gzipParts !== [] ? $gzipParts : null;
    }

    /**
     * @return list<array<string, mixed>>
     */
    private function decompressLegacyPayload(string $payload): array
    {
        $decoded = base64_decode($payload, true);
        if ($decoded === false) {
            return [];
        }

        return $this->decompressRows($decoded);
    }

    /**
     * @param  list<array<string, mixed>>  $rows
     */
    private function compressRows(array $rows): string
    {
        $encoded = json_encode($rows, JSON_THROW_ON_ERROR | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
        $compressed = gzencode($encoded, 9);
        if ($compressed === false) {
            throw new JsonException('Failed to gzip-compress replica page payload.');
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

    private function readPayloadString(mixed $payload): string
    {
        if (is_resource($payload)) {
            $payload = stream_get_contents($payload);
        }

        return (string) $payload;
    }
}
