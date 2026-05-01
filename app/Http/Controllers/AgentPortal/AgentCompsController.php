<?php

namespace App\Http\Controllers\AgentPortal;

use App\Http\Controllers\Controller;
use App\Http\Requests\Comps\CompsRunRequest;
use App\Models\User;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use App\Services\Bridge\BridgeCompsService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AgentCompsController extends Controller
{
    /**
     * Session-authenticated comps runner for the Filament marketing workspace (home value / CMA-lite).
     * Supplies Bridge dataset resolution + idx:full-style access consistent with authenticated subscribers.
     */
    public function run(
        CompsRunRequest $request,
        BridgeCompsService $comps,
        SubscriberFeedAccessService $feeds,
    ): JsonResponse {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $scopes = $feeds->resolvedSearchScopesForUser($user);
        if ($scopes === []) {
            return response()->json([
                'message' => 'No MLS datasets enabled for comps.',
                'errors' => [
                    'feed_access' => ['IDX search is not enabled for any MLS feed on your account.'],
                ],
            ], 422);
        }

        $datasets = array_values(array_unique(array_map(
            static fn (array $s): string => (string) $s['dataset_code'],
            $scopes
        )));

        $requestedDataset = trim((string) $request->query('dataset', ''));
        if ($requestedDataset !== '' && ! in_array($requestedDataset, $datasets, true)) {
            return response()->json([
                'message' => 'Dataset not permitted.',
                'errors' => [
                    'dataset' => ['That MLS dataset is not in your IDX access list.'],
                ],
            ], 422);
        }

        $dataset = $requestedDataset !== '' ? $requestedDataset : $datasets[0];
        $validated = $request->validated();

        $internal = Request::create('/agent-portal/comps/run', 'POST', $validated);
        $internal->attributes->set('bridge.full_access', true);
        $internal->attributes->set('bridge.allowed_datasets', $datasets);
        $internal->query->set('dataset', $dataset);

        $payload = $comps->run($internal, $validated);

        return response()->json($payload);
    }
}
