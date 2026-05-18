<?php

namespace App\Services\Bridge;

/**
 * Revenue impact: forcing MLS photos through the IDX images host (see config `idx.images_public_url`)
 * maximizes edge cache hit rate and page speed, which directly lifts lead-form conversion.
 */
final class BridgeImageUrlRewriter
{
    /**
     * Rewrite Bridge-hosted listing photo URLs in JSON to IDX image CDN URLs.
     *
     * @return string JSON (may be identical if nothing matched)
     */
    public function rewriteJson(string $json): string
    {
        try {
            $data = json_decode($json, true, 512, JSON_THROW_ON_ERROR);
        } catch (\JsonException) {
            return $json;
        }

        if (! is_array($data)) {
            return $json;
        }

        try {
            $rewritten = $this->rewriteValue($data, null);
        } catch (\Throwable) {
            return $json;
        }

        try {
            return json_encode($rewritten, JSON_THROW_ON_ERROR | JSON_UNESCAPED_SLASHES);
        } catch (\JsonException) {
            return $json;
        }
    }

    private function publicPhotoUrl(string $listingKey, string $photoId): string
    {
        $base = rtrim((string) config('bridge.images_public_base'), '/');
        $listingKey = rawurlencode($listingKey);
        $photoId = rawurlencode($photoId);

        return "{$base}/images/{$listingKey}/{$photoId}";
    }

    private function rewriteValue(mixed $node, ?string $listingKey): mixed
    {
        if (is_array($node)) {
            $listingKey = $this->resolveListingKey($node, $listingKey);

            if (isset($node['Media']) && is_array($node['Media'])) {
                $node['Media'] = $this->rewriteMediaItems($node['Media'], $listingKey);
            }

            foreach ($node as $key => $value) {
                if ($key === 'Media') {
                    continue;
                }
                if (is_string($value)) {
                    $node[$key] = $this->rewriteStringUrl($value, $listingKey);
                } elseif (is_array($value)) {
                    $node[$key] = $this->rewriteValue($value, $listingKey);
                }
            }

            return $node;
        }

        if (is_string($node)) {
            return $this->rewriteStringUrl($node, $listingKey);
        }

        return $node;
    }

    /**
     * @param  array<string, mixed>  $assoc
     */
    private function resolveListingKey(array $assoc, ?string $fallback): ?string
    {
        if (array_is_list($assoc)) {
            return $fallback;
        }

        foreach (['ListingKey', 'ListingId', 'ResourceRecordKey', 'ListingKeyNumeric'] as $k) {
            if (isset($assoc[$k]) && is_string($assoc[$k]) && $assoc[$k] !== '') {
                return $assoc[$k];
            }
        }

        return $fallback;
    }

    /**
     * @param  array<int, mixed>  $items
     * @return array<int, mixed>
     */
    private function rewriteMediaItems(array $items, ?string $listingKey): array
    {
        foreach ($items as $i => $item) {
            if (! is_array($item)) {
                continue;
            }

            $lk = $this->resolveListingKey($item, $listingKey);
            $photoId = null;
            foreach (['Order', 'MediaKey', 'Id', 'MediaModificationTimestamp'] as $pk) {
                if (isset($item[$pk]) && (is_string($item[$pk]) || is_int($item[$pk]))) {
                    $photoId = (string) $item[$pk];

                    break;
                }
            }

            foreach ($item as $key => $value) {
                if (! is_string($value)) {
                    continue;
                }
                if (! $this->looksLikeImageFieldName((string) $key)) {
                    continue;
                }

                $rewritten = $this->rewriteStringUrl($value, $lk);
                if (
                    $rewritten === $value
                    && $lk !== null
                    && $photoId !== null
                    && ($this->shouldRewriteHost($value) || $this->isCloudFrontHost($value))
                ) {
                    $rewritten = $this->publicPhotoUrl($lk, $photoId);
                }
                $item[$key] = $rewritten;
            }

            $items[$i] = $this->rewriteValue($item, $lk);
        }

        return $items;
    }

    private function looksLikeImageFieldName(string $key): bool
    {
        $k = strtolower($key);

        return str_contains($k, 'mediaurl')
            || str_contains($k, 'photourl')
            || str_contains($k, 'imageurl')
            || str_contains($k, 'thumbnail')
            || str_contains($k, 'photomodification');
    }

