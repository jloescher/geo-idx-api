package comps

import "time"

// --- Request types ---

// RunCompsRequest is the JSON body for POST /api/v1/comps/run.
type RunCompsRequest struct {
	Subject           SubjectInput           `json:"subject"`
	Mode              string                 `json:"mode"` // A, B, C, D, E, rent_hold_cashflow, flip_vs_hold
	Scope             ScopeInput             `json:"scope"`
	Filters           FiltersInput           `json:"filters"`
	Keywords          map[string][]Keyword   `json:"keywords"` // "disrepair", "good_condition"
	RentalParams      *RentalParamsInput     `json:"rental_params,omitempty"`
	FlipParams        *FlipParamsInput       `json:"flip_params,omitempty"`
	SimulationParams  *SimulationParamsInput `json:"simulation_params,omitempty"`
	AggregationMethod string                 `json:"aggregation_method,omitempty"` // "median" (default) | "average"
	UserID            int64                  `json:"user_id"`
}

type SubjectInput struct {
	Type            string   `json:"type"` // "mls" | "off_market"
	ListingID       string   `json:"listing_id,omitempty"`
	Lat             *float64 `json:"lat,omitempty"`
	Lng             *float64 `json:"lng,omitempty"`
	Bedrooms        *int     `json:"bedrooms,omitempty"`
	Bathrooms       *int     `json:"bathrooms,omitempty"`
	LivingAreaSqft  *int     `json:"living_area_sqft,omitempty"`
	LotSizeSqft     *int     `json:"lot_size_sqft,omitempty"`
	YearBuilt       *int     `json:"year_built,omitempty"`
	Pool            *bool    `json:"pool,omitempty"`
	HOA             *bool    `json:"hoa,omitempty"`
	SeniorCommunity *bool    `json:"senior_community,omitempty"`
	FloodZoneCode   *string  `json:"flood_zone_code,omitempty"`
	ConditionNotes  *string  `json:"condition_notes,omitempty"`
	AskingPrice     *float64 `json:"asking_price,omitempty"`
	Waterfront      *bool    `json:"waterfront,omitempty"`
	GarageSpaces    *int     `json:"garage_spaces,omitempty"`
}

type ScopeInput struct {
	Type             string   `json:"type"` // radius, polygon, neighborhood, zip
	RadiusMiles      *float64 `json:"radius_miles,omitempty"`
	CenterLat        *float64 `json:"center_lat,omitempty"`
	CenterLng        *float64 `json:"center_lng,omitempty"`
	PolygonGeoJSON   any      `json:"polygon_geojson,omitempty"`
	SubdivisionRefID *int64   `json:"subdivision_ref_id,omitempty"`
	PostalCodes      []string `json:"postal_codes,omitempty"`
}

type FiltersInput struct {
	LivingAreaPct            *int  `json:"living_area_pct,omitempty"`
	BedsTolerance            *int  `json:"beds_tolerance,omitempty"`
	BathsTolerance           *int  `json:"baths_tolerance,omitempty"`
	MatchPool                *bool `json:"match_pool,omitempty"`
	MatchHOA                 *bool `json:"match_hoa,omitempty"`
	MatchSeniorCommunity     *bool `json:"match_senior_community,omitempty"`
	MatchFlood               *bool `json:"match_flood,omitempty"`
	MatchPropertySubType     *bool `json:"match_property_sub_type,omitempty"`
	MatchWaterfront          *bool `json:"match_waterfront,omitempty"`
	YearBuiltTolerance       *int  `json:"year_built_tolerance,omitempty"`
	LotSizePct               *int  `json:"lot_size_pct,omitempty"`
	SoldMonthsBack           *int  `json:"sold_months_back,omitempty"`
	MaxSoldComps             *int  `json:"max_sold_comps,omitempty"`
	MaxCompetitionComps      *int  `json:"max_competition_comps,omitempty"`
	MaxFailedListings        *int  `json:"max_failed_listings,omitempty"`
	IncludeActivePending     *bool `json:"include_active_pending,omitempty"`
	IncludeOverpricedSignals *bool `json:"include_overpriced_signals,omitempty"`
	IncludeFailedListings    *bool `json:"include_failed_listings,omitempty"`
	// Adjustment configuration — configurable dollar amounts for per-comp adjustments.
	AdjPoolValue        *float64 `json:"adj_pool_value,omitempty"`
	AdjGaragePerSpace   *float64 `json:"adj_garage_per_space,omitempty"`
	AdjWaterfrontValue  *float64 `json:"adj_waterfront_value,omitempty"`
	AdjYearBuiltPerYear *float64 `json:"adj_year_built_per_year,omitempty"`
	AdjLotPerAcre       *float64 `json:"adj_lot_per_acre,omitempty"`
	AdjBedroomValue     *float64 `json:"adj_bedroom_value,omitempty"`
}

type Keyword struct {
	Phrase  string  `json:"phrase"`
	Weight  float64 `json:"weight"`
	IsRegex bool    `json:"is_regex"`
}

// --- Filter defaults ---

func (f *FiltersInput) livingAreaPct() int {
	if f.LivingAreaPct != nil {
		return *f.LivingAreaPct
	}
	return 10
}

func (f *FiltersInput) bedsTolerance() int {
	if f.BedsTolerance != nil {
		return *f.BedsTolerance
	}
	return 1
}

func (f *FiltersInput) bathsTolerance() int {
	if f.BathsTolerance != nil {
		return *f.BathsTolerance
	}
	return 1
}

func (f *FiltersInput) matchPool() bool {
	if f.MatchPool != nil {
		return *f.MatchPool
	}
	return true
}

func (f *FiltersInput) matchHOA() bool {
	if f.MatchHOA != nil {
		return *f.MatchHOA
	}
	return true
}

func (f *FiltersInput) matchSeniorCommunity() bool {
	if f.MatchSeniorCommunity != nil {
		return *f.MatchSeniorCommunity
	}
	return true
}

func (f *FiltersInput) matchFlood() bool {
	if f.MatchFlood != nil {
		return *f.MatchFlood
	}
	return true
}

func (f *FiltersInput) matchPropertySubType() bool {
	if f.MatchPropertySubType != nil {
		return *f.MatchPropertySubType
	}
	return true
}

func (f *FiltersInput) matchWaterfront() bool {
	if f.MatchWaterfront != nil {
		return *f.MatchWaterfront
	}
	return true
}

func (f *FiltersInput) yearBuiltTolerance() int {
	if f.YearBuiltTolerance != nil {
		return *f.YearBuiltTolerance
	}
	return 15
}

