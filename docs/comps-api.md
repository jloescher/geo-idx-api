# Comps API (`POST /api/v1/comps/run`)

MLS comparables and investor analysis for the active feed resolved by `domain.token` + MLS access middleware (`bridge_stellar`, `spark_beaches`, …).

**Data sources:**

- **Closed / sold comps** — live RESO upstream per feed provider (`bridge_*` / `spark_*` catalog codes; Closed rows are not bulk-replicated into the mirror).
- **Active / Pending competition** — PostGIS `listings` mirror for **all** enabled MLS feeds (Stellar, Beaches, etc.).
- **Rental (`rent_hold_cashflow`, `flip_vs_hold`)** — Closed leases from live Bridge/Spark; Active/Pending leases from the mirror.

## Auth and access

- Route: `POST /api/v1/comps/run`
- Middleware: `domain.token`
- Authenticated **domain** or **PAT** traffic (with `idx:access` or `idx:full` and a verified domain binding for tokens) receives **full** comps payloads, including investor modes, BPO, and home value.

## Request shape

Top-level keys:

- `subject`
- `mode`
- `scope`
- optional: `filters`, `keywords`, `aggregation_method`, `rental_params`, `flip_params`, `simulation_params`, `bpo_params`

### Subject

- `subject.type`: `mls` or `off_market`
- `mls`: requires `subject.listing_id`
- `off_market`: requires `subject.lat`, `subject.lng`
- Optional subject enrichments now supported:
  - `garage_spaces`, `carport_spaces`, `covered_spaces`, `open_parking_spaces`, `parking_stalls_total`
  - `subdivision_name`, `mls_area_major`, `view_yn`, `monthly_fees`, `flood_zone_code`
- **Deprecated** (still accepted; merged before validation): `subject.stellar_flood_zone_code` → `flood_zone_code`, `subject.stellar_total_monthly_fees` → `monthly_fees`
- **`subject.type: mls`:** flood zone and monthly fees are resolved from the upstream Property row via `ListingResoFieldResolver` (same rules as the Postgres mirror: Stellar `STELLAR_FloodZoneCode` / `STELLAR_TotalMonthlyFees`; Beaches association-fee frequency math)

### Modes

- `A`, `B`, `C`, `D`, `E`:
  - sold comps + optional active/pending competition comps
  - adjustment grid + market conditions + optional overpriced signals (full access)
- `rent_hold_cashflow`:
  - Rental comps: Closed leases from live Bridge/Spark (`PropertyType eq 'Residential Lease'`); Active/Pending from the `listings` mirror
  - returns `rental_comps` and `rental_result`
- `flip_vs_hold`:
  - combines sold comps + rental comps
  - returns `flip_vs_hold_result` with `flip`, `hold`, and `recommendation`
- `appraiser_simulation`:
  - sold comps simulation
  - returns `simulation_result` (`indicated_value`, BPO band, risk score/band)
- `bpo`:
  - URAR-style Broker Price Opinion using market-derived adjustments
  - OLS regression extracts $/unit rates from the comp dataset (no static values)
  - returns `bpo_result` with full 14-line adjustment grid, reconciliation, and confidence scoring
- `home_value`:
  - Home value estimation from owner-provided details or an active MLS listing
  - Two paths: `address` + geocoding via Google Maps, or `listing_id` + Bridge lookup
  - When `listing_id` is provided, all property details auto-populate from the MLS record
  - Condition rating auto-derived from `PropertyCondition` field or `PublicRemarks` keyword analysis
  - Condition is optional: if not provided and cannot be derived, the condition adjustment is skipped
  - Renovation credits (kitchen, bathrooms, HVAC) derived from market data (scales with local price levels)
  - Expanded `property_type` enum: `sfr`, `townhouse`, `condo`, `manufactured`, `duplex`, `triplex`, `quadplex`, `modular`
  - returns `home_value_result` with estimate, range, confidence, comparable count, and market rates summary

## Key filter extensions

Supported optional filter keys include:

