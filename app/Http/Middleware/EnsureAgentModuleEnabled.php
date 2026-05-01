<?php

namespace App\Http\Middleware;

use App\Models\User;
use App\Services\AgentPortal\FeatureFlagService;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class EnsureAgentModuleEnabled
{
    /**
     * Handle an incoming request.
     *
     * @param  Closure(Request): (Response)  $next
     */
    public function handle(Request $request, Closure $next, string $module): Response
    {
        /** @var User|null $user */
        $user = $request->user();
        if ($user === null) {
            return $next($request);
        }

        $enabled = app(FeatureFlagService::class)->isEnabled($user, $module);
        if (! $enabled) {
            return response()->json([
                'message' => sprintf('The %s module is disabled for this account.', $module),
                'module' => $module,
            ], 403);
        }

        return $next($request);
    }
}