func (f *FiltersInput) lotSizePct() int {
	if f.LotSizePct != nil {
		return *f.LotSizePct
	}
	return 50
}

func (f *FiltersInput) soldMonthsBack() int {
	if f.SoldMonthsBack != nil {
		return *f.SoldMonthsBack
	}
	return 6
}

func (f *FiltersInput) maxSoldComps() int {
	if f.MaxSoldComps != nil {
		return *f.MaxSoldComps
	}
	return 12
}

func (f *FiltersInput) maxCompetitionComps() int {
	if f.MaxCompetitionComps != nil {
		return *f.MaxCompetitionComps
	}
	return 20
}

func (f *FiltersInput) maxFailedListings() int {
	if f.MaxFailedListings != nil {
		return *f.MaxFailedListings
	}
	return 20
}

func (f *FiltersInput) includeActivePending() bool {
	if f.IncludeActivePending != nil {
		return *f.IncludeActivePending
	}
	return true
}

func (f *FiltersInput) includeOverpricedSignals() bool {
	if f.IncludeOverpricedSignals != nil {
		return *f.IncludeOverpricedSignals
	}
	return true
}

func (f *FiltersInput) includeFailedListings() bool {
	if f.IncludeFailedListings != nil {
		return *f.IncludeFailedListings
	}
	return true
}

func (f *FiltersInput) adjPoolValue() float64 {
	if f.AdjPoolValue != nil {
		return *f.AdjPoolValue
	}
	return 20000
}

func (f *FiltersInput) adjGaragePerSpace() float64 {
	if f.AdjGaragePerSpace != nil {
		return *f.AdjGaragePerSpace
	}
	return 7500
}

func (f *FiltersInput) adjWaterfrontValue() float64 {
	if f.AdjWaterfrontValue != nil {
		return *f.AdjWaterfrontValue
	}
	return 50000
}

func (f *FiltersInput) adjYearBuiltPerYear() float64 {
	if f.AdjYearBuiltPerYear != nil {
		return *f.AdjYearBuiltPerYear
	}
	return 500
}

func (f *FiltersInput) adjLotPerAcre() float64 {
	if f.AdjLotPerAcre != nil {
		return *f.AdjLotPerAcre
	}
	return 25000
}

func (f *FiltersInput) adjBedroomValue() *float64 {
	return f.AdjBedroomValue
}

// --- Simulation input types ---

type SimulationParamsInput struct {
	ContractPrice     *float64 `json:"contract_price,omitempty"`
	ConcessionPercent *float64 `json:"concession_percent,omitempty"` // 0.0-1.0
	MaxComps          *int     `json:"max_comps,omitempty"`          // default 4
}

func (p *SimulationParamsInput) maxComps() int {
	if p != nil && p.MaxComps != nil {
		return *p.MaxComps
	}
	return 4
}

func (p *SimulationParamsInput) concessionPercent() float64 {
	if p != nil && p.ConcessionPercent != nil {
		return *p.ConcessionPercent
	}
	return 0
}

// --- Internal resolved subject ---

type resolvedSubject struct {
	ListingID         string
	Address           string
	Lat               float64
	Lng               float64
	Bedrooms          *int
	Bathrooms         *int
	LivingAreaSqft    *int
	LotSizeSqft       *int
	LotSizeAcres      *float64
	YearBuilt         *int
	Pool              *bool
	HOA               *bool
	SeniorCommunity   *bool
	FloodZone         *bool
	FloodZoneCodes    []string
	Waterfront        *bool
	GarageSpaces      *int
	ListPrice         *float64
	PropertyType      *string
	PropertySubType   *string
	OnMarketDate      *time.Time
	StandardStatus    *string
	OriginalListPrice *float64
}

// --- Response types ---

type RunCompsResponse struct {
	Success           bool               `json:"success"`
	Error             string             `json:"error,omitempty"`
	Subject           *SubjectResponse   `json:"subject,omitempty"`
	SoldComps         []SoldComp         `json:"sold_comps,omitempty"`
	CompetitionComps  []CompetitionComp  `json:"competition_comps,omitempty"`
	FailedListings    []FailedListing    `json:"failed_listings,omitempty"`
	OverpricedSignals []OverpricedSignal `json:"overpriced_signals,omitempty"`
	MarketConditions  *MarketConditions  `json:"market_conditions,omitempty"`
	RentalResult      *RentalResult      `json:"rental_result,omitempty"`
	FlipVsHoldResult  *FlipVsHoldResult  `json:"flip_vs_hold_result,omitempty"`
	SimulationResult  *SimulationResult  `json:"simulation_result,omitempty"`
	Metadata          Metadata           `json:"metadata"`
	Warnings          []string           `json:"warnings,omitempty"`
}

type SubjectResponse struct {
	ListingID       string   `json:"listing_id,omitempty"`
	Address         string   `json:"address,omitempty"`
	Lat             float64  `json:"lat"`
	Lng             float64  `json:"lng"`
	Bedrooms        *int     `json:"bedrooms,omitempty"`
	Bathrooms       *int     `json:"bathrooms,omitempty"`
	LivingAreaSqft  *int     `json:"living_area_sqft,omitempty"`
	LotSizeSqft     *int     `json:"lot_size_sqft,omitempty"`
	YearBuilt       *int     `json:"year_built,omitempty"`
	ListPrice       *float64 `json:"list_price,omitempty"`
	PropertyType    *string  `json:"property_type,omitempty"`
	PropertySubType *string  `json:"property_sub_type,omitempty"`
	Waterfront      *bool    `json:"waterfront,omitempty"`
	GarageSpaces    *int     `json:"garage_spaces,omitempty"`
	SeniorCommunity *bool    `json:"senior_community,omitempty"`
	FloodZoneCodes  []string `json:"flood_zone_codes,omitempty"`
}