- `match_garage_spaces`, `min_garage_spaces`, `max_garage_spaces`
- `match_view`, `match_subdivision`, `match_mls_area_major`
- existing tolerance and adjustment controls:
  - `living_area_pct`, `beds_tolerance`, `baths_tolerance`, `year_built_tolerance`, `lot_size_pct`
  - `adj_pool_value`, `adj_garage_per_space`, `adj_waterfront_value`, `adj_year_built_per_year`, `adj_lot_per_acre`

## Response highlights

Common response envelope:

- `success`
- `subject`
- `sold_comps`
- `competition_comps`
- `failed_listings`
- `overpriced_signals`
- `market_conditions`
- `metadata`
- `warnings`

Mode-specific additions:

- `rent_hold_cashflow`: `rental_comps`, `rental_result`
- `flip_vs_hold`: `flip_vs_hold_result`, `rental_comps`
- `appraiser_simulation`: `simulation_result`
- `bpo`: `bpo_result`
- `home_value`: `home_value_result`

## Implementation notes

- Comp and subject payloads expose `monthly_fees`, `flood_zone`, and `flood_zone_codes` derived from the same `ListingResoFieldResolver` logic as mirror persist (not only `{DATASET}_TotalMonthlyFees` / `{DATASET}_FloodZoneCode` OData fields).
- Coordinates normalization supports all Bridge shapes used in Stellar:
  - GeoJSON-like: `Coordinates.coordinates`
  - tuple array: `Coordinates: [lng, lat]`
  - fallback: `Latitude` + `Longitude`
- Residential Lease rent normalization uses:
  - `ClosePrice` for `StandardStatus=Closed`
  - `ListPrice` for active/pending rows
  - `LeaseAmountFrequency=Annually` is converted to monthly (`/12`)

### BPO mode (`bpo_params`)

Optional request parameters for BPO mode:

- `bpo_params.max_comps`: max sold comps to include (3-25, default 8)
- `bpo_params.sold_months_back`: lookback window (1-18, default 12)
- `bpo_params.min_comps_for_regression`: minimum comps for OLS regression (3-25, default 6)
- `bpo_params.confidence_threshold`: minimum confidence to return estimate (0-100)
- `bpo_params.exclude_outlier_zscore`: z-score threshold for outlier removal (1-5)

All adjustment rates are derived from the comp dataset at query time:

- **6+ comps**: OLS multiple regression solves `ClosePrice = b0 + b1*LivingArea + b2*BedroomsTotal + b3*BathroomsTotalDecimal + b4*YearBuilt + b5*LotSizeAcres + b6*GarageSpaces + b7*PoolPrivateYN + b8*WaterfrontYN + b9*CloseDateEpoch`
- **3-5 comps**: Paired-sales extraction finds comp pairs differing in only one feature
- **<3 comps**: Median PPSF fallback with `median_only` method and low confidence warning

Renovation credit amounts are derived from the market's `gla_per_sf` and median GLA:

- **Kitchen**: ~2% of typical home value (floors at 50% of $8,000 default)
- **Bathrooms**: ~1.5% of typical home value (floors at 50% of $6,000 default)
- **HVAC**: ~1% of typical home value or `age_per_year × 20` when age regression is strong (floors at 50% of $4,000 default)
- Falls back to static defaults ($8K/$6K/$4K) when insufficient market data
- Credits depreciate: full credit for renovations within 5 years, 50% for 5-10 years, none after 10 years

### Home Value mode (`home_value_params`)

`home_value` mode supports two subject resolution paths:

1. **Address path** (`address` provided) -- geocoded via Google Maps API to resolve lat/lng, uses owner-provided details
2. **Listing ID path** (`listing_id` provided) -- fetches active listing from Bridge, auto-populates all subject fields from the MLS record

When `listing_id` is provided, the following fields are auto-populated from the listing record and become optional in the request: `address`, `bedrooms`, `full_bathrooms`, `living_area_sqft`, `year_built`, `property_type`, `stories`, `pool`, `waterfront`, `garage_spaces`, `subdivision_name`, `mls_area_major`.

**Subject fields for `home_value` mode:**

**Required (address path):**

