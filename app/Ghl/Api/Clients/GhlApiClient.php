<?php

namespace App\Ghl\Api\Clients;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Services\GhlAuditService;
use Illuminate\Http\Client\Response;
use Illuminate\Support\Facades\Http;

class GhlApiClient
{
    public function __construct(
        private readonly GhlAuditService $audit,
    ) {}

    public function request(GhlOAuthToken $token, string $method, string $path, array $options = []): Response
    {
        $base = rtrim((string) config('ghl.api.base_url'), '/');
        $url = $base.'/'.ltrim($path, '/');
        $started = microtime(true);

        $response = Http::withToken($token->access_token)
            ->withHeaders([
                'Version' => (string) config('ghl.api.version'),
                'Accept' => 'application/json',
            ])
            ->timeout((int) config('ghl.api.timeout'))
            ->send(strtoupper($method), $url, $options);

        $latency = (int) ((microtime(true) - $started) * 1000);

        $this->audit->forToken($token, $path, strtoupper($method), [
            'response_status' => $response->status(),
            'latency_ms' => $latency,
            'error_details' => $response->failed() ? $response->body() : null,
            'sync_status' => $response->successful() ? 'success' : 'failed',
        ]);

        return $response;
    }
}