type SoldComp struct {
	ListingID          string               `json:"listing_id"`
	Address            string               `json:"address"`
	Lat                float64              `json:"lat"`
	Lng                float64              `json:"lng"`
	SoldPrice          *float64             `json:"sold_price,omitempty"`
	SoldDate           *string              `json:"sold_date,omitempty"`
	Bedrooms           *int                 `json:"bedrooms,omitempty"`
	Bathrooms          *int                 `json:"bathrooms,omitempty"`
	LivingAreaSqft     *int                 `json:"living_area_sqft,omitempty"`
	PPSF               *float64             `json:"ppsf,omitempty"`
	DistanceMiles      float64              `json:"distance_miles"`
	DOM                *int                 `json:"dom,omitempty"`
	SimilarityScore    float64              `json:"similarity_score"`
	DisrepairScore     float64              `json:"disrepair_score"`
	GoodConditionScore float64              `json:"good_condition_score"`
	KeywordMatches     map[string][]string  `json:"keyword_matches"`
	YearBuilt          *int                 `json:"year_built,omitempty"`
	LotSizeAcres       *float64             `json:"lot_size_acres,omitempty"`
	Waterfront         *bool                `json:"waterfront,omitempty"`
	GarageSpaces       *int                 `json:"garage_spaces,omitempty"`
	Stories            *int                 `json:"stories,omitempty"`
	BathroomsFull      *int                 `json:"bathrooms_full,omitempty"`
	BathroomsHalf      *int                 `json:"bathrooms_half,omitempty"`
	Pool               *bool                `json:"pool,omitempty"`
	PropertyType       *string              `json:"property_type,omitempty"`
	PropertySubType    *string              `json:"property_sub_type,omitempty"`
	Adjustments        *AdjustmentGrid      `json:"adjustments,omitempty"`
	FloodZone          *FloodZoneAnnotation `json:"flood_zone,omitempty"`
}

type CompetitionComp struct {
	ListingID       string               `json:"listing_id"`
	Address         string               `json:"address"`
	Lat             float64              `json:"lat"`
	Lng             float64              `json:"lng"`
	Status          string               `json:"status"`
	ListPrice       *float64             `json:"list_price,omitempty"`
	PPSF            *float64             `json:"ppsf,omitempty"`
	DistanceMiles   float64              `json:"distance_miles"`
	DOM             *int                 `json:"dom,omitempty"`
	Bedrooms        *int                 `json:"bedrooms,omitempty"`
	Bathrooms       *int                 `json:"bathrooms,omitempty"`
	LivingAreaSqft  *int                 `json:"living_area_sqft,omitempty"`
	SimilarityScore float64              `json:"similarity_score"`
	YearBuilt       *int                 `json:"year_built,omitempty"`
	LotSizeAcres    *float64             `json:"lot_size_acres,omitempty"`
	Waterfront      *bool                `json:"waterfront,omitempty"`
	GarageSpaces    *int                 `json:"garage_spaces,omitempty"`
	Stories         *int                 `json:"stories,omitempty"`
	BathroomsFull   *int                 `json:"bathrooms_full,omitempty"`
	BathroomsHalf   *int                 `json:"bathrooms_half,omitempty"`
	Pool            *bool                `json:"pool,omitempty"`
	PropertyType    *string              `json:"property_type,omitempty"`
	PropertySubType *string              `json:"property_sub_type,omitempty"`
	FloodZone       *FloodZoneAnnotation `json:"flood_zone,omitempty"`
}

type FailedListing struct {
	ListingID         string               `json:"listing_id"`
	Address           string               `json:"address"`
	StandardStatus    string               `json:"standard_status"`
	MLSStatus         string               `json:"mls_status"`
	LastListPrice     *float64             `json:"last_list_price,omitempty"`
	OriginalListPrice *float64             `json:"original_list_price,omitempty"`
	Bedrooms          *int                 `json:"bedrooms,omitempty"`
	Bathrooms         *int                 `json:"bathrooms,omitempty"`
	LivingAreaSqft    *int                 `json:"living_area_sqft,omitempty"`
	PPSF              *float64             `json:"ppsf,omitempty"`
	DistanceMiles     float64              `json:"distance_miles"`
	DOM               *int                 `json:"dom,omitempty"`
	OnMarketDate      *string              `json:"on_market_date,omitempty"`
	BecameInactiveAt  *string              `json:"became_inactive_at,omitempty"`
	PriceReductions   int                  `json:"price_reductions"`
	SimilarityScore   float64              `json:"similarity_score"`
	YearBuilt         *int                 `json:"year_built,omitempty"`
	LotSizeAcres      *float64             `json:"lot_size_acres,omitempty"`
	Waterfront        *bool                `json:"waterfront,omitempty"`
	GarageSpaces      *int                 `json:"garage_spaces,omitempty"`
	Stories           *int                 `json:"stories,omitempty"`
	BathroomsFull     *int                 `json:"bathrooms_full,omitempty"`
	BathroomsHalf     *int                 `json:"bathrooms_half,omitempty"`
	Pool              *bool                `json:"pool,omitempty"`
	PropertyType      *string              `json:"property_type,omitempty"`
	PropertySubType   *string              `json:"property_sub_type,omitempty"`
	FloodZone         *FloodZoneAnnotation `json:"flood_zone,omitempty"`
}

// --- Adjustment types (appraisal-style per-comp dollar adjustments) ---

type AdjustmentLine struct {
	Feature    string  `json:"feature"`       // "gla", "pool", "garage", "waterfront", "year_built", "lot_size"
	SubjectVal any     `json:"subject_value"` // subject's value for this feature
	CompVal    any     `json:"comp_value"`    // comp's value for this feature
	Adjustment float64 `json:"adjustment"`    // positive = adds value to comp
	Reasoning  string  `json:"reasoning"`     // human-readable explanation of the adjustment
}

type AdjustmentGrid struct {
	Lines          []AdjustmentLine `json:"lines"`
	NetAdjustment  float64          `json:"net_adjustment"`
	AdjustedPrice  float64          `json:"adjusted_price"`          // sold_price + net_adjustment
	GrossAdjPct    float64          `json:"gross_adjustment_pct"`    // abs_sum / sold_price * 100
	HighAdjWarning bool             `json:"high_adjustment_warning"` // true if gross > 25%
}

// --- Market condition summary ---

type MarketConditions struct {
	MedianSoldPPSF     *float64 `json:"median_sold_ppsf,omitempty"`
	MedianSoldDOM      *int     `json:"median_sold_dom,omitempty"`
	AvgListToSaleRatio *float64 `json:"avg_list_to_sale_ratio,omitempty"`
	MonthsOfInventory  *float64 `json:"months_of_inventory,omitempty"`
}

type OverpricedSignal struct {
	ListingID string `json:"listing_id"`
	Indicator string `json:"indicator"`
	Detail    string `json:"detail"`
	Severity  string `json:"severity"` // "low", "moderate", "high"
}

type Metadata struct {
	TotalSoldCandidates         int     `json:"total_sold_candidates"`
	TotalCompetitionCandidates  int     `json:"total_competition_candidates"`
	TotalFailedCandidates       int     `json:"total_failed_candidates"`
	ScopeApplied                string  `json:"scope_applied"`
	RadiusMiles                 float64 `json:"radius_miles,omitempty"`
	ProcessingMs                int64   `json:"processing_ms"`
	MlgCanViewFiltered          int     `json:"mlg_can_view_filtered"`
	OverpricedAvailable         bool    `json:"overpriced_available"`
	OverpricedUnavailableReason *string `json:"overpriced_unavailable_reason"`
	FailedListingsAvailable     bool    `json:"failed_listings_available"`
	AggregationMethod           string  `json:"aggregation_method,omitempty"`
}

