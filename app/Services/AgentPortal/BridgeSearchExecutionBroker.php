<?php

namespace App\Services\AgentPortal;

use App\Services\AgentPortal\Contracts\SearchExecutionBrokerInterface;
use App\Services\Bridge\BridgeSearchClient;

final class BridgeSearchExecutionBroker implements SearchExecutionBrokerInterface
{
    public function __construct(
        private readonly BridgeSearchClient $searchClient,
    ) {}

    public function execute(array $compiledQueries): array
    {
        $rows = [];
        foreach ($compiledQueries as $query) {
            if (! is_array($query)) {
                continue;
            }
            $dataset = (string) ($query['dataset_code'] ?? '');
            if ($dataset === '') {
                continue;
            }

            $filter = $this->toBridgeFilter((array) ($query['filters'] ?? []));
            try {
                $bridge = $this->searchClient->search(
                    dataset: $dataset,
                    filter: $filter,
                    orderby: 'OnMarketDate desc',
                    top: 25,
                    skip: 0,
                    select: '*',
                    unselect: '',
                );
            } catch (\Throwable) {
                $bridge = ['value' => []];
            }

            $mapped = array_map(
                fn (array $record): array => $this->searchClient->mapToListingResult($record, $dataset)->toArray(),
                (array) ($bridge['value'] ?? []),
            );
            $mapped = $this->applyGeometries($mapped, (array) ($query['geometries'] ?? []));

            $rows[] = [
                'dataset' => $dataset,
                'mls_code' => (string) ($query['mls_code'] ?? ''),
                'items' => $mapped,
                'count' => count($mapped),
            ];
        }

        return $rows;
    }

    public function merge(array $mlsResults): array
    {
        $items = [];
        $bySource = [];
        foreach ($mlsResults as $result) {
            if (! is_array($result)) {
                continue;
            }
            $sourceKey = trim((string) ($result['mls_code'] ?? '')).':'.trim((string) ($result['dataset'] ?? ''));
            if ($sourceKey !== ':') {
                $bySource[$sourceKey] = [
                    'mls_code' => (string) ($result['mls_code'] ?? ''),
                    'dataset_code' => (string) ($result['dataset'] ?? ''),
                    'item_count' => (int) ($result['count'] ?? 0),
                ];
            }
            foreach ((array) ($result['items'] ?? []) as $item) {
                if (! is_array($item)) {
                    continue;
                }
                $item['source_mls'] = (string) ($result['mls_code'] ?? '');
                $item['source_dataset'] = (string) ($result['dataset'] ?? '');
                $items[] = $item;
            }
        }

        return [
            'items' => $items,
            'meta' => [
                'sources' => count($mlsResults),
                'total_items' => count($items),
                'by_source' => array_values($bySource),
            ],
        ];
    }

    /**
     * @param  array<int, array<string, mixed>>  $filters
     */
    private function toBridgeFilter(array $filters): string
    {
        $parts = [];
        foreach ($filters as $filter) {
            if (! is_array($filter)) {
                continue;
            }
            $field = (string) ($filter['field'] ?? '');
            $op = (string) ($filter['operator'] ?? '');
            $value = $filter['value'] ?? null;
            if ($field === '' || $op === '') {
                continue;
            }

            if ($field === 'location.viewport' && $op === 'bbox') {
                $bboxStr = is_string($value) ? $value : '';
                $coords = array_map('floatval', explode(',', $bboxStr));
                if (count($coords) === 4) {
                    [$s, $w, $n, $e] = $coords;
                    $parts[] = "Latitude ge {$s}";
                    $parts[] = "Latitude le {$n}";
                    $parts[] = "Longitude ge {$w}";
                    $parts[] = "Longitude le {$e}";
                }

                continue;
            }

            $mapped = match ($field) {
                'property.list_price' => 'ListPrice',
                'property.bedrooms_total' => 'BedroomsTotal',
                'property.bathrooms_total' => 'BathroomsTotalDecimal',
                'location.city' => 'City',
                'listing.status' => 'StandardStatus',
                default => null,
            };
            if ($mapped === null) {
                continue;
            }

            if (in_array($op, ['eq', 'gte', 'lte'], true)) {
                $bridgeOp = match ($op) {
                    'eq' => 'eq',
                    'gte' => 'ge',
                    'lte' => 'le',
                };
                if (is_numeric($value)) {
                    $parts[] = "{$mapped} {$bridgeOp} {$value}";
                } elseif (is_string($value) && $value !== '') {
                    $parts[] = "tolower({$mapped}) {$bridgeOp} '".strtolower($value)."'";
                }
            } elseif ($op === 'contains' && is_string($value) && $value !== '') {
                $parts[] = "substringof('".strtolower($value)."', tolower({$mapped})) eq true";
            }
        }

        if ($parts === []) {
            return "tolower(StandardStatus) eq 'active'";
        }

        return implode(' and ', $parts);
    }

