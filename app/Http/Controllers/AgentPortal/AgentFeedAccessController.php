<?php

namespace App\Http\Controllers\AgentPortal;

use App\Http\Controllers\Controller;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AgentFeedAccessController extends Controller
{
    public function __invoke(Request $request, SubscriberFeedAccessService $feeds): JsonResponse
    {
        $user = $request->user();
        abort_if($user === null, 401);

        return response()->json([
            'scopes' => $feeds->detailedFeedScopesForUser($user),
        ]);
    }
}
