package comps

import "encoding/json"

// RunRequest is the POST /api/v1/comps/run body (docs/comps-api.md).
type RunRequest struct {
	Subject           SubjectInput       `json:"subject"`
	Mode              string             `json:"mode"`
	Scope             ScopeInput         `json:"scope"`
	Filters           FiltersInput       `json:"filters"`
	Keywords          []string           `json:"keywords"`
	AggregationMethod string             `json:"aggregation_method"`
	RentalParams      json.RawMessage    `json:"rental_params"`
	FlipParams        json.RawMessage    `json:"flip_params"`
	SimulationParams  json.RawMessage    `json:"simulation_params"`
	BPOParams         BPOParamsInput     `json:"bpo_params"`
	HomeValueParams   HomeValueParamsIn  `json:"home_value_params"`
}

type SubjectInput struct {
	Type                    string   `json:"type"`
	ListingID               string   `json:"listing_id"`
	Lat                     *float64 `json:"lat"`
	Lng                     *float64 `json:"lng"`
	Bedrooms                *float64 `json:"bedrooms"`
	Bathrooms               *float64 `json:"bathrooms"`
	LivingAreaSqft          *float64 `json:"living_area_sqft"`
	LotSizeSqft             *float64 `json:"lot_size_sqft"`
	GarageSpaces            *float64 `json:"garage_spaces"`
	MonthlyFees             *float64 `json:"monthly_fees"`
	FloodZoneCode           *string  `json:"flood_zone_code"`
	StellarFloodZoneCode    *string  `json:"stellar_flood_zone_code"`
	StellarTotalMonthlyFees *float64 `json:"stellar_total_monthly_fees"`
	SubdivisionName         *string  `json:"subdivision_name"`
	MLSAreaMajor            *string  `json:"mls_area_major"`
	ViewYN                  *bool    `json:"view_yn"`
	Address                 *string  `json:"address"`
	PropertyType            *string  `json:"property_type"`
	YearBuilt               *int     `json:"year_built"`
	Condition               *string  `json:"condition"`
	RenovatedKitchenYear    *int     `json:"renovated_kitchen_year"`
	RenovatedBathroomsYear  *int     `json:"renovated_bathrooms_year"`
	RenovatedHVACYear       *int     `json:"renovated_hvac_year"`
}

type ScopeInput struct {
	Type         string   `json:"type"`
	RadiusMiles  *float64 `json:"radius_miles"`
	PostalCodes  []string `json:"postal_codes"`
}

type FiltersInput struct {
	SoldMonthsBack          *int     `json:"sold_months_back"`
	MaxSoldComps            *int     `json:"max_sold_comps"`
	MaxCompetitionComps     *int     `json:"max_competition_comps"`
	IncludeActivePending    *bool    `json:"include_active_pending"`
	IncludeOverpriced       *bool    `json:"include_overpriced_signals"`
	LivingAreaPct           *float64 `json:"living_area_pct"`
	BedsTolerance           *float64 `json:"beds_tolerance"`
	BathsTolerance          *float64 `json:"baths_tolerance"`
	YearBuiltTolerance      *int     `json:"year_built_tolerance"`
	LotSizePct              *float64 `json:"lot_size_pct"`
	MatchGarageSpaces       *bool    `json:"match_garage_spaces"`
	MinGarageSpaces         *float64 `json:"min_garage_spaces"`
	MaxGarageSpaces         *float64 `json:"max_garage_spaces"`
	MatchView               *bool    `json:"match_view"`
	MatchSubdivision        *bool    `json:"match_subdivision"`
	MatchMLSAreaMajor       *bool    `json:"match_mls_area_major"`
	AdjPoolValue            *float64 `json:"adj_pool_value"`
	AdjGaragePerSpace       *float64 `json:"adj_garage_per_space"`
	AdjWaterfrontValue      *float64 `json:"adj_waterfront_value"`
	AdjYearBuiltPerYear     *float64 `json:"adj_year_built_per_year"`
	AdjLotPerAcre           *float64 `json:"adj_lot_per_acre"`
}

type BPOParamsInput struct {
	MaxComps              *int     `json:"max_comps"`
	SoldMonthsBack        *int     `json:"sold_months_back"`
	MinCompsForRegression *int     `json:"min_comps_for_regression"`
	ConfidenceThreshold   *float64 `json:"confidence_threshold"`
	ExcludeOutlierZScore  *float64 `json:"exclude_outlier_zscore"`
}

type HomeValueParamsIn struct {
	SoldMonthsBack *int `json:"sold_months_back"`
	MaxComps       *int `json:"max_comps"`
}

// SubjectProfile is the resolved subject used for comp selection.
type SubjectProfile struct {
	Type           string  `json:"type"`
	ListingKey     string  `json:"listing_key,omitempty"`
	ListingID      string  `json:"listing_id,omitempty"`
	Lat            float64 `json:"lat"`
	Lng            float64 `json:"lng"`
	Bedrooms       float64 `json:"bedrooms"`
	Bathrooms      float64 `json:"bathrooms"`
	LivingArea     float64 `json:"living_area_sqft"`
	LotSizeAcres   float64 `json:"lot_size_acres"`
	YearBuilt      int     `json:"year_built"`
	GarageSpaces   float64 `json:"garage_spaces"`
	PoolPrivate    bool    `json:"pool_private"`
	Waterfront     bool    `json:"waterfront"`
	ListPrice      float64 `json:"list_price,omitempty"`
	MonthlyFees    float64 `json:"monthly_fees,omitempty"`
	FloodZone      string  `json:"flood_zone,omitempty"`
	Subdivision    string  `json:"subdivision_name,omitempty"`
	MLSAreaMajor   string  `json:"mls_area_major,omitempty"`
	PropertyType   string  `json:"property_type,omitempty"`
	Condition              string `json:"condition,omitempty"`
	RenovatedKitchenYear   int    `json:"renovated_kitchen_year,omitempty"`
	RenovatedBathroomsYear int    `json:"renovated_bathrooms_year,omitempty"`
	RenovatedHVACYear      int    `json:"renovated_hvac_year,omitempty"`
	Raw                    json.RawMessage `json:"-"`
}

