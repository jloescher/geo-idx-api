<?php

namespace App\Services\Geocoding;

use Illuminate\Contracts\Support\Arrayable;

final readonly class GeocodingResult implements Arrayable
{
    public function __construct(
        public float $lat,
        public float $lng,
        public string $formattedAddress,
        public string $placeId,
    ) {}

    public function toArray(): array
    {
        return [
            'lat' => $this->lat,
            'lng' => $this->lng,
            'formatted_address' => $this->formattedAddress,
            'place_id' => $this->placeId,
        ];
    }
}
