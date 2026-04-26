<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Services\Bridge\BridgeHttpService;
use App\Services\Bridge\BridgeProxyAuditLogger;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class ImageProxyController extends Controller
{
    public function __construct(
        private readonly BridgeHttpService $bridge,
        private readonly BridgeProxyAuditLogger $audit,
    ) {}

    /**
     * Revenue impact: immutable long-cache headers let Cloudflare POPs serve repeat photo views
     * for a year without origin hits, cutting infra cost while speeding high-intent listing galleries.
     */
    public function show(Request $request, string $listingKey, string $photoId): Response
    {
        $url = $this->bridge->listingPhotoUrl($listingKey, $photoId);
        $response = $this->bridge->getBinaryFromUrl($url, $request);

        if (! $response->successful()) {
            $this->audit->log(
                $request,
                'image.detail',
                null,
                $request->attributes->get('bridge.domain_slug'),
                $request->attributes->get('bridge.token_name'),
                $request->attributes->get('bridge.user_id'),
            );

            return response('Image not found.', $response->status());
        }

        return $this->originImageResponse(
            $request,
            $listingKey,
            $photoId,
            $response->body(),
            $response->header('Content-Type')
        );
    }

    private function originImageResponse(
        Request $request,
        string $listingKey,
        string $photoId,
        string $body,
        ?string $contentType = null,
    ): Response {
        $mime = $contentType ?: 'application/octet-stream';

        $this->audit->log(
            $request,
            'image.detail',
            1,
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        $public = rtrim((string) config('bridge.images_public_base'), '/').'/images/'.$listingKey.'/'.$photoId;

        return response($body, 200, [
            'Content-Type' => $mime,
            'Cache-Control' => 'public, max-age=31536000, immutable',
            'X-IDX-Proxied-Public-Url' => $public,
        ]);
    }
}