// CompRecord is a normalized comparable listing.
type CompRecord struct {
	ListingKey       string          `json:"listing_key"`
	StandardStatus   string          `json:"standard_status"`
	ClosePrice       float64         `json:"close_price,omitempty"`
	ListPrice        float64         `json:"list_price,omitempty"`
	Bedrooms         float64         `json:"bedrooms"`
	Bathrooms        float64         `json:"bathrooms"`
	LivingArea       float64         `json:"living_area_sqft"`
	LotSizeAcres     float64         `json:"lot_size_acres"`
	YearBuilt        int             `json:"year_built"`
	GarageSpaces     float64         `json:"garage_spaces"`
	PoolPrivate      bool            `json:"pool_private"`
	Waterfront       bool            `json:"waterfront"`
	Lat              float64         `json:"lat"`
	Lng              float64         `json:"lng"`
	CloseDate        string          `json:"close_date,omitempty"`
	DistanceMiles    float64         `json:"distance_miles"`
	AdjustedPrice    float64         `json:"adjusted_price,omitempty"`
	Adjustments      []AdjustmentLine `json:"adjustments,omitempty"`
	MonthlyFees      float64         `json:"monthly_fees,omitempty"`
	FloodZone        string          `json:"flood_zone,omitempty"`
	KeywordScore     float64         `json:"keyword_score,omitempty"`
	Property         json.RawMessage `json:"property"`
}

type AdjustmentLine struct {
	Feature string  `json:"feature"`
	Amount  float64 `json:"amount"`
}

// BpoMarketRates holds market-derived $/unit rates from the sold comp set.
type BpoMarketRates struct {
	GLAPerSF         float64  `json:"gla_per_sf"`
	BedPerRoom       float64  `json:"bed_per_room"`
	BathPerFull      float64  `json:"bath_per_full"`
	AgePerYear       float64  `json:"age_per_year"`
	LotPerAcre       float64  `json:"lot_per_acre"`
	GaragePerSpace   float64  `json:"garage_per_space"`
	PoolValue        float64  `json:"pool_value"`
	WaterfrontValue  float64  `json:"waterfront_value"`
	TimePerMonthPct  float64  `json:"time_per_month_pct"`
	Intercept        float64  `json:"intercept"`
	RSquared         float64  `json:"r_squared"`
	Method           string   `json:"method"`
	CompCount        int      `json:"comp_count"`
	MedianGLA        float64  `json:"median_gla"`
	Warnings         []string `json:"warnings,omitempty"`
}

// BpoGridLine is one URAR sales-comparison adjustment line.
type BpoGridLine struct {
	Feature      string  `json:"feature"`
	SubjectValue any     `json:"subject_value,omitempty"`
	CompValue    any     `json:"comp_value,omitempty"`
	Unit         string  `json:"unit,omitempty"`
	RateSource   string  `json:"rate_source"`
	RatePerUnit  float64 `json:"rate_per_unit,omitempty"`
	Adjustment   float64 `json:"adjustment"`
	Reasoning    string  `json:"reasoning,omitempty"`
}

// BpoCompGrid is the per-comp URAR adjustment grid and reconciliation inputs.
type BpoCompGrid struct {
	CompIndex          int           `json:"comp_index"`
	ListingKey         string        `json:"listing_key"`
	SalePrice          float64       `json:"sale_price"`
	SaleDate           string        `json:"sale_date,omitempty"`
	Lines              []BpoGridLine `json:"lines"`
	NetAdjustment      float64       `json:"net_adjustment"`
	GrossAdjustment    float64       `json:"gross_adjustment"`
	GrossAdjustmentPct float64       `json:"gross_adjustment_pct"`
	AdjustedPrice      float64       `json:"adjusted_price"`
	Weight             float64       `json:"weight,omitempty"`
}

// RenovationCredit is a subject-side value add from recent renovations.
type RenovationCredit struct {
	Feature    string  `json:"feature"`
	Year       int     `json:"year,omitempty"`
	Amount     float64 `json:"amount"`
	RateSource string  `json:"rate_source"`
	Reasoning  string  `json:"reasoning,omitempty"`
}

// RunResponse is the comps API envelope.
type RunResponse struct {
	Success            bool                   `json:"success"`
	Subject            SubjectProfile         `json:"subject"`
	SoldComps          []CompRecord           `json:"sold_comps"`
	CompetitionComps   []CompRecord           `json:"competition_comps"`
	FailedListings     []string               `json:"failed_listings"`
	OverpricedSignals  []map[string]any       `json:"overpriced_signals"`
	MarketConditions   map[string]any         `json:"market_conditions"`
	Metadata           map[string]any         `json:"metadata"`
	Warnings           []string               `json:"warnings"`
	RentalComps        []CompRecord           `json:"rental_comps,omitempty"`
	RentalResult       map[string]any         `json:"rental_result,omitempty"`
	FlipVsHoldResult   map[string]any         `json:"flip_vs_hold_result,omitempty"`
	SimulationResult   map[string]any         `json:"simulation_result,omitempty"`
	BPOResult          map[string]any         `json:"bpo_result,omitempty"`
	HomeValueResult    map[string]any         `json:"home_value_result,omitempty"`
}
