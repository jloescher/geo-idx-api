<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Services\Bridge\BridgeHttpService;
use App\Services\Bridge\BridgeProxyAuditLogger;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Storage;
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
        $disk = Storage::disk('images');
        $relativePath = $listingKey.'/'.$photoId;

        $originTtl = (int) config('bridge.image_cache_ttl_seconds');

        if ($disk->exists($relativePath)) {
            $lastModified = $disk->lastModified($relativePath);
            if (is_int($lastModified) && $lastModified > (time() - $originTtl)) {
                return $this->cachedFileResponse($request, $listingKey, $photoId, $disk->path($relativePath));
            }
        }

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

        $disk->put($relativePath, $response->body());

        return $this->cachedFileResponse(
            $request,
            $listingKey,
            $photoId,
            $disk->path($relativePath),
            $response->header('Content-Type')
        );
    }

    private function cachedFileResponse(
        Request $request,
        string $listingKey,
        string $photoId,
        string $absolutePath,
        ?string $contentType = null,
    ): Response {
        $mime = $contentType ?: (mime_content_type($absolutePath) ?: 'application/octet-stream');

        $this->audit->log(
            $request,
            'image.detail',
            1,
            $request->attributes->get('bridge.domain_slug'),
            $request->attributes->get('bridge.token_name'),
            $request->attributes->get('bridge.user_id'),
        );

        $public = rtrim((string) config('bridge.images_public_base'), '/').'/images/'.$listingKey.'/'.$photoId;

        return response()->file($absolutePath, [
            'Content-Type' => $mime,
            'Cache-Control' => 'public, max-age=31536000, immutable',
            'X-IDX-Proxied-Public-Url' => $public,
        ]);
    }
}
