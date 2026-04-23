<?php

namespace App\Ghl\Sync\Services;

use App\Ghl\Api\Clients\GhlApiClient;
use App\Ghl\OAuth\Models\GhlOAuthToken;
use Illuminate\Http\Client\Response;

class OpportunityCreator
{
    public function __construct(
        private readonly GhlApiClient $client,
    ) {}

    /**
     * @param  array<string, mixed>  $payload
     */
    public function create(GhlOAuthToken $token, array $payload): Response
    {
        return $this->client->request($token, 'post', '/opportunities/', [
            'json' => $payload,
        ]);
    }
}
