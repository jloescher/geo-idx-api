<?php

declare(strict_types=1);

namespace App\Services\Mls;

/**
 * Revenue impact: one registry drives replication kickoff, dashboard feeds, and per-provider queues.
 */
final class MlsDatasetRegistry
{
    /**
     * @return array<string, array<string, mixed>>
     */
    public function datasets(): array
    {
        $configured = config('mls.datasets', []);
        if (! is_array($configured)) {
            return [];
        }

        $out = [];
        foreach ($configured as $slug => $def) {
            if (! is_string($slug) || $slug === '' || ! is_array($def)) {
                continue;
            }
            if (($def['enabled'] ?? true) === false) {
                continue;
            }
            $out[$slug] = $def;
        }

        return $out;
    }

    /**
     * @return list<string>
     */
    public function slugsForProvider(string $provider): array
    {
        $slugs = [];
        foreach ($this->datasets() as $slug => $def) {
            if (($def['provider'] ?? '') === $provider) {
                $slugs[] = $slug;
            }
        }

        return $slugs;
    }

    /**
     * @return array<string, mixed>|null
     */
    public function definition(string $slug): ?array
    {
        return $this->datasets()[$slug] ?? null;
    }

    public function persistChunkSize(string $slug): int
    {
        $def = $this->definition($slug);

        return max(1, (int) ($def['persist_chunk_size'] ?? 50));
    }

    public function provider(string $slug): string
    {
        $def = $this->definition($slug);

        return is_array($def) ? (string) ($def['provider'] ?? 'bridge') : 'bridge';
    }

    public function persistSequential(string $slug): bool
    {
        $def = $this->definition($slug);

        return is_array($def) && ($def['persist_sequential'] ?? false) === true;
    }
}
