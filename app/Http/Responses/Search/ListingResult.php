<?php

namespace App\Http\Responses\Search;

final readonly class ListingResult
{
    /**
     * @param  list<string>|null  $specialListingConditions
     * @param  array<string,string>|null  $primaryImage
     */
    public function __construct(
        public string $listingId,
        public ?string $standardStatus,
        public ?float $listPrice,
        public ?float $closePrice,
        public ?float $originalListPrice,
        public ?float $previousListPrice,
        public ?int $bedroomsTotal,
        public ?float $bathroomsTotal,
        public ?int $livingArea,
        public ?float $lotSizeAcres,
        public ?int $yearBuilt,
        public ?string $streetNumber,
        public ?string $streetName,
        public ?string $city,
        public ?string $state,
        public ?string $postalCode,
        public ?string $county,
        public ?string $fullAddress,
        public ?string $propertyType,
        public ?string $propertySubType,
        public ?array $specialListingConditions,
        public ?bool $waterfrontYn,
        public ?bool $poolPrivateYn,
        public ?bool $dockYn,
        public ?bool $newConstructionYn,
        public ?float $latitude,
        public ?float $longitude,
        public ?string $onMarketDate,
        public ?string $modificationTimestamp,
        public ?int $daysOnMarket,
        public ?string $floodZoneCode,
        public ?array $primaryImage,
        public ?string $listAgentMlsId,
        public ?string $listOfficeMlsId,
        public ?float $distanceMiles,
        public ?int $storiesTotal,
        public ?float $monthlyFees,
        public ?bool $garageYn,
        public ?bool $associationYn,
        public ?bool $spaYn,
        public ?bool $fireplaceYn,
        public ?bool $seniorCommunityYn,
        public ?string $priceChangeTimestamp,
    ) {}

    public function toArray(): array
    {
        $data = get_object_vars($this);

        return array_filter($data, static fn ($v) => $v !== null);
    }
}