    /**
     * @param  list<array<string, mixed>>  $items
     * @param  list<array<string, mixed>>  $geometries
     * @return list<array<string, mixed>>
     */
    private function applyGeometries(array $items, array $geometries): array
    {
        if ($geometries === []) {
            return $items;
        }

        $include = array_values(array_filter($geometries, fn (array $g): bool => (($g['mode'] ?? '') === 'include')));
        $exclude = array_values(array_filter($geometries, fn (array $g): bool => (($g['mode'] ?? '') === 'exclude')));

        $filtered = [];
        foreach ($items as $item) {
            $lat = isset($item['latitude']) && is_numeric($item['latitude']) ? (float) $item['latitude'] : null;
            $lng = isset($item['longitude']) && is_numeric($item['longitude']) ? (float) $item['longitude'] : null;
            if ($lat === null || $lng === null) {
                continue;
            }

            $included = $include === [];
            foreach ($include as $geometry) {
                if ($this->containsPoint($geometry, $lat, $lng)) {
                    $included = true;
                    break;
                }
            }
            if (! $included) {
                continue;
            }

            $isExcluded = false;
            foreach ($exclude as $geometry) {
                if ($this->containsPoint($geometry, $lat, $lng)) {
                    $isExcluded = true;
                    break;
                }
            }

            if (! $isExcluded) {
                $filtered[] = $item;
            }
        }

        return $filtered;
    }

    /**
     * @param  array<string, mixed>  $geometry
     */
    private function containsPoint(array $geometry, float $lat, float $lng): bool
    {
        $type = (string) ($geometry['geometry_type'] ?? '');
        $geojson = is_array($geometry['geojson'] ?? null) ? $geometry['geojson'] : [];

        if ($type === 'circle') {
            $center = is_array($geojson['center'] ?? null) ? $geojson['center'] : [];
            $radius = isset($geojson['radius_m']) && is_numeric($geojson['radius_m']) ? (float) $geojson['radius_m'] : null;
            $clat = isset($center['lat']) && is_numeric($center['lat']) ? (float) $center['lat'] : null;
            $clng = isset($center['lng']) && is_numeric($center['lng']) ? (float) $center['lng'] : null;
            if ($radius === null || $clat === null || $clng === null) {
                return false;
            }

            return $this->distanceMeters($lat, $lng, $clat, $clng) <= $radius;
        }

        $coordinates = $geojson['coordinates'] ?? null;
        if (! is_array($coordinates) || $coordinates === []) {
            return false;
        }
        $ring = $coordinates[0] ?? null;
        if (! is_array($ring) || count($ring) < 3) {
            return false;
        }

        $inside = false;
        $j = count($ring) - 1;
        for ($i = 0; $i < count($ring); $i++) {
            $pi = $ring[$i];
            $pj = $ring[$j];
            if (! is_array($pi) || ! is_array($pj) || count($pi) < 2 || count($pj) < 2) {
                $j = $i;

                continue;
            }
            $xi = (float) $pi[0];
            $yi = (float) $pi[1];
            $xj = (float) $pj[0];
            $yj = (float) $pj[1];

            $intersects = (($yi > $lat) !== ($yj > $lat))
                && ($lng < ($xj - $xi) * ($lat - $yi) / (($yj - $yi) ?: 0.0000001) + $xi);
            if ($intersects) {
                $inside = ! $inside;
            }
            $j = $i;
        }

        return $inside;
    }

    private function distanceMeters(float $lat1, float $lng1, float $lat2, float $lng2): float
    {
        $earth = 6371000.0;
        $dLat = deg2rad($lat2 - $lat1);
        $dLng = deg2rad($lng2 - $lng1);
        $a = sin($dLat / 2) * sin($dLat / 2)
            + cos(deg2rad($lat1)) * cos(deg2rad($lat2)) * sin($dLng / 2) * sin($dLng / 2);
        $c = 2 * atan2(sqrt($a), sqrt(1 - $a));

        return $earth * $c;
    }
}
