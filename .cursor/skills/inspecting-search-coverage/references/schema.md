# Schema.org Structured Data

When to use: Validating JSON-LD implementation, rich snippet eligibility, and structured data coverage for real estate listings and IDX pages.

## Patterns

**RESO Property Schema Mapping**
Bridge RESO API responses (`/api/v1/properties`) return `Property` entities with fields like `ListingKey`, `ListPrice`, `BedroomsTotal`, `BathroomsTotalInteger`, `LivingArea`, and `Media`. Map these to Schema.org `RealEstateListing` or `House` types with `offers` (PriceSpecification), `numberOfRooms`, `floorSize`, and `image` properties.

**GIS Geometry Schema**
GeoJSON responses from `/api/v1/gis` contain `Polygon` geometries that can be embedded as Schema.org `GeoShape` or `Place` with `geo` properties. Use the `bbox` metadata to generate `geoMidpoint` and `geoRadius` for neighborhood and radius-search pages.

**Organization and LocalBusiness for GHL**
GHL-integrated locations should implement `RealEstateAgent` or `RealEstateAgency` schema using data from `ghl_oauth_tokens` and `ghl_registered_urls`. Include `areaServed` referencing the MLS market area and `hasOfferCatalog` pointing to IDX subscription tiers.

## Warning

MLS compliance restrictions may limit which structured data properties can be publicly exposed. The `BridgeTeaser` service caps listing data for non-full tokens—ensure that schema markup on teaser pages accurately represents the limited data shown and does not misrepresent the full listing availability to search engines.