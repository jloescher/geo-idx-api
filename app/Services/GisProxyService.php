<?php

namespace App\Services;

use App\Jobs\PersistGisGeoJsonBackupJob;
use App\Models\GisCache;
use App\Models\GisSourceState;
use Carbon\Carbon;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Storage;
use Illuminate\Support\Str;

class GisProxyService
{
    /**
     * Revenue impact: one normalized payload shape lets geo-web chain `/listings`
     * then `/gis` without bespoke parsers — faster shipped maps, more contact taps.
     *
     * @return array<string, mixed>
     */
    public function fetch(Request $request, ?string $mlsCode = null): array
    {
        $fullAccess = true;
        $layers = (string) $request->query('layers', 'parcels');

        $envelope = $this->resolveEnvelope($request);
        if ($envelope === null) {
            abort(422, 'Provide bbox as west,south,east,north, or lat/lng with radius, or north/south/east/west.');
        }

        $this->assertReasonableBbox($envelope);

        $countyHint = $this->detectCountyHint($envelope);
        $limit = $this->resolveLimit($request, $fullAccess);
        $hash = $this->queryHash($envelope, $limit, $fullAccess, $mlsCode, $layers);

        $cacheKey = 'gis.response.'.$hash;
        $edgeTtlSeconds = (int) config('gis.edge_cache_ttl_seconds');

        /** @var array<string, mixed>|null $warm */
        $warm = Cache::get($cacheKey);
        if (is_array($warm) && $this->isEdgePayloadValid($warm)) {
            $warm['meta']['cached'] = true;
            $warm['meta']['cache_hit'] = 'laravel_cache';

            return $warm;
        }

        /** @var GisCache|null $row */
        $row = GisCache::query()->where('query_hash', $hash)->first();
        if ($row instanceof GisCache && $this->isPostgresRowValid($row)) {
            $decoded = json_decode((string) $row->geojson, true);
            if (is_array($decoded)) {
                $this->mergeOriginMetaIntoPayload($decoded, $row);
                $decoded['meta']['cached'] = true;
                $decoded['meta']['cache_hit'] = 'postgres';
                $decoded['meta']['expires_at'] = $row->expires_at->toIso8601String();
                Cache::put($cacheKey, $decoded, now()->addSeconds($edgeTtlSeconds));

                return $decoded;
            }
        }

        $payload = $this->materializeFromSources($envelope, $limit, $fullAccess, $countyHint, $mlsCode, $layers);
        $sourceKey = (string) ($payload['meta']['source_used'] ?? 'degraded_osm_hint');
        $generation = $this->currentGeneration($sourceKey);
        $expiresAt = now()->addDays($this->originMaxAgeDaysFor($sourceKey));

        $payload['meta']['cache_generation'] = $generation;
        $payload['meta']['blob_valid_until'] = $expiresAt->toIso8601String();
        $payload['meta']['expires_at'] = $expiresAt->toIso8601String();

        $encoded = json_encode($payload, JSON_THROW_ON_ERROR);

        GisCache::query()->updateOrCreate(
            ['query_hash' => $hash],
            [
                'geojson' => $encoded,
                'county' => $countyHint,
                'expires_at' => $expiresAt,
                'source_used' => $sourceKey,
                'source_generation' => $generation,
            ]
        );

        Cache::put($cacheKey, $payload, now()->addSeconds($edgeTtlSeconds));
        $this->persistFilesystemBackup($hash, $encoded);

        return $payload;
    }

    /**
     * Revenue impact: edge hits avoid Postgres round trips during dense map drags,
     * keeping p95 latency inside conversion-sensitive UX windows.
     *
     * @param  array<string, mixed>  $payload
     */
    private function isEdgePayloadValid(array $payload): bool
    {
        $meta = $payload['meta'] ?? null;
        if (! is_array($meta)) {
            return false;
        }
        $key = (string) ($meta['source_used'] ?? '');
        if ($key === '') {
            return false;
        }
        $untilRaw = $meta['blob_valid_until'] ?? null;
        if (! is_string($untilRaw)) {
            return false;
        }
        try {
            $until = Carbon::parse($untilRaw);
        } catch (\Throwable) {
            return false;
        }
        if ($until->isPast()) {
            return false;
        }
        if ($key === 'degraded_osm_hint') {
            return true;
        }
        $gen = (int) ($meta['cache_generation'] ?? -1);
        if ($gen < 0) {
            return false;
        }

        return $gen === $this->currentGeneration($key);
    }