    private function rewriteStringUrl(string $value, ?string $listingKey): string
    {
        if (! str_contains($value, 'http') && ! str_contains($value, '/listings/')) {
            return $value;
        }

        $normalizedIdxImages = $this->normalizeIdxImagesUrl($value);
        if ($normalizedIdxImages !== null) {
            return $normalizedIdxImages;
        }

        $pattern = '~https?://[^/\s"]+/.+?/listings/([^/"\s?#]+)/photos/([^/"\s?#]+)~i';

        $out = preg_replace_callback(
            $pattern,
            function (array $m): string {
                $full = $m[0];
                if (! $this->shouldRewriteHost($full)) {
                    return $full;
                }

                $lk = rawurldecode($m[1]);
                $pid = rawurldecode($m[2]);

                return $this->publicPhotoUrl($lk, $pid);
            },
            $value
        );

        if (! is_string($out)) {
            return $value;
        }

        if ($out === $value && $listingKey !== null && $this->shouldRewriteHost($value) && preg_match('~\.(?:jpe?g|png|webp)(\?|$)~i', $value)) {
            if (preg_match('~[?&](?:id|photo|order)=([^&\s"#]+)~i', $value, $mm)) {
                return $this->publicPhotoUrl($listingKey, rawurldecode($mm[1]));
            }
        }

        return $out;
    }

    private function normalizeIdxImagesUrl(string $url): ?string
    {
        $host = (string) parse_url($url, PHP_URL_HOST);
        $path = (string) parse_url($url, PHP_URL_PATH);

        if ($host === '' || $path === '') {
            return null;
        }
        if (! $this->isIdxImagesHost($host)) {
            return null;
        }
        if (preg_match('~^/images/([^/]+)/([^/?#]+)$~', $path, $m) !== 1) {
            return null;
        }

        return $this->publicPhotoUrl(rawurldecode($m[1]), rawurldecode($m[2]));
    }

    private function shouldRewriteHost(string $url): bool
    {
        $host = (string) parse_url($url, PHP_URL_HOST);
        if ($host === '') {
            return str_contains($url, '/listings/') && str_contains($url, '/photos/');
        }

        $host = strtolower($host);
        $allowed = $this->rewriteHosts();

        foreach ($allowed as $h) {
            if ($h !== '' && ($host === $h || str_ends_with($host, '.'.$h))) {
                return true;
            }
        }

        return false;
    }

    private function isCloudFrontHost(string $url): bool
    {
        $host = (string) parse_url($url, PHP_URL_HOST);
        if ($host === '') {
            return false;
        }

        return str_ends_with(strtolower($host), '.cloudfront.net');
    }

    private function isIdxImagesHost(string $host): bool
    {
        $host = strtolower($host);
        $configured = (string) parse_url((string) config('bridge.images_public_base'), PHP_URL_HOST);
        if ($configured !== '' && $host === strtolower($configured)) {
            return true;
        }

        $idxImagesHost = (string) parse_url((string) config('idx_urls.images_public_url'), PHP_URL_HOST);
        if ($idxImagesHost !== '' && $host === strtolower($idxImagesHost)) {
            return true;
        }

        return false;
    }

    /**
     * @return list<string>
     */
    private function rewriteHosts(): array
    {
        $hosts = ['bridgedataoutput.com', 'api.bridgedataoutput.com'];
        $primary = parse_url((string) config('bridge.host'), PHP_URL_HOST);
        if (is_string($primary) && $primary !== '') {
            array_unshift($hosts, strtolower($primary));
        }

        $extra = config('bridge.image_rewrite_hosts');
        if (is_array($extra)) {
            foreach ($extra as $h) {
                if (is_string($h) && $h !== '') {
                    $hosts[] = strtolower($h);
                }
            }
        }

        $sparkExtra = config('spark.image_rewrite_hosts');
        if (is_array($sparkExtra)) {
            foreach ($sparkExtra as $h) {
                if (is_string($h) && $h !== '') {
                    $hosts[] = strtolower($h);
                }
            }
        }

        return array_values(array_unique($hosts));
    }
}