// --- Internal row types for scanning SQL results ---

type compRow struct {
	ListingID         *string
	StandardStatus    *string
	MLSStatus         *string
	StreetNumber      *string
	StreetDirPrefix   *string
	StreetName        *string
	StreetSuffix      *string
	StreetDirSuffix   *string
	UnitNumber        *string
	City              *string
	State             *string
	PostalCode        *string
	Latitude          *float64
	Longitude         *float64
	ListPrice         *float64
	ClosePrice        *float64
	CloseDate         *time.Time
	OriginalListPrice *float64
	PreviousListPrice *float64
	LivingArea        *int
	LotSizeAcres      *float64
	BedroomsTotal     *int
	BathroomsTotal    *int
	YearBuilt         *int
	PoolPrivateYn     *bool
	AssociationYn     *bool
	SeniorCommunityYn *bool
	LowRiskFloodYn    *bool
	FloodZoneCodes    []string
	WaterfrontYn      *bool
	GarageSpaces      *int
	PropertyType      *string
	PropertySubType   *string
	StoriesTotal      *int
	BathroomsFull     *int
	BathroomsHalf     *int
	OnMarketDate      *time.Time
	BecameInactiveAt  *time.Time
	PublicRemarks     *string
	MfrLeasePrice     *float64
	DaysOnMarket      *int
	DistanceMeters    float64
	SimilarityScore   float64
	TotalCandidates   int
}

// --- Rental input types ---

type RentalParamsInput struct {
	Financing                 FinancingInput       `json:"financing"`
	Ownership                 OwnershipInput       `json:"ownership"`
	TargetHoldPeriodMonths    *int                 `json:"target_hold_period_months,omitempty"`
	OccupancyRate             *float64             `json:"occupancy_rate,omitempty"`
	PropertyManagementPercent *float64             `json:"property_management_percent,omitempty"`
	MaintenancePercent        *float64             `json:"maintenance_percent_of_rent,omitempty"`
	CapexPercent              *float64             `json:"capex_percent_of_rent,omitempty"`
	VacancyPercent            *float64             `json:"vacancy_percent,omitempty"`
	AllowCrossFamily          *bool                `json:"allow_cross_family,omitempty"`
	RentWeighting             *RentWeightingConfig `json:"rent_weighting,omitempty"`
	DSCRConfig                *DSCRConfig          `json:"dscr_config,omitempty"`
}

func (p *RentalParamsInput) targetHoldPeriodMonths() int {
	if p.TargetHoldPeriodMonths != nil {
		return *p.TargetHoldPeriodMonths
	}
	return 12
}

func (p *RentalParamsInput) occupancyRate() float64 {
	if p.OccupancyRate != nil {
		return *p.OccupancyRate
	}
	return 0.95
}

func (p *RentalParamsInput) propertyManagementPercent() float64 {
	if p.PropertyManagementPercent != nil {
		return *p.PropertyManagementPercent
	}
	return 0.08
}

func (p *RentalParamsInput) maintenancePercent() float64 {
	if p.MaintenancePercent != nil {
		return *p.MaintenancePercent
	}
	return 0.05
}

func (p *RentalParamsInput) capexPercent() float64 {
	if p.CapexPercent != nil {
		return *p.CapexPercent
	}
	return 0.05
}

func (p *RentalParamsInput) vacancyPercent() float64 {
	if p.VacancyPercent != nil {
		return *p.VacancyPercent
	}
	return 1.0 - p.occupancyRate()
}

func (p *RentalParamsInput) allowCrossFamily() bool {
	if p.AllowCrossFamily != nil {
		return *p.AllowCrossFamily
	}
	return false
}

type FinancingInput struct {
	PurchasePrice      float64  `json:"purchase_price"`
	DownPaymentAmount  *float64 `json:"down_payment_amount,omitempty"`
	DownPaymentPercent *float64 `json:"down_payment_percent,omitempty"`
	LoanTermYears      int      `json:"loan_term_years"`
	InterestRate       float64  `json:"interest_rate"`
	PMIMonthly         *float64 `json:"pmi_monthly,omitempty"`
	PMIRateAnnual      *float64 `json:"pmi_rate_annual,omitempty"`
	ClosingCosts       *float64 `json:"closing_costs,omitempty"`
}

func (f *FinancingInput) pmiRateAnnual() float64 {
	if f.PMIRateAnnual != nil {
		return *f.PMIRateAnnual
	}
	return 0.0075
}

func (f *FinancingInput) closingCosts() float64 {
	if f.ClosingCosts != nil {
		return *f.ClosingCosts
	}
	return 0
}

type OwnershipInput struct {
	AnnualPropertyTaxes       float64  `json:"annual_property_taxes"`
	AnnualHomeownersInsurance float64  `json:"annual_homeowners_insurance"`
	FloodInsuranceAnnual      *float64 `json:"flood_insurance_annual,omitempty"`
	MonthlyHOA                *float64 `json:"monthly_hoa,omitempty"`
	UtilitiesMonthly          *float64 `json:"utilities_monthly_paid_by_owner,omitempty"`
	OtherMonthly              *float64 `json:"other_monthly,omitempty"`
}

func (o *OwnershipInput) floodInsuranceAnnual() float64 {
	if o.FloodInsuranceAnnual != nil {
		return *o.FloodInsuranceAnnual
	}
	return 0
}

func (o *OwnershipInput) monthlyHOA() float64 {
	if o.MonthlyHOA != nil {
		return *o.MonthlyHOA
	}
	return 0
}

func (o *OwnershipInput) utilitiesMonthly() float64 {
	if o.UtilitiesMonthly != nil {
		return *o.UtilitiesMonthly
	}
	return 0
}

func (o *OwnershipInput) otherMonthly() float64 {
	if o.OtherMonthly != nil {
		return *o.OtherMonthly
	}
	return 0
}

// --- Rental response types ---