    private function isPostgresRowValid(GisCache $row): bool
    {
        if (! $row->expires_at->isFuture()) {
            return false;
        }
        $key = (string) $row->source_used;
        if ($key === 'degraded_osm_hint') {
            return true;
        }

        return (int) $row->source_generation === $this->currentGeneration($key);
    }

    /**
     * @param  array<string, mixed>  $decoded
     */
    private function mergeOriginMetaIntoPayload(array &$decoded, GisCache $row): void
    {
        if (! isset($decoded['meta']) || ! is_array($decoded['meta'])) {
            $decoded['meta'] = [];
        }
        $decoded['meta']['cache_generation'] = (int) $row->source_generation;
        $decoded['meta']['blob_valid_until'] = $row->expires_at->toIso8601String();
    }

    private function currentGeneration(string $sourceKey): int
    {
        if ($sourceKey === 'degraded_osm_hint') {
            return 0;
        }

        return (int) GisSourceState::query()->where('source_key', $sourceKey)->value('generation');
    }

    private function originMaxAgeDaysFor(string $sourceKey): int
    {
        /** @var array<string, int> $map */
        $map = (array) config('gis.origin_max_age_days_by_source');
        if (isset($map[$sourceKey])) {
            return max(1, (int) $map[$sourceKey]);
        }

        return max(1, (int) config('gis.default_origin_max_age_days'));
    }

    /**
     * @param  array{minLon: float, minLat: float, maxLon: float, maxLat: float}  $envelope
     */
    private function assertReasonableBbox(array $envelope): void
    {
        $maxSpan = (float) config('gis.max_bbox_span_degrees');
        $lonSpan = abs($envelope['maxLon'] - $envelope['minLon']);
        $latSpan = abs($envelope['maxLat'] - $envelope['minLat']);
        if ($lonSpan > $maxSpan || $latSpan > $maxSpan) {
            abort(422, 'Requested bbox exceeds maximum span; zoom in or reduce radius.');
        }
    }

    /**
     * @return array{minLon: float, minLat: float, maxLon: float, maxLat: float}|null
     */
    private function resolveEnvelope(Request $request): ?array
    {
        if ($request->filled('bbox')) {
            $parts = array_map('floatval', explode(',', (string) $request->query('bbox')));

            return $this->orderedEnvelope($parts[0], $parts[1], $parts[2], $parts[3]);
        }

        if ($request->filled(['north', 'south', 'east', 'west'])) {
            $n = (float) $request->query('north');
            $s = (float) $request->query('south');
            $e = (float) $request->query('east');
            $w = (float) $request->query('west');

            return $this->orderedEnvelope($w, $s, $e, $n);
        }

        $lat = $request->query('lat', $request->query('latitude'));
        $lng = $request->query('lng', $request->query('lon', $request->query('longitude')));
        if ($lat !== null && $lng !== null && $request->filled('radius')) {
            return $this->envelopeFromRadius((float) $lat, (float) $lng, (float) $request->query('radius'));
        }

        return null;
    }

    /**
     * @return array{minLon: float, minLat: float, maxLon: float, maxLat: float}
     */
    private function orderedEnvelope(float $a, float $b, float $c, float $d): array
    {
        // Accept either west,south,east,north ordering (geo-web / Bridge style) or mixed; normalize.
        $lons = [$a, $c];
        $lats = [$b, $d];

        return [
            'minLon' => min($lons),
            'minLat' => min($lats),
            'maxLon' => max($lons),
            'maxLat' => max($lats),
        ];
    }

    /**
     * @return array{minLon: float, minLat: float, maxLon: float, maxLat: float}
     */
    private function envelopeFromRadius(float $lat, float $lng, float $radiusMeters): array
    {
        $latDelta = $radiusMeters / 111_320;
        $cos = cos(deg2rad($lat));
        $cos = abs($cos) < 0.00001 ? 0.00001 : $cos;
        $lngDelta = $radiusMeters / (111_320 * $cos);

        return [
            'minLon' => $lng - $lngDelta,
            'minLat' => $lat - $latDelta,
            'maxLon' => $lng + $lngDelta,
            'maxLat' => $lat + $latDelta,
        ];
    }

