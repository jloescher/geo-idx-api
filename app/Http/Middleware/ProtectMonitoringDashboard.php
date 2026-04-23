<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class ProtectMonitoringDashboard
{
    public function handle(Request $request, Closure $next): Response
    {
        if (! app()->environment('production')) {
            return $next($request);
        }

        $username = (string) env('MONITORING_DASHBOARD_USERNAME', '');
        $password = (string) env('MONITORING_DASHBOARD_PASSWORD', '');

        if (
            $username === ''
            || $password === ''
            || ! hash_equals($username, (string) $request->getUser())
            || ! hash_equals($password, (string) $request->getPassword())
        ) {
            return response('Unauthorized', 401, ['WWW-Authenticate' => 'Basic realm="Monitoring Dashboard"']);
        }

        return $next($request);
    }
}
