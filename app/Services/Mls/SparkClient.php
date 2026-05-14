<?php

namespace App\Services\Mls;

use App\Enums\MlsProvider;
use Illuminate\Http\Client\Response;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Symfony\Component\HttpKernel\Exception\HttpException;

/**
 * Revenue impact: Spark-specific OAuth + RESO keeps Space Coast / Beaches feeds billable and attributable.
 *
 * Compliance: per-feed OAuth credentials must never be shared across MLS orgs; tokens stay server-side (IDX only).
 */
final class SparkClient
{
    /**
     * @param  array<string, mixed>  $definition
     */
    public function __construct(
        private string $feedCode,
        private array $definition,
    ) {}

    public function feedCode(): string
    {
        return $this->feedCode;
    }

    public function provider(): MlsProvider
    {
        return MlsProvider::Spark;
    }

    /**
     * Fetch RESO OData JSON using a Bearer token obtained via client credentials.
     *
     * @param  array<string, scalar|list<string>>  $query
     */
    public function getJson(string $relativePath, Request $incoming, array $query = []): Response
    {
        $base = (string) ($this->definition['reso_base_url'] ?? '');
        if ($base === '') {
            throw new HttpException(503, 'Spark RESO base URL is not configured for feed '.$this->feedCode);
        }

        $url = rtrim($base, '/').'/'.ltrim($relativePath, '/');
        $token = $this->accessToken();

        return Http::timeout((int) config('bridge.timeout_seconds', 30))
            ->withHeaders($this->forwardClientHeaders($incoming))
            ->withToken($token)
            ->acceptJson()
            ->get($url, $query);
    }

    /**
     * Active + Pending RESO Property collection (Bridge-shaped `value` array for collection proxies).
     *
     * @param  array<string, scalar|list<string>>  $extraQuery
     */
    public function getActivePendingPropertyCollection(Request $incoming, array $extraQuery = []): Response
    {
        $filter = "StandardStatus eq 'Active' or StandardStatus eq 'Pending'";
        $query = array_merge([
            '$filter' => $filter,
            '$top' => 200,
        ], $extraQuery);

        return $this->getJson('Property', $incoming, $query);
    }

    /**
     * Single Property entity (including closed) for on-demand IDX detail views.
     *
     * @param  array<string, scalar|list<string>>  $query
     */
    public function getPropertyEntity(Request $incoming, string $listingKey, array $query = []): Response
    {
        $escaped = str_replace("'", "''", $listingKey);

        return $this->getJson("Property('{$escaped}')", $incoming, $query);
    }

    private function accessToken(): string
    {
        $store = (string) config('mls.spark_token_cache_store', 'file');
        $ttl = (int) config('mls.spark_token_cache_ttl_seconds', 3300);
        $cacheKey = 'mls_spark_token:'.$this->feedCode;

        /** @var string $token */
        $token = Cache::store($store)->remember($cacheKey, $ttl, function (): string {
            return $this->fetchFreshAccessToken();
        });

        return $token;
    }

    private function fetchFreshAccessToken(): string
    {
        $tokenUrl = (string) ($this->definition['token_url'] ?? '');
        $clientId = (string) ($this->definition['client_id'] ?? '');
        $clientSecret = (string) ($this->definition['client_secret'] ?? '');
        $scope = (string) ($this->definition['oauth_scope'] ?? 'api');

        if ($tokenUrl === '' || $clientId === '' || $clientSecret === '') {
            throw new HttpException(503, 'Spark OAuth is not fully configured for feed '.$this->feedCode);
        }

        $response = Http::asForm()->timeout((int) config('bridge.timeout_seconds', 30))->post($tokenUrl, [
            'grant_type' => 'client_credentials',
            'client_id' => $clientId,
            'client_secret' => $clientSecret,
            'scope' => $scope,
        ]);

        if (! $response->successful()) {
            throw new HttpException(502, 'Spark token endpoint rejected credentials for feed '.$this->feedCode);
        }

        $access = $response->json('access_token');
        if (! is_string($access) || $access === '') {
            throw new HttpException(502, 'Spark token response missing access_token for feed '.$this->feedCode);
        }

        return $access;
    }

    /**
     * @return array<string, string>
     */
    private function forwardClientHeaders(Request $incoming): array
    {
        $headers = [];

        $map = [
            'accept' => 'Accept',
            'accept-language' => 'Accept-Language',
            'prefer' => 'Prefer',
            'if-none-match' => 'If-None-Match',
        ];

        foreach ($map as $from => $to) {
            $value = $incoming->headers->get($from);
            if (is_string($value) && $value !== '') {
                $headers[$to] = $value;
            }
        }

        return $headers;
    }
}
