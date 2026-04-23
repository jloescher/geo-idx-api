<?php

namespace App\Services\Bridge;

final class BridgeTeaser
{
    /**
     * Revenue impact: teaser limits protect paid MLS inventory from scrapers while
     * still allowing immediate lead capture on the first screen of results.
     */
    public static function apply(string $json, int $limit = 3): string
    {
        try {
            $data = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (\JsonException) {
            return $json;
        }

        if (! is_array($data)) {
            return $json;
        }

        return json_encode(self::limitTopLevelList($data, $limit), JSON_THROW_ON_ERROR);
    }

    public static function countListings(string $json): ?int
    {
        try {
            $data = json_decode($json, true, flags: JSON_THROW_ON_ERROR);
        } catch (\JsonException) {
            return null;
        }

        if (! is_array($data)) {
            return null;
        }

        foreach (['value', 'bundle', 'd', 'listings'] as $key) {
            if (isset($data[$key]) && is_array($data[$key]) && self::isList($data[$key])) {
                return count($data[$key]);
            }
        }

        if (self::isList($data)) {
            return count($data);
        }

        return null;
    }

    /**
     * @param  array<mixed>  $data
     * @return array<mixed>
     */
    private static function limitTopLevelList(array $data, int $limit): array
    {
        foreach (['value', 'bundle', 'd', 'listings'] as $key) {
            if (isset($data[$key]) && is_array($data[$key]) && self::isList($data[$key])) {
                $data[$key] = array_slice($data[$key], 0, $limit);

                return $data;
            }
        }

        if (self::isList($data)) {
            return array_slice($data, 0, $limit);
        }

        return $data;
    }

    /**
     * @param  array<mixed>  $maybeList
     */
    private static function isList(array $maybeList): bool
    {
        if ($maybeList === []) {
            return true;
        }

        return array_keys($maybeList) === range(0, count($maybeList) - 1);
    }
}