type RentalResult struct {
	SubjectSummary    SubjectResponse        `json:"subject_summary"`
	RentEstimate      RentEstimate           `json:"rent_estimate"`
	LoanSummary       LoanSummary            `json:"loan_summary"`
	Monthly           MonthlyBreakdown       `json:"monthly_breakdown"`
	Annual            AnnualMetrics          `json:"annual_metrics"`
	Scenarios         ScenarioOutput         `json:"scenarios"`
	ScenarioFlags     ScenarioFlags          `json:"scenario_flags"`
	Warnings          []string               `json:"warnings"`
	RentalComps       []RentalComp           `json:"rental_comps"`
	Metadata          RentalMetadata         `json:"metadata"`
	InvestorMonthly   *MonthlyBreakdown      `json:"monthly_breakdown_investor,omitempty"`
	QualifyingMonthly *MonthlyBreakdown      `json:"monthly_breakdown_qualifying,omitempty"`
	DSCROverlay       *DSCROverlay           `json:"dscr_overlay,omitempty"`
	SelfSufficiency   *SelfSufficiencyResult `json:"self_sufficiency,omitempty"`
}

type RentEstimate struct {
	Recommended          float64  `json:"recommended"`
	Low                  float64  `json:"low"`
	High                 float64  `json:"high"`
	ActiveMedian         *float64 `json:"active_median,omitempty"`
	CompCount            int      `json:"comp_count"`
	ActiveCompCount      int      `json:"active_comp_count"`
	WeightedMean         float64  `json:"weighted_mean"`
	WeightedMedian       float64  `json:"weighted_median"`
	MedianBlendRatio     float64  `json:"median_blend_ratio"`
	ClosedEffectiveShare float64  `json:"closed_effective_share"`
	WinsorizedCount      int      `json:"winsorized_count"`
	ActiveCompMedian     *float64 `json:"active_competition_median_rent,omitempty"`
	MethodVersion        string   `json:"method_version"`
}

type RentalComp struct {
	ListingID       string               `json:"listing_id"`
	Address         string               `json:"address"`
	Lat             float64              `json:"lat"`
	Lng             float64              `json:"lng"`
	PropertySubType *string              `json:"property_sub_type,omitempty"`
	StatusLabel     string               `json:"status_label"`
	MatchQuality    string               `json:"match_quality"`
	SimilarityScore float64              `json:"similarity_score"`
	Rent            *float64             `json:"rent,omitempty"`
	RentSource      string               `json:"rent_source"`
	Sqft            *int                 `json:"sqft,omitempty"`
	Bedrooms        *int                 `json:"bedrooms,omitempty"`
	Bathrooms       *int                 `json:"bathrooms,omitempty"`
	YearBuilt       *int                 `json:"year_built,omitempty"`
	DistanceMiles   float64              `json:"distance_miles"`
	DOM             *int                 `json:"dom,omitempty"`
	CloseDate       *string              `json:"close_date,omitempty"`
	Pool            *bool                `json:"pool,omitempty"`
	Waterfront      *bool                `json:"waterfront,omitempty"`
	WeightBreakdown *CompWeightBreakdown `json:"weight_breakdown,omitempty"`
	FloodZone       *FloodZoneAnnotation `json:"flood_zone,omitempty"`
}

type LoanSummary struct {
	PurchasePrice  float64 `json:"purchase_price"`
	DownPayment    float64 `json:"down_payment"`
	DownPaymentPct float64 `json:"down_payment_pct"`
	LoanAmount     float64 `json:"loan_amount"`
	InterestRate   float64 `json:"interest_rate"`
	LoanTermYears  int     `json:"loan_term_years"`
	MonthlyPI      float64 `json:"monthly_pi"`
	MonthlyPMI     float64 `json:"monthly_pmi"`
	ClosingCosts   float64 `json:"closing_costs"`
	CashInvested   float64 `json:"cash_invested"`
}

type MonthlyBreakdown struct {
	GrossRent          float64 `json:"gross_rent"`
	Vacancy            float64 `json:"vacancy"`
	EGI                float64 `json:"effective_gross_income"`
	PropertyManagement float64 `json:"property_management"`
	Maintenance        float64 `json:"maintenance"`
	CapEx              float64 `json:"capex"`
	HOA                float64 `json:"hoa"`
	Taxes              float64 `json:"taxes"`
	Insurance          float64 `json:"insurance"`
	FloodInsurance     float64 `json:"flood_insurance"`
	Utilities          float64 `json:"utilities"`
	Other              float64 `json:"other"`
	TotalOperating     float64 `json:"total_operating"`
	NOI                float64 `json:"noi"`
	DebtService        float64 `json:"debt_service"`
	CashFlow           float64 `json:"cash_flow"`
}

type AnnualMetrics struct {
	AnnualNOI      float64 `json:"annual_noi"`
	AnnualCashFlow float64 `json:"annual_cash_flow"`
	CashOnCash     float64 `json:"cash_on_cash"`
	DSCR           float64 `json:"dscr"`
	CapRate        float64 `json:"cap_rate"`
}

type ScenarioOutput struct {
	Conservative ScenarioResult `json:"conservative"`
	Base         ScenarioResult `json:"base"`
	Upside       ScenarioResult `json:"upside"`
}

type ScenarioResult struct {
	MonthlyRent     float64 `json:"monthly_rent"`
	MonthlyCashFlow float64 `json:"monthly_cash_flow"`
	AnnualCashFlow  float64 `json:"annual_cash_flow"`
	CashOnCash      float64 `json:"cash_on_cash"`
	DSCR            float64 `json:"dscr"`
}

type ScenarioFlags struct {
	CashFlowPositive             bool `json:"cashflow_positive"`
	DSCRPass                     bool `json:"dscr_pass"`
	ConservativeCashFlowPositive bool `json:"conservative_cashflow_positive"`
}

type RentalMetadata struct {
	TotalClosedCandidates int     `json:"total_closed_candidates"`
	TotalActiveCandidates int     `json:"total_active_candidates"`
	ScopeApplied          string  `json:"scope_applied"`
	RadiusMiles           float64 `json:"radius_miles,omitempty"`
	ProcessingMs          int64   `json:"processing_ms"`
	FallbackUsed          string  `json:"fallback_used,omitempty"`
}

// rankedRentalComp is an internal type for Go-side re-ranking of rental comps.
type rankedRentalComp struct {
	row            compRow
	matchQuality   string  // "exact_subtype", "family_match", "cross_family_low_confidence"
	subTypeFactor  float64 // 1.0, 0.6, 0.2
	recencyFactor  float64 // 0.0-1.0
	finalScore     float64 // composite of SQL + subtype + recency
	rent           *float64
	rentSource     string // "close_price" or "mfr_lease_price"
	distanceMiles  float64
	isClosedLeased bool
	kernelSim      float64
	distanceDecay  float64
	recencyDecay   float64
	statusMult     float64
	rawWeight      float64
	normWeight     float64
	rentPerSqft    float64
}

// --- Rent weighting config ---

