<?php

namespace App\Services\AgentPortal;

final class AgentSearchGeometrySafeguardService
{
    public const MAX_GEOMETRIES_PER_QUERY = 6;

    public const MAX_POLYGON_VERTICES = 200;

    /**
     * @param  list<array<string, mixed>>  $geometries
     * @return array<string, list<string>>
     */
    public function validate(array $geometries): array
    {
        $errors = [];
        if (count($geometries) > self::MAX_GEOMETRIES_PER_QUERY) {
            $errors['geometries'][] = sprintf(
                'Too many geometries. Maximum allowed is %d.',
                self::MAX_GEOMETRIES_PER_QUERY
            );
        }

        $maxPolygonSpan = (float) config('agent_search.max_polygon_bbox_span_deg', 0.85);
        $maxCircleRadius = (float) config('agent_search.max_circle_radius_m', 75000);

        foreach ($geometries as $index => $geometry) {
            if (! is_array($geometry)) {
                continue;
            }
            $type = (string) ($geometry['geometry_type'] ?? '');
            $geojson = is_array($geometry['geojson'] ?? null) ? $geometry['geojson'] : [];

            if ($type === 'circle') {
                $radius = isset($geojson['radius_m']) && is_numeric($geojson['radius_m']) ? (float) $geojson['radius_m'] : null;
                if ($radius !== null && $radius > $maxCircleRadius) {
                    $errors['geometries'][] = sprintf(
                        'Geometry #%d circle radius exceeds maximum (%.0f m).',
                        $index + 1,
                        $maxCircleRadius
                    );
                }

                continue;
            }

            if ($type !== 'polygon') {
                continue;
            }
            $coordinates = $geojson['coordinates'] ?? null;
            if (! is_array($coordinates) || $coordinates === []) {
                continue;
            }
            $ring = $coordinates[0] ?? null;
            $vertexCount = is_array($ring) ? count($ring) : 0;
            if ($vertexCount > self::MAX_POLYGON_VERTICES) {
                $errors['geometries'][] = sprintf(
                    'Geometry #%d exceeds max polygon vertices (%d).',
                    $index + 1,
                    self::MAX_POLYGON_VERTICES
                );
            }
            if (is_array($ring) && $vertexCount >= 3) {
                $span = $this->polygonRingBoundingBoxMaxSpanDegrees($ring);
                if ($span > $maxPolygonSpan) {
                    $errors['geometries'][] = sprintf(
                        'Geometry #%d bounding box is too large (max span %.2f°; yours %.2f°).',
                        $index + 1,
                        $maxPolygonSpan,
                        $span
                    );
                }
            }
        }

        return $errors;
    }

    /**
     * @param  list<mixed>  $ring
     */
    private function polygonRingBoundingBoxMaxSpanDegrees(array $ring): float
    {
        $minLat = 90.0;
        $maxLat = -90.0;
        $minLng = 180.0;
        $maxLng = -180.0;
        $has = false;

        foreach ($ring as $point) {
            if (! is_array($point) || count($point) < 2) {
                continue;
            }
            $lng = (float) $point[0];
            $lat = (float) $point[1];
            $has = true;
            $minLat = min($minLat, $lat);
            $maxLat = max($maxLat, $lat);
            $minLng = min($minLng, $lng);
            $maxLng = max($maxLng, $lng);
        }

        if (! $has || $minLat > $maxLat || $minLng > $maxLng) {
            return 0.0;
        }

        return max($maxLat - $minLat, $maxLng - $minLng);
    }
}