    /**
     * @param  array{minLon: float, minLat: float, maxLon: float, maxLat: float}  $envelope
     */
    private function detectCountyHint(array $envelope): ?string
    {
        $pinellas = (array) config('gis.sources.pinellas.bbox_wgs84');
        $hills = (array) config('gis.sources.hillsborough.bbox_wgs84');
        $hitsPinellas = $this->bboxIntersectsEnvelope($envelope, $pinellas);
        $hitsHills = $this->bboxIntersectsEnvelope($envelope, $hills);
        if ($hitsPinellas && $hitsHills) {
            return 'multi';
        }
        if ($hitsPinellas) {
            return 'pinellas';
        }
        if ($hitsHills) {
            return 'hillsborough';
        }

        return null;
    }

    /**
     * @param  array{minLon: float, minLat: float, maxLon: float, maxLat: float}  $env
     * @param  array{0: float, 1: float, 2: float, 3: float}  $box  west,south,east,north
     */
    private function bboxIntersectsEnvelope(array $env, array $box): bool
    {
        if (count($box) !== 4) {
            return false;
        }
        [$w, $s, $e, $n] = $box;

        return ! ($env['maxLon'] < $w || $env['minLon'] > $e || $env['maxLat'] < $s || $env['minLat'] > $n);
    }

    private function resolveLimit(Request $request, bool $fullAccess): int
    {
        $cap = (int) config('gis.max_features_cap');
        $requested = (int) ($request->query('limit', $fullAccess ? min(400, $cap) : min(120, $cap)));

        return max(1, min($cap, $requested));
    }

    /**
     * @param  array{minLon: float, minLat: float, maxLon: float, maxLat: float}  $envelope
     */
    private function queryHash(array $envelope, int $limit, bool $fullAccess, ?string $mlsCode, string $layers): string
    {
        $parts = [
            json_encode($envelope),
            (string) $limit,
            $fullAccess ? '1' : '0',
            $mlsCode ?? '',
            Str::lower($layers),
        ];

        return hash('sha256', implode('|', $parts));
    }

    /**
     * @param  array{minLon: float, minLat: float, maxLon: float, maxLat: float}  $envelope
     * @return array<string, mixed>
     */
    private function materializeFromSources(
        array $envelope,
        int $limit,
        bool $fullAccess,
        ?string $countyHint,
        ?string $mlsCode,
        string $layers,
    ): array {
        $warnings = [];
        $primary = (array) config('gis.sources.primary');
        $pinellas = (array) config('gis.sources.pinellas');
        $hillsborough = (array) config('gis.sources.hillsborough');

        $dor = (array) config('gis.dor_county_numbers');
        $wherePrimary = $this->dorWhereClause($countyHint, $dor);

        $fc = $this->queryArcGisSource($primary, $envelope, $limit, (string) $primary['out_fields'], $wherePrimary);
        if ($fc !== null) {
            return $this->finalizePayload($fc, $envelope, 'primary', $countyHint, $fullAccess, $mlsCode, $layers, $warnings, false);
        }
        $warnings[] = 'Primary Florida statewide cadastral source failed or timed out; attempting county failover.';

        $pinellasOverlap = $this->bboxIntersectsEnvelope($envelope, (array) $pinellas['bbox_wgs84']);
        if ($pinellasOverlap) {
            $fc = $this->queryArcGisSource($pinellas, $envelope, $limit, (string) $pinellas['out_fields'], '1=1');
            if ($fc !== null) {
                $warnings[] = 'Served from Pinellas County Enterprise GIS parcels.';

                return $this->finalizePayload($fc, $envelope, 'pinellas', $countyHint, $fullAccess, $mlsCode, $layers, $warnings, false);
            }
            $warnings[] = 'Pinellas failover failed.';
        }

        $hillsOverlap = $this->bboxIntersectsEnvelope($envelope, (array) $hillsborough['bbox_wgs84']);
        if ($hillsOverlap) {
            $fc = $this->queryArcGisSource($hillsborough, $envelope, $limit, (string) $hillsborough['out_fields'], '1=1');
            if ($fc !== null) {
                $warnings[] = 'Served from Hillsborough County HC_Parcels.';

                return $this->finalizePayload($fc, $envelope, 'hillsborough', $countyHint, $fullAccess, $mlsCode, $layers, $warnings, false);
            }
            $warnings[] = 'Hillsborough failover failed.';
        }

        $warnings[] = 'All ArcGIS parcel sources failed; returning empty FeatureCollection with OSM tile fallback hints.';

        return $this->finalizePayload([
            'type' => 'FeatureCollection',
            'features' => [],
        ], $envelope, 'degraded_osm_hint', $countyHint, $fullAccess, $mlsCode, $layers, $warnings, true);
    }