// RentWeightingConfig tunes the kernel-based rent estimation.
type RentWeightingConfig struct {
	ClosedWeightMultiplier *float64 `json:"closed_weight_multiplier,omitempty"`
	ActiveWeightMultiplier *float64 `json:"active_weight_multiplier,omitempty"`
	DistanceDecayMiles     *float64 `json:"distance_decay_miles,omitempty"`
	RecencyHalfLifeDays    *int     `json:"recency_half_life_days,omitempty"`
	WinsorizePercent       *float64 `json:"winsorize_percent,omitempty"`
	MinWeightFloor         *float64 `json:"min_weight_floor,omitempty"`
	MedianBlend            *float64 `json:"median_blend,omitempty"`
	PreferClosedMinShare   *float64 `json:"prefer_closed_min_share,omitempty"`
	RangeLowPct            *float64 `json:"range_low_pct,omitempty"`
	RangeHighPct           *float64 `json:"range_high_pct,omitempty"`
}

func (c *RentWeightingConfig) closedWeightMultiplier() float64 {
	if c != nil && c.ClosedWeightMultiplier != nil {
		return *c.ClosedWeightMultiplier
	}
	return 1.25
}

func (c *RentWeightingConfig) activeWeightMultiplier() float64 {
	if c != nil && c.ActiveWeightMultiplier != nil {
		return *c.ActiveWeightMultiplier
	}
	return 0.85
}

func (c *RentWeightingConfig) distanceDecayMiles() float64 {
	if c != nil && c.DistanceDecayMiles != nil {
		return *c.DistanceDecayMiles
	}
	return 0.75
}

func (c *RentWeightingConfig) recencyHalfLifeDays() int {
	if c != nil && c.RecencyHalfLifeDays != nil {
		return *c.RecencyHalfLifeDays
	}
	return 90
}

func (c *RentWeightingConfig) winsorizePercent() float64 {
	if c != nil && c.WinsorizePercent != nil {
		return *c.WinsorizePercent
	}
	return 0.10
}

func (c *RentWeightingConfig) minWeightFloor() float64 {
	if c != nil && c.MinWeightFloor != nil {
		return *c.MinWeightFloor
	}
	return 0.02
}

func (c *RentWeightingConfig) medianBlend() float64 {
	if c != nil && c.MedianBlend != nil {
		return *c.MedianBlend
	}
	return 0.60
}

func (c *RentWeightingConfig) preferClosedMinShare() float64 {
	if c != nil && c.PreferClosedMinShare != nil {
		return *c.PreferClosedMinShare
	}
	return 0.65
}

func (c *RentWeightingConfig) rangeLowPct() float64 {
	if c != nil && c.RangeLowPct != nil {
		return *c.RangeLowPct
	}
	return 25
}

func (c *RentWeightingConfig) rangeHighPct() float64 {
	if c != nil && c.RangeHighPct != nil {
		return *c.RangeHighPct
	}
	return 75
}

// --- DSCR config ---

type DSCRConfig struct {
	MinThreshold          *float64 `json:"min_threshold,omitempty"`
	StressRateBPS         *int     `json:"stress_rate_bps,omitempty"`
	IOOptionEnabled       *bool    `json:"io_option_enabled,omitempty"`
	QualifyUses           *string  `json:"qualify_uses,omitempty"`
	IncludeTaxesInsurance *bool    `json:"include_taxes_insurance,omitempty"`
	IncludeHOA            *bool    `json:"include_hoa,omitempty"`
	RentQualifyingPercent *float64 `json:"rent_qualifying_percent,omitempty"`
	SelfSufficiencyRule   *string  `json:"self_sufficiency_rule,omitempty"`
	SelfSufficiencyRatio  *float64 `json:"self_sufficiency_ratio,omitempty"`
}

func (c *DSCRConfig) minThreshold() float64 {
	if c != nil && c.MinThreshold != nil {
		return *c.MinThreshold
	}
	return 1.10
}

func (c *DSCRConfig) stressRateBPS() int {
	if c != nil && c.StressRateBPS != nil {
		return *c.StressRateBPS
	}
	return 0
}

func (c *DSCRConfig) ioOptionEnabled() bool {
	if c != nil && c.IOOptionEnabled != nil {
		return *c.IOOptionEnabled
	}
	return false
}

func (c *DSCRConfig) qualifyUses() string {
	if c != nil && c.QualifyUses != nil {
		return *c.QualifyUses
	}
	return "qualifying_view"
}

func (c *DSCRConfig) includeTaxesInsurance() bool {
	if c != nil && c.IncludeTaxesInsurance != nil {
		return *c.IncludeTaxesInsurance
	}
	return true
}

func (c *DSCRConfig) includeHOA() bool {
	if c != nil && c.IncludeHOA != nil {
		return *c.IncludeHOA
	}
	return true
}

func (c *DSCRConfig) rentQualifyingPercent() float64 {
	if c != nil && c.RentQualifyingPercent != nil {
		return *c.RentQualifyingPercent
	}
	return 0.75
}

func (c *DSCRConfig) selfSufficiencyRule() string {
	if c != nil && c.SelfSufficiencyRule != nil {
		return *c.SelfSufficiencyRule
	}
	return "qual_income_gte_pitia"
}

func (c *DSCRConfig) selfSufficiencyRatio() float64 {
	if c != nil && c.SelfSufficiencyRatio != nil {
		return *c.SelfSufficiencyRatio
	}
	return 1.0
}

// --- Flip input types ---

type FlipParamsInput struct {
	RepairBudget            float64           `json:"repair_budget"`
	RehabContingencyPercent *float64          `json:"rehab_contingency_percent,omitempty"`
	HoldingPeriodMonths     *int              `json:"holding_period_months,omitempty"`
	FlipFinancing           *FlipFinancing    `json:"flip_financing,omitempty"`
	SellingCosts            *FlipSellingCosts `json:"selling_costs,omitempty"`
	BuyCosts                *FlipBuyCosts     `json:"buy_costs,omitempty"`
	DesiredProfitAmount     *float64          `json:"desired_profit_amount,omitempty"`
	DesiredProfitPercentARV *float64          `json:"desired_profit_percent_of_arv,omitempty"`
}

func (f *FlipParamsInput) rehabContingencyPercent() float64 {
	if f.RehabContingencyPercent != nil {
		return *f.RehabContingencyPercent
	}
	return 0.10
}

func (f *FlipParamsInput) holdingPeriodMonths() int {
	if f.HoldingPeriodMonths != nil {
		return *f.HoldingPeriodMonths
	}
	return 6
}

type FlipFinancing struct {
	InterestRate  float64  `json:"interest_rate"`
	PointsPercent *float64 `json:"points_percent,omitempty"`
	InterestOnly  *bool    `json:"interest_only,omitempty"`
}

