<?php

namespace App\Services;

use App\Models\GisSourceState;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class GisSourceMetadataService
{
    /**
     * Revenue impact: one lightweight GET per source beats re-querying every stored
     * bbox when FGIO or a county republishes, protecting gross margin on map traffic.
     */
    public function probeAllSources(): void
    {
        foreach (['primary', 'pinellas', 'hillsborough'] as $tier) {
            $source = (array) config('gis.sources.'.$tier);
            $key = (string) ($source['key'] ?? '');
            if ($key === '') {
                continue;
            }
            $queryUrl = (string) ($source['query_url'] ?? '');
            if ($queryUrl === '') {
                continue;
            }
            $this->probeSource($key, $queryUrl);
        }
    }

    public function probeSource(string $sourceKey, string $queryUrl): void
    {
        $metadataUrl = self::metadataUrlFromQueryUrl($queryUrl);
        $fingerprint = $this->fetchFingerprint($metadataUrl);

        /** @var GisSourceState $state */
        $state = GisSourceState::query()->firstOrCreate(
            ['source_key' => $sourceKey],
            ['fingerprint' => null, 'generation' => 0]
        );

        $state->last_checked_at = now();

        if ($fingerprint === null) {
            $state->save();

            return;
        }

        $previous = $state->fingerprint;
        if ($previous !== null && $previous !== $fingerprint) {
            $state->generation = (int) $state->generation + 1;
            $state->last_changed_at = now();
            Log::info('gis.source_generation_bumped', [
                'source_key' => $sourceKey,
                'generation' => $state->generation,
            ]);
        }

        $state->fingerprint = $fingerprint;
        $state->save();
    }

    public static function metadataUrlFromQueryUrl(string $queryUrl): string
    {
        if (str_ends_with($queryUrl, '/query')) {
            return substr($queryUrl, 0, -strlen('/query')).'?f=json';
        }

        return $queryUrl;
    }

    private function fetchFingerprint(string $metadataUrl): ?string
    {
        try {
            $response = Http::connectTimeout((int) config('gis.metadata_connect_timeout_seconds'))
                ->timeout((int) config('gis.metadata_timeout_seconds'))
                ->acceptJson()
                ->get($metadataUrl);
        } catch (\Throwable) {
            return null;
        }

        if (! $response->successful()) {
            return null;
        }

        try {
            /** @var array<string, mixed> $json */
            $json = $response->json();
        } catch (\Throwable) {
            return null;
        }

        return $this->fingerprintFromLayerJson($json);
    }

    /**
     * @param  array<string, mixed>  $json
     */
    private function fingerprintFromLayerJson(array $json): string
    {
        $editing = $json['editingInfo'] ?? [];
        if (! is_array($editing)) {
            $editing = [];
        }

        $parts = [
            (string) ($json['currentVersion'] ?? ''),
            json_encode($editing, JSON_THROW_ON_ERROR),
            (string) ($json['serviceItemId'] ?? ''),
            (string) ($json['name'] ?? ''),
            (string) ($json['cacheMaxAge'] ?? ''),
        ];

        return hash('sha256', implode('|', $parts));
    }
}
