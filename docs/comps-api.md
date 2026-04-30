# Comps API (`POST /api/v1/comps/run`)

Bridge-backed comparables and investor analysis endpoint for the active MLS dataset resolved by `domain.token` + `MlsDatasetResolver`.

## Auth and access

- Route: `POST /api/v1/comps/run`
- Middleware: `domain.token`
- Token/domain behavior:
  - domain auth or `idx:access` token: A–E comps only (sold comps teaser cap applies)
  - `idx:full` token: full comps payload + investor modes
- Investor modes (`rent_hold_cashflow`, `flip_vs_hold`, `appraiser_simulation`, `bpo`) require `idx:full`.

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

### Modes

- `A`, `B`, `C`, `D`, `E`:
  - sold comps + optional active/pending competition comps
  - adjustment grid + market conditions + optional overpriced signals (full access)
- `rent_hold_cashflow`:
  - Bridge rental comps (`PropertyType eq 'Residential Lease'`) for closed + active/pending inventories
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
  - requires `idx:full`

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

## Implementation notes

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