func (f *FlipFinancing) pointsPercent() float64 {
	if f != nil && f.PointsPercent != nil {
		return *f.PointsPercent
	}
	return 0
}

func (f *FlipFinancing) interestOnly() bool {
	if f != nil && f.InterestOnly != nil {
		return *f.InterestOnly
	}
	return true
}

type FlipSellingCosts struct {
	AgentCommissionPercent  *float64 `json:"agent_commission_percent,omitempty"`
	ClosingCostsPercent     *float64 `json:"closing_costs_percent,omitempty"`
	SellerConcessionsAmount *float64 `json:"seller_concessions_amount,omitempty"`
}

func (s *FlipSellingCosts) agentCommissionPercent() float64 {
	if s != nil && s.AgentCommissionPercent != nil {
		return *s.AgentCommissionPercent
	}
	return 0.05
}

func (s *FlipSellingCosts) closingCostsPercent() float64 {
	if s != nil && s.ClosingCostsPercent != nil {
		return *s.ClosingCostsPercent
	}
	return 0.02
}

func (s *FlipSellingCosts) sellerConcessionsAmount() float64 {
	if s != nil && s.SellerConcessionsAmount != nil {
		return *s.SellerConcessionsAmount
	}
	return 0
}

type FlipBuyCosts struct {
	ClosingCostsPercent *float64 `json:"closing_costs_percent,omitempty"`
}

func (b *FlipBuyCosts) closingCostsPercent() float64 {
	if b != nil && b.ClosingCostsPercent != nil {
		return *b.ClosingCostsPercent
	}
	return 0.02
}

// --- New response types ---

// CompWeightBreakdown provides per-comp weight explainability.
type CompWeightBreakdown struct {
	RentPerSqft      float64 `json:"rent_per_sqft"`
	KernelSimilarity float64 `json:"kernel_similarity"`
	DistanceDecay    float64 `json:"distance_decay"`
	RecencyDecay     float64 `json:"recency_decay"`
	StatusMultiplier float64 `json:"status_multiplier"`
	RawWeight        float64 `json:"raw_weight"`
	NormalizedWeight float64 `json:"normalized_weight"`
}

// DSCROverlay holds DSCR lender qualification results.
type DSCROverlay struct {
	DSCRValue          float64  `json:"dscr_value"`
	DSCRThreshold      float64  `json:"dscr_threshold"`
	Pass               bool     `json:"pass"`
	QualifyViewUsed    string   `json:"qualify_view_used"`
	DebtServiceUsed    string   `json:"debt_service_used"`
	DebtServiceMonthly float64  `json:"debt_service_monthly"`
	StressedRate       *float64 `json:"stressed_interest_rate,omitempty"`
	StressedDSCR       *float64 `json:"stressed_dscr,omitempty"`
	Notes              []string `json:"notes"`
}

// SelfSufficiencyResult holds the self-sufficiency test outcome.
type SelfSufficiencyResult struct {
	RentQualifyingPercent   float64 `json:"rent_qualifying_percent"`
	QualifyingIncomeMonthly float64 `json:"qualifying_income_monthly"`
	PITIAMonthly            float64 `json:"pitia_monthly"`
	SurplusDeficitMonthly   float64 `json:"surplus_deficit_monthly"`
	Pass                    bool    `json:"pass"`
	Rule                    string  `json:"rule"`
}

// FlipSummary holds the flip analysis results.
type FlipSummary struct {
	ARV             float64             `json:"arv"`
	ARVMethod       string              `json:"arv_method"`
	ARVCompCount    int                 `json:"arv_comp_count"`
	PurchasePrice   float64             `json:"purchase_price"`
	TotalRepairs    float64             `json:"total_repairs"`
	BuyClosingCosts float64             `json:"buy_closing_costs"`
	CarryingCosts   float64             `json:"carrying_costs"`
	SellingCosts    float64             `json:"selling_costs"`
	TotalCostBasis  float64             `json:"total_cost_basis"`
	SalePrice       float64             `json:"sale_price"`
	NetProfit       float64             `json:"net_profit"`
	CashInvested    float64             `json:"cash_invested"`
	ROI             float64             `json:"roi"`
	AnnualizedROI   float64             `json:"annualized_roi"`
	MaxOfferPrice   float64             `json:"max_offer_price_for_target_profit"`
	MarginOfSafety  float64             `json:"margin_of_safety_percent"`
	MonthlyCarrying FlipMonthlyCarrying `json:"monthly_carrying"`
	KeyAssumptions  []string            `json:"key_assumptions"`
}

type FlipMonthlyCarrying struct {
	Taxes     float64 `json:"taxes"`
	Insurance float64 `json:"insurance"`
	HOA       float64 `json:"hoa"`
	Utilities float64 `json:"utilities"`
	Financing float64 `json:"financing"`
	Total     float64 `json:"total"`
}

// HoldSummary holds the hold analysis results for flip_vs_hold comparison.
type HoldSummary struct {
	RecommendedRent           float64 `json:"recommended_rent"`
	MonthlyCashFlowInvestor   float64 `json:"monthly_cashflow_investor"`
	MonthlyCashFlowQualifying float64 `json:"monthly_cashflow_qualifying"`
	AnnualCashFlow            float64 `json:"annual_cash_flow"`
	CashOnCash                float64 `json:"cash_on_cash"`
	DSCR                      float64 `json:"dscr"`
	CapRate                   float64 `json:"cap_rate"`
	CashInvested              float64 `json:"cash_invested"`
	SelfSufficient            bool    `json:"self_sufficient"`
	RentalCompCount           int     `json:"rental_comp_count"`
}

// FlipVsHoldComparison holds side-by-side boolean flags (no opinions).
type FlipVsHoldComparison struct {
	FlipProfitable         bool     `json:"flip_profitable"`
	HoldCashFlowPositive   bool     `json:"hold_cashflow_positive"`
	FlipROIAboveThreshold  bool     `json:"flip_roi_above_threshold"`
	HoldDSCRPass           bool     `json:"hold_dscr_pass"`
	FlipHigherShortTermROI bool     `json:"flip_higher_short_term_roi"`
	HoldSelfSufficient     bool     `json:"hold_self_sufficient"`
	BothViable             bool     `json:"both_viable"`
	NeitherViable          bool     `json:"neither_viable"`
	Explanation            []string `json:"explanation"`
}

