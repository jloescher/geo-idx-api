<?php

namespace App\Ghl\Sync\Services;

use App\Ghl\Api\Clients\GhlApiClient;
use App\Ghl\OAuth\Models\GhlOAuthToken;
use Illuminate\Http\Client\Response;

class TagManager
{
    public function __construct(
        private readonly GhlApiClient $client,
    ) {}

    /**
     * @param  list<string>  $tags
     */
    public function addTags(GhlOAuthToken $token, string $contactId, array $tags): Response
    {
        return $this->client->request($token, 'post', "/contacts/{$contactId}/tags", [
            'json' => ['tags' => $tags],
        ]);
    }
}
