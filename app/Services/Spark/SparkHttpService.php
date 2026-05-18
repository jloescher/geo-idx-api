<?php

declare(strict_types=1);

namespace App\Services\Spark;

use Illuminate\Http\Client\Response;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Http;

/**
 * Revenue impact: server-side Spark RESO access keeps BeachesMLS credentials off browsers.
 *
 * Compliance: Bearer token remains server-side per IDX feed agreement.
 */
final class SparkHttpService
{
    public function __construct(
        private readonly SparkRateLimitGuard $rateLimitGuard,
    ) {}

    /**
     * @param  array<string, scalar|list<string>>  $query
     */
    public function serverJsonGet(string $fullUrl, array $query = []): Response
    {
        return $this->rateLimitedRequest(function () use ($fullUrl, $query): Response {
            $token = $this->resolveBearerToken();
            if ($token === null) {
                abort(503, 'Spark access token is not configured.');
            }

            $timeout = (int) config('spark.timeout_seconds');

            return SparkUpstreamRetry::run(fn () => Http::timeout($timeout)
                ->withToken($token)
                ->acceptJson()
                ->get($fullUrl, $query));
        });
    }

    public function getAuthorizedJson(string $url, Request $incoming): Response
    {
        $token = $this->resolveBearerToken();
        if ($token === null) {
            abort(503, 'Spark access token is not configured.');
        }

        $timeout = (int) config('spark.timeout_seconds');

        return $this->rateLimitedRequest(function () use ($incoming, $url, $token, $timeout): Response {
            return SparkUpstreamRetry::run(fn () => Http::timeout($timeout)
                ->withHeaders($this->forwardClientHeaders($incoming))
                ->withToken($token)
                ->acceptJson()
                ->get($url));
        });
    }

    public function getJsonFromUrl(string $url, Request $incoming, array $query = [], array $stripQueryKeys = []): Response
    {
        $token = $this->resolveBearerToken();
        if ($token === null) {
            abort(503, 'Spark access token is not configured.');
        }

        $mergedQuery = array_merge($incoming->query(), $query);
        foreach (array_merge(['domain', 'teaser'], $stripQueryKeys) as $internal) {
            unset($mergedQuery[$internal]);
        }

        $timeout = (int) config('spark.timeout_seconds');

        return $this->rateLimitedRequest(function () use ($incoming, $url, $token, $mergedQuery, $timeout): Response {
            return SparkUpstreamRetry::run(fn () => Http::timeout($timeout)
                ->withHeaders($this->forwardClientHeaders($incoming))
                ->withToken($token)
                ->acceptJson()
                ->get($url, $mergedQuery));
        });
    }

    public function propertyCollectionUrl(): string
    {
        return rtrim((string) config('spark.reso_base_url'), '/').'/Property';
    }

    public function propertyEntityUrl(string $listingKey): string
    {
        $encoded = rawurlencode($listingKey);

        return rtrim((string) config('spark.reso_base_url'), '/')."/Property('{$encoded}')";
    }

    /**
     * Resolve Spark CDN URL for a listing photo via expanded Media on the Property entity.
     */
    public function resolveListingPhotoUrl(string $listingKey, string $mediaKey): string
    {
        $response = $this->serverJsonGet($this->propertyEntityUrl($listingKey), [
            '$expand' => 'Media',
        ]);

        if (! $response->successful()) {
            abort($response->status(), 'Spark listing media could not be loaded.');
        }

        $payload = $response->json();
        if (! is_array($payload)) {
            abort(502, 'Spark listing media response was invalid.');
        }

        $mediaItems = $payload['Media'] ?? [];
        if (! is_array($mediaItems)) {
            abort(404, 'Listing photo not found.');
        }

        foreach ($mediaItems as $media) {
            if (! is_array($media)) {
                continue;
            }
            $key = $media['MediaKey'] ?? $media['OriginatingSystemMediaKey'] ?? null;
            if (! is_string($key) || $key !== $mediaKey) {
                continue;
            }
            $url = $media['MediaURL'] ?? null;
            if (is_string($url) && $url !== '') {
                return $url;
            }
        }

        abort(404, 'Listing photo not found.');
    }

    public function getBinaryFromUrl(string $url, Request $incoming): Response
    {
        $timeout = (int) config('spark.timeout_seconds');

        return SparkUpstreamRetry::run(fn () => Http::timeout($timeout)
            ->withHeaders($this->forwardClientHeaders($incoming))
            ->withHeaders([
                'Accept' => 'image/jpeg,image/webp,image/*;q=0.8,*/*;q=0.5',
            ])
            ->get($url));
    }

    /**
     * @return array<string, string>
     */
    private function forwardClientHeaders(Request $incoming): array
    {
        $headers = [];
        foreach (['Accept', 'Accept-Language', 'User-Agent'] as $name) {
            $value = $incoming->header($name);
            if (is_string($value) && $value !== '') {
                $headers[$name] = $value;
            }
        }

        return $headers;
    }

    /**
     * @param  callable(): Response  $request
     */
    private function rateLimitedRequest(callable $request): Response
    {
        $this->rateLimitGuard->acquire();
        $response = $request();
        $this->rateLimitGuard->recordFromResponse($response);

        return $response;
    }

    private function resolveBearerToken(): ?string
    {
        $token = config('spark.access_token');

        return is_string($token) && trim($token) !== '' ? trim($token) : null;
    }
}
