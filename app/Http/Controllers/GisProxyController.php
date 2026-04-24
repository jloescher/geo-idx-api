<?php

namespace App\Http\Controllers;

use App\Http\Requests\GisProxyRequest;
use App\Services\GisProxyService;
use Illuminate\Http\JsonResponse;

class GisProxyController extends Controller
{
    public function __construct(
        private readonly GisProxyService $gis,
    ) {}

    /**
     * Revenue impact: `/v1/gis` pairs with `/v1/listings` so geo-web can paint
     * parcels immediately after listing markers, increasing map engagement depth.
     */
    public function show(GisProxyRequest $request): JsonResponse
    {
        $payload = $this->gis->fetch($request, null);

        return $this->jsonResponse($payload);
    }

    /**
     * Revenue impact: MLS-scoped routing future-proofs multi-board Florida revenue
     * without breaking existing Stellar embeds (same GIS, different analytics keys).
     */
    public function showForMls(GisProxyRequest $request, string $mlsCode): JsonResponse
    {
        $allowed = (array) config('gis.florida_mls_codes', []);
        $normalized = mb_strtolower($mlsCode);
        $allowedLower = array_map(fn (mixed $c): string => mb_strtolower((string) $c), $allowed);
        if ($allowed !== [] && ! in_array($normalized, $allowedLower, true)) {
            abort(404, 'Unknown Florida MLS code.');
        }

        $payload = $this->gis->fetch($request, $normalized);

        return $this->jsonResponse($payload);
    }

    /**
     * @param  array<string, mixed>  $payload
     */
    private function jsonResponse(array $payload): JsonResponse
    {
        $ttl = (int) config('gis.cache_ttl_seconds');

        // Revenue impact: short edge cache mirrors listings freshness so Cloudflare can shield origin during spikes.
        return response()->json($payload)
            ->header('Cache-Control', 'public, max-age='.min(600, max(60, $ttl)));
    }
}
