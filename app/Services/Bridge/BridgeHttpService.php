<?php

namespace App\Services\Bridge;

use Illuminate\Http\Client\Response;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Http;

class BridgeHttpService
{
    /**
     * Revenue impact: a single HTTP wrapper keeps Bridge credentials off edge devices
     * and centralizes timeouts so slow MLS responses do not tie up checkout flows.
     */
    public function getJsonFromUrl(string $url, Request $incoming, array $query = []): Response
    {
        $token = config('bridge.server_token');
        if (! is_string($token) || $token === '') {
            abort(503, 'Bridge server token is not configured.');
        }

        $mergedQuery = array_merge($incoming->query(), $query);
        foreach (['domain', 'teaser'] as $internal) {
            unset($mergedQuery[$internal]);
        }

        return Http::timeout((int) config('bridge.timeout_seconds'))
            ->withHeaders($this->forwardClientHeaders($incoming))
            ->withToken($token)
            ->acceptJson()
            ->get($url, $mergedQuery);
    }

    /**
     * Bridge Web API paths documented as /{dataset}/listings, /{dataset}/agents, etc.
     */
    public function webUrl(string $resourcePath): string
    {
        $resourcePath = ltrim($resourcePath, '/');

        return $this->joinUrl($this->datasetPathPrefix().'/'.$resourcePath);
    }

    /**
     * RESO resources (Property, Member, Office, OpenHouse, Lookup).
     */
    public function resoCollectionUrl(string $resource): string
    {
        $resource = ltrim($resource, '/');
        $dataset = (string) config('bridge.dataset');
        $resoRoot = trim((string) config('bridge.reso_root', ''), '/');

        if ($resoRoot === '') {
            return $this->joinUrl($dataset.'/'.$resource);
        }

        return $this->joinUrl($resoRoot.'/'.$dataset.'/'.$resource);
    }

    public function resoEntityUrl(string $resource, string $key): string
    {
        $safeKey = str_replace("'", "''", $key);

        return $this->resoCollectionUrl($resource)."('{$safeKey}')";
    }

    public function listingPhotoUrl(string $listingKey, string $photoId): string
    {
        $template = (string) config('bridge.listing_photo_path_template');

        if (str_starts_with($template, 'http://') || str_starts_with($template, 'https://')) {
            return str_replace(
                ['{dataset}', '{listingKey}', '{photoId}'],
                [(string) config('bridge.dataset'), rawurlencode($listingKey), rawurlencode($photoId)],
                $template
            );
        }

        $path = str_replace(
            ['{dataset}', '{listingKey}', '{photoId}'],
            [(string) config('bridge.dataset'), rawurlencode($listingKey), rawurlencode($photoId)],
            $template
        );

        return $this->joinUrl(ltrim($path, '/'));
    }

    /**
     * Doc-style absolute paths that are not always under /{dataset}/ (e.g. /pub/parcels).
     */
    public function docPath(string $path): string
    {
        return rtrim((string) config('bridge.host'), '/').'/'.ltrim($path, '/');
    }

    public function getBinaryFromUrl(string $url, Request $incoming): Response
    {
        $token = config('bridge.server_token');
        if (! is_string($token) || $token === '') {
            abort(503, 'Bridge server token is not configured.');
        }

        return Http::timeout((int) config('bridge.timeout_seconds'))
            ->withHeaders($this->forwardClientHeaders($incoming))
            ->withToken($token)
            ->withHeaders([
                'Accept' => 'image/jpeg,image/webp,image/*;q=0.8,*/*;q=0.5',
            ])
            ->get($url);
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

    private function joinUrl(string $path): string
    {
        return rtrim((string) config('bridge.host'), '/').'/'.ltrim($path, '/');
    }

    private function datasetPathPrefix(): string
    {
        $prefix = trim((string) config('bridge.path_prefix', ''), '/');
        $dataset = trim((string) config('bridge.dataset', ''), '/');

        if ($prefix === '') {
            return $dataset;
        }

        return $prefix.'/'.$dataset;
    }
}