    /**
     * @param  array<string, mixed>  $source
     * @param  array{minLon: float, minLat: float, maxLon: float, maxLat: float}  $envelope
     * @return array<string, mixed>|null
     */
    private function queryArcGisSource(array $source, array $envelope, int $limit, string $outFields, string $where): ?array
    {
        $url = (string) $source['query_url'];
        $params = [
            'f' => 'geojson',
            'where' => $where,
            'geometry' => implode(',', [
                $envelope['minLon'],
                $envelope['minLat'],
                $envelope['maxLon'],
                $envelope['maxLat'],
            ]),
            'geometryType' => 'esriGeometryEnvelope',
            'inSR' => 4326,
            'spatialRel' => 'esriSpatialRelIntersects',
            'outFields' => $outFields,
            'returnGeometry' => 'true',
            'outSR' => 4326,
            'resultRecordCount' => $limit,
        ];

        try {
            $response = Http::connectTimeout((int) config('gis.http_connect_timeout_seconds'))
                ->timeout((int) config('gis.http_timeout_seconds'))
                ->acceptJson()
                ->get($url, $params);
        } catch (\Throwable) {
            return null;
        }

        if (! $response->successful()) {
            return null;
        }

        try {
            /** @var array<string, mixed> $decoded */
            $decoded = $response->json();
        } catch (\Throwable) {
            return null;
        }

        if (($decoded['type'] ?? null) !== 'FeatureCollection' || ! isset($decoded['features']) || ! is_array($decoded['features'])) {
            return null;
        }

        return $decoded;
    }

    /**
     * @param  array<string, int>  $dor
     */
    private function dorWhereClause(?string $countyHint, array $dor): string
    {
        if ($countyHint === 'pinellas' && isset($dor['pinellas'])) {
            return 'CO_NO='.$dor['pinellas'];
        }
        if ($countyHint === 'hillsborough' && isset($dor['hillsborough'])) {
            return 'CO_NO='.$dor['hillsborough'];
        }

        return '1=1';
    }

    /**
     * @param  array<string, mixed>  $featureCollection
     * @param  array{minLon: float, minLat: float, maxLon: float, maxLat: float}  $envelope
     * @param  array<int, string>  $warnings
     * @return array<string, mixed>
     */
    private function finalizePayload(
        array $featureCollection,
        array $envelope,
        string $tier,
        ?string $countyHint,
        bool $fullAccess,
        ?string $mlsCode,
        string $layers,
        array $warnings,
        bool $degraded,
    ): array {
        $sourceKey = match ($tier) {
            'primary' => (string) config('gis.sources.primary.key'),
            'pinellas' => (string) config('gis.sources.pinellas.key'),
            'hillsborough' => (string) config('gis.sources.hillsborough.key'),
            default => 'degraded_osm_hint',
        };

        $meta = [
            'source_used' => $sourceKey,
            'source_tier' => $tier,
            'county_hint' => $countyHint,
            'teaser' => false,
            'full_access' => $fullAccess,
            'mls_code' => $mlsCode,
            'layers' => array_values(array_filter(array_map(trim(...), explode(',', $layers)))),
            'cached' => false,
            'cache_hit' => null,
            'warnings' => $warnings,
            'degraded' => $degraded,
            'bbox' => [
                'min_lon' => $envelope['minLon'],
                'min_lat' => $envelope['minLat'],
                'max_lon' => $envelope['maxLon'],
                'max_lat' => $envelope['maxLat'],
            ],
        ];

        if ($fullAccess) {
            $meta['context_layers'] = config('gis.full_access_context_layers', []);
        }

        if ($degraded) {
            $meta['leaflet_fallback'] = config('gis.degraded_tile_layer');
        }

        $featureCollection['meta'] = $meta;

        return $featureCollection;
    }