| Field | Type | Description |
|-------|------|-------------|
| `address` | string | Full street address for geocoding (required unless `listing_id` provided) |
| `property_type` | string | One of: `sfr`, `townhouse`, `condo`, `manufactured`, `duplex`, `triplex`, `quadplex`, `modular` |
| `bedrooms` | integer | Number of bedrooms (1-20) |
| `full_bathrooms` | integer | Number of full bathrooms |
| `living_area_sqft` | integer | Gross living area in square feet |
| `year_built` | integer | Year the property was built |

**Required (all paths):**

| Field | Type | Description |
|-------|------|-------------|
| `half_bathrooms` | integer | Number of half bathrooms (defaults to 0 if null) |

**Optional:**

| Field | Type | Description |
|-------|------|-------------|
| `listing_id` | string | MLS listing ID -- fetches property details from Bridge, alternative to `address` |
| `condition` | string | One of: `poor`, `fair`, `good`, `excellent` (auto-derived from listing when using `listing_id`) |
| `garage_spaces` | integer | Garage parking spaces |
| `pool` | boolean | Has private pool |
| `waterfront` | boolean | Waterfront property |
| `lot_size_sqft` | integer | Lot size in square feet |
| `hoa_monthly_fee` | float | Monthly HOA fee |
| `stories` | integer | Number of stories |
| `roof_year_replaced` | integer | Year roof was replaced |
| `renovated_kitchen_year` | integer | Year kitchen was renovated |
| `renovated_bathrooms_year` | integer | Year bathrooms were renovated |
| `renovated_hvac_year` | integer | Year HVAC was replaced |
| `enclosed_lanai_sqft` | integer | Enclosed lanai area |
| `screen_pool_enclosure` | boolean | Has screened pool enclosure |

**Condition derivation** (when `listing_id` is used and `condition` is not explicitly provided):

1. `PropertyCondition` field from Bridge is checked first. In Stellar MLS, `PropertyCondition` describes **construction state** (not physical quality). Valid values: `Existing`, `New Construction`, `Completed`, `Fixer`, `Proposed`, `Pre-Construction`, `Under Construction`, `Under Renovation`. Only `Fixer` maps to a condition quality rating (`poor`).
2. If `PropertyCondition` doesn't yield a quality rating, `PublicRemarks` text is analyzed for condition keywords:
   - **Poor**: "fixer upper", "needs major work", "tear down", "investor special", "distressed", etc.
   - **Excellent**: "mint condition", "turnkey", "completely renovated", "like new", "no expense spared", etc.
   - **Good**: "well maintained", "move-in ready", "updated", "immaculate", etc.
   - **Fair**: "needs some tlc", "as-is", "needs updating", "handyman special", etc.
3. Returns `null` if no confident match -- condition adjustment is skipped entirely

**PropertySubType mapping:**

The `property_type` enum maps bidirectionally to RESO `PropertySubType` values from the Stellar MLS dataset:

| Widget enum | RESO PropertySubType |
|-------------|---------------------|
| `sfr` | Single Family Residence |
| `townhouse` | Townhouse |
| `condo` | Condominium, Condo - Hotel |
| `manufactured` | Manufactured Home, Manufactured On Land, Mobile Home |
| `duplex` | Duplex, 1/2 Duplex |
| `triplex` | Triplex |
| `quadplex` | Quadruplex |
| `modular` | Modular Home |

Additional Stellar MLS subtypes that map to `sfr`: `Apartment`, `Villa`, `Multi Family (5+)`, `Residential`.

Other Stellar MLS subtypes (non-residential) are not mapped and will return `null`: `Commercial`, `Industrial`, `Retail`, `Office`, `Warehouse`, `Hotel/Motel`, `Mixed Use`, `Business`, `Farm`, `Ranch`, `Agriculture`, `Unimproved Land`, etc.

When using `listing_id`, the listing's `PropertySubType` is reverse-mapped to the widget enum automatically.

Optional request parameters:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `home_value_params.sold_months_back` | integer | 12 | Lookback window for sold comps (1-18) |
| `home_value_params.max_comps` | integer | 25 | Maximum sold comps to retrieve |

Geocoding requires `GOOGLE_MAPS_GEOCODING_API_KEY` in the environment. Results are cached for 30 days.

