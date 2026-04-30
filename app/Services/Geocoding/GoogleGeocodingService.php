<?php

namespace App\Services\Geocoding;

use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

final class GoogleGeocodingService
{
    public function __construct(
        private readonly string $apiKey,
        private readonly int $cacheTtl,
        private readonly int $timeout,
    ) {}

    public function geocode(string $address): ?GeocodingResult
    {
        $cacheKey = $this->cacheKey($address);

        return Cache::remember($cacheKey, $this->cacheTtl, function () use ($address): ?GeocodingResult {
            return $this->fetchFromApi($address);
        });
    }

    private function fetchFromApi(string $address): ?GeocodingResult
    {
        if ($this->apiKey === '' || $this->apiKey === '0') {
            Log::warning('Geocoding skipped: Google Maps API key not configured.');

            return null;
        }

        $response = Http::timeout($this->timeout)
            ->get('https://maps.googleapis.com/maps/api/geocode/json', [
                'address' => $address,
                'key' => $this->apiKey,
                'region' => 'us',
            ]);

        if (! $response->successful()) {
            Log::warning('Geocoding API request failed.', [
                'status' => $response->status(),
                'address' => $address,
            ]);

            return null;
        }

        $data = $response->json();

        if (($data['status'] ?? '') !== 'OK' || empty($data['results'][0]['geometry']['location'])) {
            Log::info('Geocoding returned no results.', [
                'status' => $data['status'] ?? 'unknown',
                'address' => $address,
            ]);

            return null;
        }

        $result = $data['results'][0];
        $location = $result['geometry']['location'];

        return new GeocodingResult(
            lat: (float) $location['lat'],
            lng: (float) $location['lng'],
            formattedAddress: $result['formatted_address'],
            placeId: $result['place_id'],
        );
    }

    private function cacheKey(string $address): string
    {
        return 'geocode:'.md5(mb_strtolower(trim($address)));
    }
}
