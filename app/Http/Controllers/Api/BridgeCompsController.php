<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Http\Requests\Comps\CompsRunRequest;
use App\Services\Bridge\BridgeCompsService;
use Illuminate\Http\JsonResponse;

class BridgeCompsController extends Controller
{
    public function __construct(
        private readonly BridgeCompsService $comps,
    ) {}

    /**
     * Revenue impact: comparables are a premium workflow; idx:full unlocks full comp sets
     * and competition/overpriced analytics while domain traffic receives a capped teaser slice.
     */
    public function run(CompsRunRequest $request): JsonResponse
    {
        $payload = $this->comps->run($request, $request->validated());

        return response()->json($payload);
    }
}