    /**
     * @param  array<string, mixed>  $fc
     * @return array<string, mixed>
     */
    private function applyTeaserMode(array $fc): array
    {
        $max = (int) config('gis.teaser_max_features');
        $decimals = (int) config('gis.teaser_coordinate_decimals');
        /** @var list<array<string, mixed>> $features */
        $features = array_slice($fc['features'], 0, $max);
        foreach ($features as $i => $feature) {
            if (! is_array($feature)) {
                continue;
            }
            $features[$i] = $this->stripTeaserFeature($feature, $decimals);
        }
        $fc['features'] = $features;

        return $fc;
    }

    /**
     * @param  array<string, mixed>  $feature
     * @return array<string, mixed>
     */
    private function stripTeaserFeature(array $feature, int $decimals): array
    {
        if (isset($feature['geometry']) && is_array($feature['geometry'])) {
            $feature['geometry'] = $this->roundGeometryCoordinates($feature['geometry'], $decimals);
        }
        if (isset($feature['properties']) && is_array($feature['properties'])) {
            $feature['properties'] = $this->pickTeaserProperties($feature['properties']);
        }

        return $feature;
    }

    /**
     * @param  array<string, mixed>  $geometry
     * @return array<string, mixed>
     */
    private function roundGeometryCoordinates(array $geometry, int $decimals): array
    {
        if (($geometry['type'] ?? null) === 'Polygon' && isset($geometry['coordinates']) && is_array($geometry['coordinates'])) {
            $geometry['coordinates'] = $this->roundPolygonRings($geometry['coordinates'], $decimals);
        }
        if (($geometry['type'] ?? null) === 'MultiPolygon' && isset($geometry['coordinates']) && is_array($geometry['coordinates'])) {
            $rings = [];
            foreach ($geometry['coordinates'] as $polygon) {
                if (is_array($polygon)) {
                    $rings[] = $this->roundPolygonRings($polygon, $decimals);
                }
            }
            $geometry['coordinates'] = $rings;
        }

        return $geometry;
    }

    /**
     * @param  array<mixed>  $rings
     * @return array<mixed>
     */
    private function roundPolygonRings(array $rings, int $decimals): array
    {
        $out = [];
        foreach ($rings as $ring) {
            if (! is_array($ring)) {
                continue;
            }
            $r = [];
            foreach ($ring as $pt) {
                if (is_array($pt) && isset($pt[0], $pt[1]) && is_numeric($pt[0]) && is_numeric($pt[1])) {
                    $r[] = [round((float) $pt[0], $decimals), round((float) $pt[1], $decimals)];
                }
            }
            $out[] = $r;
        }

        return $out;
    }

    /**
     * @param  array<string, mixed>  $properties
     * @return array<string, mixed>
     */
    private function pickTeaserProperties(array $properties): array
    {
        $keep = [];
        foreach ($properties as $k => $v) {
            if (! is_string($k)) {
                continue;
            }
            if (preg_match('/(PARCEL|FOLIO|STRAP|SITE|ADDRESS|CO_NO|OBJECTID)/i', $k) === 1) {
                if (is_scalar($v)) {
                    $keep[$k] = Str::limit((string) $v, 48, '');
                }
            }
        }

        return $keep !== [] ? $keep : ['OBJECTID' => $properties['OBJECTID'] ?? null];
    }

    private function persistFilesystemBackup(string $hash, string $encoded): void
    {
        if (config('gis.queue_backup_writes')) {
            PersistGisGeoJsonBackupJob::dispatch($hash, $encoded)
                ->onQueue((string) config('gis.queue'));

            return;
        }

        Storage::disk('gis_backup')->put($hash.'.json', $encoded);
    }
}