// FlipVsHoldResult is the top-level flip_vs_hold response.
type FlipVsHoldResult struct {
	SubjectSummary SubjectResponse      `json:"subject_summary"`
	FlipSummary    FlipSummary          `json:"flip_summary"`
	HoldSummary    HoldSummary          `json:"hold_summary"`
	Comparison     FlipVsHoldComparison `json:"comparison"`
	Warnings       []string             `json:"warnings"`
	SoldComps      []SoldComp           `json:"sold_comps"`
	RentalComps    []RentalComp         `json:"rental_comps"`
	Metadata       FlipVsHoldMetadata   `json:"metadata"`
}

type FlipVsHoldMetadata struct {
	TotalSoldCandidates int     `json:"total_sold_candidates"`
	TotalClosedRentals  int     `json:"total_closed_rental_candidates"`
	TotalActiveRentals  int     `json:"total_active_rental_candidates"`
	ScopeApplied        string  `json:"scope_applied"`
	RadiusMiles         float64 `json:"radius_miles,omitempty"`
	ProcessingMs        int64   `json:"processing_ms"`
	ARVMethod           string  `json:"arv_method"`
	HoldFallbackUsed    string  `json:"hold_fallback_used,omitempty"`
}

// --- Simulation response types ---

type SimulationResult struct {
	BrokerPriceOpinion  BPOSummary          `json:"broker_price_opinion"`
	ValuationSimulation SimulationSummary   `json:"valuation_simulation"`
	ValuationRisk       *ValuationRiskScore `json:"valuation_risk_score,omitempty"`
	MarketDerivedRates  MarketDerivedRates  `json:"market_derived_rates"`
	SupportingComps     []SoldComp          `json:"supporting_comps"`
	MarketConditions    *MarketConditions   `json:"market_conditions"`
	Warnings            []string            `json:"warnings"`
	Disclaimers         []string            `json:"disclaimers"`
	Metadata            SimulationMetadata  `json:"metadata"`
}

type MarketDerivedRates struct {
	GLAPerSqft       float64 `json:"gla_per_sqft"`
	PoolValue        float64 `json:"pool_value"`
	GaragePerSpace   float64 `json:"garage_per_space"`
	WaterfrontValue  float64 `json:"waterfront_value"`
	YearBuiltPerYear float64 `json:"year_built_per_year"`
	LotPerAcre       float64 `json:"lot_per_acre"`
	BedroomValue     float64 `json:"bedroom_value"`
	SampleSize       int     `json:"sample_size"`
	Confidence       string  `json:"confidence"` // "high", "moderate", "low"
}

type BPOSummary struct {
	ReportType         string  `json:"report_type"` // "Broker Price Opinion"
	IndicatedValue     float64 `json:"indicated_value"`
	ValueRangeLow      float64 `json:"value_range_low"`
	ValueRangeHigh     float64 `json:"value_range_high"`
	MethodologySummary string  `json:"methodology_summary"`
	Confidence         string  `json:"confidence"` // "high", "moderate", "low"
}

type SimulationSummary struct {
	SimulatedValue    float64                `json:"simulated_value"`
	ValueRangeLow     float64                `json:"value_range_low"`
	ValueRangeHigh    float64                `json:"value_range_high"`
	PrimaryCompsUsed  []SimulationCompDetail `json:"primary_comps_used"`
	GrossAdjRange     string                 `json:"gross_adjustment_range"` // "8.2% - 18.5%"
	CommentarySummary string                 `json:"commentary_summary"`
	ConfidenceLevel   string                 `json:"confidence_level"`
	TimeAdjApplied    bool                   `json:"time_adjustment_applied"`
	QuarterlyTrendPct float64                `json:"quarterly_trend_pct"`
	MedianPPSF        float64                `json:"median_ppsf"`
}

type SimulationCompDetail struct {
	ListingID         string          `json:"listing_id"`
	Address           string          `json:"address"`
	SoldPrice         float64         `json:"sold_price"`
	SoldDate          string          `json:"sold_date"`
	DistanceMiles     float64         `json:"distance_miles"`
	Adjustments       *AdjustmentGrid `json:"adjustments"`
	BedroomAdj        *AdjustmentLine `json:"bedroom_adjustment,omitempty"`
	TimeAdj           *AdjustmentLine `json:"time_adjustment,omitempty"`
	FullAdjustedPrice float64         `json:"full_adjusted_price"`
	GrossAdjPct       float64         `json:"gross_adj_pct"`
	ReconcileWeight   float64         `json:"reconcile_weight"`
	SubTypeMatch      bool            `json:"sub_type_match"`
}

type ValuationRiskScore struct {
	Score              float64         `json:"score"`     // 0-100
	RiskBand           string          `json:"risk_band"` // "Low", "Moderate", "Elevated", "High"
	GapAmount          float64         `json:"gap_amount"`
	GapPercent         float64         `json:"gap_percent"`
	Components         []RiskComponent `json:"components"`
	RecommendedActions []string        `json:"recommended_actions"`
}

type RiskComponent struct {
	Name     string  `json:"name"`
	RawScore float64 `json:"raw_score"` // 0-100
	Weight   float64 `json:"weight"`    // 0.0-1.0
	Weighted float64 `json:"weighted"`  // raw * weight
	Detail   string  `json:"detail"`
}

type SimulationMetadata struct {
	TotalSoldCandidates int     `json:"total_sold_candidates"`
	CompsSelected       int     `json:"comps_selected"`
	ScopeApplied        string  `json:"scope_applied"`
	RadiusMiles         float64 `json:"radius_miles"`
	ProcessingMs        int64   `json:"processing_ms"`
}

func buildAddress(streetNumber, streetDirPrefix, streetName, streetSuffix, streetDirSuffix, unitNumber, city, state, postalCode *string) string {
	var parts []string
	if streetNumber != nil && *streetNumber != "" {
		parts = append(parts, *streetNumber)
	}
	if streetDirPrefix != nil && *streetDirPrefix != "" {
		parts = append(parts, *streetDirPrefix)
	}
	if streetName != nil && *streetName != "" {
		parts = append(parts, *streetName)
	}
	if streetSuffix != nil && *streetSuffix != "" {
		parts = append(parts, *streetSuffix)
	}
	if streetDirSuffix != nil && *streetDirSuffix != "" {
		parts = append(parts, *streetDirSuffix)
	}

	street := ""
	for i, p := range parts {
		if i > 0 {
			street += " "
		}
		street += p
	}

	if unitNumber != nil && *unitNumber != "" {
		street += " #" + *unitNumber
	}

	result := street
	if city != nil && *city != "" {
		if result != "" {
			result += ", "
		}
		result += *city
	}
	if state != nil && *state != "" {
		if result != "" {
			result += ", "
		}
		result += *state
	}
	if postalCode != nil && *postalCode != "" {
		if result != "" {
			result += " "
		}
		result += *postalCode
	}
	return result
}
