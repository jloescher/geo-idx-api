package comps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/analytics"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/mls"
)

const maxCompsBodyBytes = 1 << 20 // 1 MB

// Handler handles comps API requests.
type Handler struct {
	Pool      *pgxpool.Pool
	Registry  *db.Registry
	Analytics *analytics.Client
}

// NewHandler creates a new comps handler.
func NewHandler(pool *pgxpool.Pool, registry *db.Registry) *Handler {
	return &Handler{Pool: pool, Registry: registry}
}

// WithAnalytics sets the analytics client for the handler.
func (h *Handler) WithAnalytics(client *analytics.Client) *Handler {
	h.Analytics = client
	return h
}

// HandleRunComps handles POST /api/v1/comps/run.
func (h *Handler) HandleRunComps(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	// 1. Parse request body.
	body := http.MaxBytesReader(w, r.Body, maxCompsBodyBytes)
	defer body.Close()

	payload, err := io.ReadAll(body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeTimedErrorResponse(w, http.StatusRequestEntityTooLarge, "Request body too large", start)
			return
		}
		writeTimedErrorResponse(w, http.StatusBadRequest, "Invalid request body", start)
		return
	}

	var req RunCompsRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		writeTimedErrorResponse(w, http.StatusBadRequest, "Invalid JSON: "+err.Error(), start)
		return
	}

	// 2. Normalize MLS prefix for DB queries.
	req.Subject.ListingID = mls.AddPrefix(req.Subject.ListingID)

	// 3. Validate.
	if err := validateRequest(&req); err != nil {
		writeTimedErrorResponse(w, http.StatusBadRequest, err.Error(), start)
		return
	}

	// 3. Resolve subject.
	subject, err := h.resolveSubject(ctx, req.Subject)
	if err != nil {
		if errors.Is(err, errSubjectNotFound) {
			writeJSON(w, http.StatusOK, RunCompsResponse{
				Success:  false,
				Error:    "Subject listing not found",
				Metadata: Metadata{ProcessingMs: time.Since(start).Milliseconds()},
			})
			return
		}
		log.Printf("comps: resolveSubject failed: %v", err)
		writeTimedErrorResponse(w, http.StatusInternalServerError, "Internal server error", start)
		return
	}

	// Rental mode: separate pipeline.
	if req.Mode == "rent_hold_cashflow" {
		h.handleRentalCashflow(w, r, req, subject, start)
		return
	}

	// Flip vs Hold mode: separate pipeline.
	if req.Mode == "flip_vs_hold" {
		h.handleFlipVsHold(w, r, req, subject, start)
		return
	}

	// Appraiser simulation mode: separate pipeline.
	if req.Mode == "appraiser_simulation" {
		h.handleAppraiserSimulation(w, r, req, subject, start)
		return
	}

	// 4. Build scope and filter clauses.
	scope, err := buildScopeClause(subject, req.Scope)
	if err != nil {
		writeTimedErrorResponse(w, http.StatusBadRequest, err.Error(), start)
		return
	}

	filterClause, filterArgs := buildFilterClauses(subject, &req.Filters)

	// 5. Determine the normalization distance for similarity scoring.
	// For radius scope, use the actual radius. For others, use a default 10 miles.
	normDistMeters := 16093.44 // 10 miles default
	if req.Scope.Type == "radius" && req.Scope.RadiusMiles != nil {
		normDistMeters = *req.Scope.RadiusMiles * 1609.344
	}

	// 6. Execute queries in parallel.
	soldSince := time.Now().AddDate(0, -req.Filters.soldMonthsBack(), 0)

	// Common similarity parameters: $1=lng, $2=lat, $3=norm_dist,
	// $4=living_area, $5=beds, $6=baths, $7=year_built,
	// $8=pool, $9=hoa, $10=flood, $11=lot_acres, $12=property_type
	simArgs := buildSimilarityArgs(subject, normDistMeters)

	g, gCtx := errgroup.WithContext(ctx)

	var soldRows []compRow
	var soldTotal int
	g.Go(func() error {
		var err error
		soldRows, soldTotal, err = h.queryComps(gCtx, "CompsSoldBase", simArgs, scope, filterClause, filterArgs, subject.ListingID, soldSince, req.Filters.maxSoldComps())
		return err
	})

	var compRows []compRow
	var compTotal int
	if req.Filters.includeActivePending() {
		g.Go(func() error {
			var err error
			compRows, compTotal, err = h.queryComps(gCtx, "CompsCompetitionBase", simArgs, scope, filterClause, filterArgs, subject.ListingID, time.Time{}, req.Filters.maxCompetitionComps())
			return err
		})
	}

	var failedRows []compRow
	var failedTotal int
	if req.Filters.includeFailedListings() {
		g.Go(func() error {
			var err error
			failedRows, failedTotal, err = h.queryComps(gCtx, "CompsFailedBase", simArgs, scope, filterClause, filterArgs, subject.ListingID, time.Time{}, req.Filters.maxFailedListings())
			return err
		})
	}

	if err := g.Wait(); err != nil {
		log.Printf("comps: query failed: %v", err)
		writeTimedErrorResponse(w, http.StatusInternalServerError, "Internal server error", start)
		return
	}

	// 6b. Go-side flood zone partitioning (replaces SQL boolean filter).
	var floodWarnings []string
	if req.Filters.matchFlood() && len(subject.FloodZoneCodes) > 0 {
		var w string
		soldRows, w = partitionByFloodZone(subject.FloodZoneCodes, soldRows)
		if w != "" {
			floodWarnings = append(floodWarnings, w)
		}
		compRows, w = partitionByFloodZone(subject.FloodZoneCodes, compRows)
		if w != "" && len(floodWarnings) == 0 {
			floodWarnings = append(floodWarnings, w)
		}
		failedRows, w = partitionByFloodZone(subject.FloodZoneCodes, failedRows)
		if w != "" && len(floodWarnings) == 0 {
			floodWarnings = append(floodWarnings, w)
		}
	}

	// 7. Compute market conditions and PPSF (needed for adjustments).
	aggMethod, _ := parseAggregationMethod(req.AggregationMethod)
	marketConditions := computeMarketConditions(soldRows, soldTotal, compTotal, req.Filters.soldMonthsBack(), aggMethod)
	medianPPSF := ppsfFromRows(soldRows, aggMethod)

	// 8. Map results and compute keyword scores + adjustment grids.
	soldComps := mapSoldComps(soldRows, req.Keywords, req.Mode, subject, medianPPSF, &req.Filters, subject.FloodZoneCodes)
	competitionComps := mapCompetitionComps(compRows, subject.FloodZoneCodes)
	failedListings := mapFailedListings(failedRows, subject.FloodZoneCodes)

	// 9. Compute overpriced signals.
	var overpricedSignals []OverpricedSignal
	if req.Filters.includeOverpricedSignals() {
		overpricedSignals = computeOverpricedSignals(soldComps, competitionComps, failedListings, aggMethod)
	}
	if overpricedSignals == nil {
		overpricedSignals = []OverpricedSignal{}
	}

	// 10. Build subject response.
	subjectResp := &SubjectResponse{
		ListingID:       mls.StripPrefix(subject.ListingID),
		Address:         subject.Address,
		Lat:             subject.Lat,
		Lng:             subject.Lng,
		Bedrooms:        subject.Bedrooms,
		Bathrooms:       subject.Bathrooms,
		LivingAreaSqft:  subject.LivingAreaSqft,
		LotSizeSqft:     subject.LotSizeSqft,
		YearBuilt:       subject.YearBuilt,
		ListPrice:       subject.ListPrice,
		PropertyType:    subject.PropertyType,
		PropertySubType: subject.PropertySubType,
		Waterfront:      subject.Waterfront,
		GarageSpaces:    subject.GarageSpaces,
		SeniorCommunity: subject.SeniorCommunity,
		FloodZoneCodes:  subject.FloodZoneCodes,
	}

	// 11. Build metadata.
	processingMs := time.Since(start).Milliseconds()
	radiusMiles := 0.0
	if req.Scope.RadiusMiles != nil {
		radiusMiles = *req.Scope.RadiusMiles
	}

	metadata := Metadata{
		TotalSoldCandidates:        soldTotal,
		TotalCompetitionCandidates: compTotal,
		TotalFailedCandidates:      failedTotal,
		ScopeApplied:               req.Scope.Type,
		RadiusMiles:                radiusMiles,
		ProcessingMs:               processingMs,
		OverpricedAvailable:        req.Filters.includeOverpricedSignals(),
		FailedListingsAvailable:    req.Filters.includeFailedListings(),
		AggregationMethod:          string(aggMethod),
	}

	// 12. Track analytics.
	h.trackCompsEvent(ctx, req.Mode, req.Scope.Type, len(soldComps), len(competitionComps), len(failedListings), processingMs)

	// 13. Return response.
	writeJSON(w, http.StatusOK, RunCompsResponse{
		Success:           true,
		Subject:           subjectResp,
		SoldComps:         soldComps,
		CompetitionComps:  competitionComps,
		FailedListings:    failedListings,
		OverpricedSignals: overpricedSignals,
		MarketConditions:  marketConditions,
		Metadata:          metadata,
		Warnings:          floodWarnings,
	})
}

// --- Validation ---

var validModes = map[string]bool{"A": true, "B": true, "C": true, "D": true, "E": true, "rent_hold_cashflow": true, "flip_vs_hold": true, "appraiser_simulation": true}
var validScopeTypes = map[string]bool{"radius": true, "polygon": true, "neighborhood": true, "zip": true}

func validateRequest(req *RunCompsRequest) error {
	if req.Subject.Type == "" {
		return fmt.Errorf("subject.type is required")
	}
	if req.Subject.Type != "mls" && req.Subject.Type != "off_market" {
		return fmt.Errorf("subject.type must be 'mls' or 'off_market'")
	}
	if req.Subject.Type == "mls" && req.Subject.ListingID == "" {
		return fmt.Errorf("subject.listing_id is required when type is 'mls'")
	}
	if req.Subject.Type == "off_market" {
		if req.Subject.Lat == nil || req.Subject.Lng == nil {
			return fmt.Errorf("subject.lat and subject.lng are required when type is 'off_market'")
		}
	}

	if !validModes[req.Mode] {
		return fmt.Errorf("mode must be one of: A, B, C, D, E, rent_hold_cashflow, flip_vs_hold, appraiser_simulation")
	}

	if req.Mode == "rent_hold_cashflow" {
		if err := validateRentalParams(req); err != nil {
			return err
		}
	}

	if req.Mode == "flip_vs_hold" {
		if err := validateFlipVsHoldParams(req); err != nil {
			return err
		}
	}

	if req.Mode == "appraiser_simulation" {
		if err := validateSimulationParams(req); err != nil {
			return err
		}
	}

	if _, err := parseAggregationMethod(req.AggregationMethod); err != nil {
		return err
	}

	if !validScopeTypes[req.Scope.Type] {
		return fmt.Errorf("scope.type must be one of: radius, polygon, neighborhood, zip")
	}

	return nil
}

// --- Subject resolution ---

var errSubjectNotFound = fmt.Errorf("subject not found")

func (h *Handler) resolveSubject(ctx context.Context, input SubjectInput) (*resolvedSubject, error) {
	if input.Type == "mls" {
		return h.resolveMLSSubject(ctx, input.ListingID)
	}
	return resolveOffMarketSubject(input), nil
}

func (h *Handler) resolveMLSSubject(ctx context.Context, listingID string) (*resolvedSubject, error) {
	sql := h.Registry.SQL("CompsResolveSubject")
	row := h.Pool.QueryRow(ctx, sql, listingID)

	var (
		lid, stdStatus, mlsStatus                                             *string
		lat, lng                                                              *float64
		livingArea, beds, baths, yearBuilt                                    *int
		lotAcres, listPrice, closePrice, origPrice                            *float64
		pool, hoa, seniorCommunity, flood, waterfront                         *bool
		floodZoneCodes                                                        []string
		garageSpaces                                                          *int
		propType, propSubType                                                 *string
		onMarketDate, closeDate                                               *time.Time
		streetNum, streetDirPre, streetName, streetSuf, streetDirSuf, unitNum *string
		city, state, postalCode                                               *string
	)

	err := row.Scan(
		&lid, &stdStatus, &mlsStatus,
		&lat, &lng,
		&livingArea, &lotAcres,
		&beds, &baths, &yearBuilt,
		&pool, &hoa, &seniorCommunity, &flood, &floodZoneCodes,
		&waterfront, &garageSpaces,
		&propType, &propSubType,
		&listPrice, &closePrice, &closeDate, &onMarketDate,
		&origPrice,
		&streetNum, &streetDirPre, &streetName, &streetSuf, &streetDirSuf, &unitNum,
		&city, &state, &postalCode,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errSubjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resolve subject: %w", err)
	}

	s := &resolvedSubject{
		Bedrooms:          beds,
		Bathrooms:         baths,
		YearBuilt:         yearBuilt,
		Pool:              pool,
		HOA:               hoa,
		SeniorCommunity:   seniorCommunity,
		FloodZone:         flood,
		FloodZoneCodes:    floodZoneCodes,
		Waterfront:        waterfront,
		GarageSpaces:      garageSpaces,
		ListPrice:         listPrice,
		PropertyType:      propType,
		PropertySubType:   propSubType,
		OnMarketDate:      onMarketDate,
		StandardStatus:    stdStatus,
		OriginalListPrice: origPrice,
		LotSizeAcres:      lotAcres,
	}

	if lid != nil {
		s.ListingID = *lid
	}
	if lat != nil {
		s.Lat = *lat
	}
	if lng != nil {
		s.Lng = *lng
	}
	if livingArea != nil {
		s.LivingAreaSqft = livingArea
	}
	if lotAcres != nil {
		sqft := int(*lotAcres * 43560)
		s.LotSizeSqft = &sqft
	}

	s.Address = buildAddress(streetNum, streetDirPre, streetName, streetSuf, streetDirSuf, unitNum, city, state, postalCode)

	return s, nil
}

func resolveOffMarketSubject(input SubjectInput) *resolvedSubject {
	s := &resolvedSubject{
		Bedrooms:        input.Bedrooms,
		Bathrooms:       input.Bathrooms,
		YearBuilt:       input.YearBuilt,
		Pool:            input.Pool,
		HOA:             input.HOA,
		SeniorCommunity: input.SeniorCommunity,
		Waterfront:      input.Waterfront,
		GarageSpaces:    input.GarageSpaces,
	}
	if input.Lat != nil {
		s.Lat = *input.Lat
	}
	if input.Lng != nil {
		s.Lng = *input.Lng
	}
	if input.LivingAreaSqft != nil {
		s.LivingAreaSqft = input.LivingAreaSqft
	}
	if input.LotSizeSqft != nil {
		s.LotSizeSqft = input.LotSizeSqft
		acres := float64(*input.LotSizeSqft) / 43560.0
		s.LotSizeAcres = &acres
	}
	if input.AskingPrice != nil {
		s.ListPrice = input.AskingPrice
	}
	if input.FloodZoneCode != nil {
		isLowRisk := *input.FloodZoneCode == "" || strings.HasPrefix(strings.ToUpper(*input.FloodZoneCode), "X")
		s.FloodZone = &isLowRisk
		s.FloodZoneCodes = parseFloodZoneCodes(*input.FloodZoneCode)
	}
	return s
}

// --- Query execution ---

func buildSimilarityArgs(subject *resolvedSubject, normDistMeters float64) []any {
	var livingArea, beds, baths, yearBuilt any
	var pool, hoa, flood any
	var lotAcres any
	var propType any
	var waterfront any
	var garageSpaces any

	if subject.LivingAreaSqft != nil {
		livingArea = *subject.LivingAreaSqft
	}
	if subject.Bedrooms != nil {
		beds = *subject.Bedrooms
	}
	if subject.Bathrooms != nil {
		baths = *subject.Bathrooms
	}
	if subject.YearBuilt != nil {
		yearBuilt = *subject.YearBuilt
	}
	if subject.Pool != nil {
		pool = *subject.Pool
	}
	if subject.HOA != nil {
		hoa = *subject.HOA
	}
	if subject.FloodZone != nil {
		flood = *subject.FloodZone
	}
	if subject.LotSizeAcres != nil {
		lotAcres = *subject.LotSizeAcres
	}
	if subject.PropertyType != nil {
		propType = *subject.PropertyType
	}
	if subject.Waterfront != nil {
		waterfront = *subject.Waterfront
	}
	if subject.GarageSpaces != nil {
		garageSpaces = *subject.GarageSpaces
	}

	// $1=lng, $2=lat, $3=norm_dist, $4=living_area, $5=beds, $6=baths,
	// $7=year_built, $8=pool, $9=hoa, $10=flood, $11=lot_acres, $12=property_type,
	// $13=waterfront, $14=garage_spaces
	return []any{
		subject.Lng, subject.Lat, normDistMeters,
		livingArea, beds, baths, yearBuilt,
		pool, hoa, flood, lotAcres, propType,
		waterfront, garageSpaces,
	}
}

// queryComps executes a named comp query with scope and filter injection.
// For sold comps, soldSince is set; for competition/failed, it's zero.
func (h *Handler) queryComps(
	ctx context.Context,
	queryName string,
	simArgs []any,
	scope scopeResult,
	filterClause string,
	filterArgs []any,
	subjectListingID string,
	soldSince time.Time,
	maxResults int,
) ([]compRow, int, error) {
	baseSQL := h.Registry.SQL(db.QueryName(queryName))

	// Inject scope clause.
	scopeSQL := scope.clause
	if filterClause != "" {
		scopeSQL += "\n  " + filterClause
	}

	// Renumber scope/filter placeholders starting after the similarity args.
	nextIdx := len(simArgs) + 1
	scopeSQL, nextIdx = renumberPlaceholders(scopeSQL, nextIdx)

	// Build combined clause for injection.
	baseSQL = strings.Replace(baseSQL, "/* SCOPE */", scopeSQL, 1)
	baseSQL = strings.Replace(baseSQL, "/* FILTERS */", "", 1) // filters already in scopeSQL

	// Build args: simArgs + scope args + filter args + (soldSince or nothing) + maxResults
	args := make([]any, 0, len(simArgs)+len(scope.args)+len(filterArgs)+2)
	args = append(args, simArgs...)
	args = append(args, scope.args...)
	args = append(args, filterArgs...)

	// Replace named placeholder tokens with next sequential parameter indices.
	// These unique tokens won't collide with renumbered $SCOPE/$FILTER placeholders.
	baseSQL = strings.Replace(baseSQL, "$EXCLUDE_LISTING", fmt.Sprintf("$%d", nextIdx), 1)
	nextIdx++
	args = append(args, subjectListingID)

	if queryName == "CompsSoldBase" || queryName == "CompsRentalClosedBase" {
		baseSQL = strings.Replace(baseSQL, "$SOLD_SINCE", fmt.Sprintf("$%d", nextIdx), 1)
		nextIdx++
		baseSQL = strings.Replace(baseSQL, "$RESULT_LIMIT", fmt.Sprintf("$%d", nextIdx), 1)
		args = append(args, soldSince, maxResults)
	} else {
		baseSQL = strings.Replace(baseSQL, "$RESULT_LIMIT", fmt.Sprintf("$%d", nextIdx), 1)
		args = append(args, maxResults)
	}

	rows, err := h.Pool.Query(ctx, baseSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query %s: %w", queryName, err)
	}
	defer rows.Close()

	var results []compRow
	totalCandidates := 0

	for rows.Next() {
		var r compRow
		scanTargets := buildScanTargets(&r, queryName)
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, 0, fmt.Errorf("scan %s: %w", queryName, err)
		}
		totalCandidates = r.TotalCandidates
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows %s: %w", queryName, err)
	}

	return results, totalCandidates, nil
}

func buildScanTargets(r *compRow, queryName string) []any {
	base := []any{
		&r.ListingID, &r.StandardStatus, &r.MLSStatus,
		&r.StreetNumber, &r.StreetDirPrefix, &r.StreetName, &r.StreetSuffix,
		&r.StreetDirSuffix, &r.UnitNumber, &r.City, &r.State, &r.PostalCode,
		&r.Latitude, &r.Longitude,
	}

	switch queryName {
	case "CompsSoldBase", "CompsRentalClosedBase":
		base = append(base,
			&r.ListPrice, &r.ClosePrice, &r.CloseDate,
			&r.OriginalListPrice, &r.PreviousListPrice,
		)
	case "CompsCompetitionBase", "CompsRentalActiveBase":
		base = append(base,
			&r.ListPrice, &r.OriginalListPrice, &r.PreviousListPrice,
		)
	case "CompsFailedBase":
		base = append(base,
			&r.ListPrice, &r.OriginalListPrice, &r.PreviousListPrice,
		)
	}

	base = append(base,
		&r.LivingArea, &r.LotSizeAcres,
		&r.BedroomsTotal, &r.BathroomsTotal, &r.YearBuilt,
		&r.PoolPrivateYn, &r.AssociationYn, &r.SeniorCommunityYn, &r.LowRiskFloodYn,
		&r.WaterfrontYn, &r.GarageSpaces,
		&r.PropertyType, &r.PropertySubType,
		&r.StoriesTotal, &r.BathroomsFull, &r.BathroomsHalf,
		&r.OnMarketDate,
	)

	if queryName == "CompsSoldBase" || queryName == "CompsFailedBase" || queryName == "CompsRentalClosedBase" {
		base = append(base, &r.BecameInactiveAt)
	}

	base = append(base, &r.PublicRemarks, &r.FloodZoneCodes)

	if queryName == "CompsRentalClosedBase" || queryName == "CompsRentalActiveBase" {
		base = append(base, &r.MfrLeasePrice)
	}

	base = append(base,
		&r.DaysOnMarket,
		&r.DistanceMeters,
		&r.SimilarityScore,
		&r.TotalCandidates,
	)

	return base
}

// --- Result mapping ---

func mapSoldComps(rows []compRow, keywords map[string][]Keyword, mode string, subject *resolvedSubject, medianPPSF float64, filters *FiltersInput, subjectFloodCodes []string) []SoldComp {
	comps := make([]SoldComp, 0, len(rows))
	for _, r := range rows {
		remarks := ""
		if r.PublicRemarks != nil {
			remarks = *r.PublicRemarks
		}
		kw := computeKeywordScores(remarks, keywords, mode)

		c := SoldComp{
			Address:            buildAddress(r.StreetNumber, r.StreetDirPrefix, r.StreetName, r.StreetSuffix, r.StreetDirSuffix, r.UnitNumber, r.City, r.State, r.PostalCode),
			SoldPrice:          r.ClosePrice,
			Bedrooms:           r.BedroomsTotal,
			Bathrooms:          r.BathroomsTotal,
			LivingAreaSqft:     r.LivingArea,
			DOM:                r.DaysOnMarket,
			DistanceMiles:      r.DistanceMeters / 1609.344,
			SimilarityScore:    r.SimilarityScore * 100, // scale to 0-100
			DisrepairScore:     kw.DisrepairScore,
			GoodConditionScore: kw.GoodConditionScore,
			KeywordMatches: map[string][]string{
				"disrepair":      kw.DisrepairMatches,
				"good_condition": kw.GoodCondMatches,
			},
			YearBuilt:       r.YearBuilt,
			LotSizeAcres:    r.LotSizeAcres,
			Waterfront:      r.WaterfrontYn,
			GarageSpaces:    r.GarageSpaces,
			Stories:         r.StoriesTotal,
			BathroomsFull:   r.BathroomsFull,
			BathroomsHalf:   r.BathroomsHalf,
			Pool:            r.PoolPrivateYn,
			PropertyType:    r.PropertyType,
			PropertySubType: r.PropertySubType,
		}

		if r.ListingID != nil {
			c.ListingID = mls.StripPrefix(*r.ListingID)
		}
		if r.Latitude != nil {
			c.Lat = *r.Latitude
		}
		if r.Longitude != nil {
			c.Lng = *r.Longitude
		}
		if r.CloseDate != nil {
			d := r.CloseDate.Format("2006-01-02")
			c.SoldDate = &d
		}
		if r.ClosePrice != nil && r.LivingArea != nil && *r.LivingArea > 0 {
			ppsf := *r.ClosePrice / float64(*r.LivingArea)
			c.PPSF = &ppsf
		}

		c.Adjustments = computeAdjustmentGrid(subject, r, medianPPSF, filters)

		if len(subjectFloodCodes) > 0 || len(r.FloodZoneCodes) > 0 {
			ann := annotateFloodZone(subjectFloodCodes, r.FloodZoneCodes)
			c.FloodZone = &ann
		}

		comps = append(comps, c)
	}
	return comps
}

func mapCompetitionComps(rows []compRow, subjectFloodCodes []string) []CompetitionComp {
	comps := make([]CompetitionComp, 0, len(rows))
	for _, r := range rows {
		c := CompetitionComp{
			Address:         buildAddress(r.StreetNumber, r.StreetDirPrefix, r.StreetName, r.StreetSuffix, r.StreetDirSuffix, r.UnitNumber, r.City, r.State, r.PostalCode),
			ListPrice:       r.ListPrice,
			Bedrooms:        r.BedroomsTotal,
			Bathrooms:       r.BathroomsTotal,
			LivingAreaSqft:  r.LivingArea,
			DOM:             r.DaysOnMarket,
			DistanceMiles:   r.DistanceMeters / 1609.344,
			SimilarityScore: r.SimilarityScore * 100,
			YearBuilt:       r.YearBuilt,
			LotSizeAcres:    r.LotSizeAcres,
			Waterfront:      r.WaterfrontYn,
			GarageSpaces:    r.GarageSpaces,
			Stories:         r.StoriesTotal,
			BathroomsFull:   r.BathroomsFull,
			BathroomsHalf:   r.BathroomsHalf,
			Pool:            r.PoolPrivateYn,
			PropertyType:    r.PropertyType,
			PropertySubType: r.PropertySubType,
		}

		if r.ListingID != nil {
			c.ListingID = mls.StripPrefix(*r.ListingID)
		}
		if r.StandardStatus != nil {
			c.Status = *r.StandardStatus
		}
		if r.Latitude != nil {
			c.Lat = *r.Latitude
		}
		if r.Longitude != nil {
			c.Lng = *r.Longitude
		}
		if r.ListPrice != nil && r.LivingArea != nil && *r.LivingArea > 0 {
			ppsf := *r.ListPrice / float64(*r.LivingArea)
			c.PPSF = &ppsf
		}

		if len(subjectFloodCodes) > 0 || len(r.FloodZoneCodes) > 0 {
			ann := annotateFloodZone(subjectFloodCodes, r.FloodZoneCodes)
			c.FloodZone = &ann
		}

		comps = append(comps, c)
	}
	return comps
}

func mapFailedListings(rows []compRow, subjectFloodCodes []string) []FailedListing {
	listings := make([]FailedListing, 0, len(rows))
	for _, r := range rows {
		fl := FailedListing{
			Address:           buildAddress(r.StreetNumber, r.StreetDirPrefix, r.StreetName, r.StreetSuffix, r.StreetDirSuffix, r.UnitNumber, r.City, r.State, r.PostalCode),
			LastListPrice:     r.ListPrice,
			OriginalListPrice: r.OriginalListPrice,
			Bedrooms:          r.BedroomsTotal,
			Bathrooms:         r.BathroomsTotal,
			LivingAreaSqft:    r.LivingArea,
			DOM:               r.DaysOnMarket,
			DistanceMiles:     r.DistanceMeters / 1609.344,
			SimilarityScore:   r.SimilarityScore * 100,
			YearBuilt:         r.YearBuilt,
			LotSizeAcres:      r.LotSizeAcres,
			Waterfront:        r.WaterfrontYn,
			GarageSpaces:      r.GarageSpaces,
			Stories:           r.StoriesTotal,
			BathroomsFull:     r.BathroomsFull,
			BathroomsHalf:     r.BathroomsHalf,
			Pool:              r.PoolPrivateYn,
			PropertyType:      r.PropertyType,
			PropertySubType:   r.PropertySubType,
		}

		if r.ListingID != nil {
			fl.ListingID = mls.StripPrefix(*r.ListingID)
		}
		if r.StandardStatus != nil {
			fl.StandardStatus = *r.StandardStatus
		}
		if r.MLSStatus != nil {
			fl.MLSStatus = *r.MLSStatus
		}
		if r.OnMarketDate != nil {
			d := r.OnMarketDate.Format("2006-01-02")
			fl.OnMarketDate = &d
		}
		if r.BecameInactiveAt != nil {
			d := r.BecameInactiveAt.Format("2006-01-02")
			fl.BecameInactiveAt = &d
		}
		if r.ListPrice != nil && r.LivingArea != nil && *r.LivingArea > 0 {
			ppsf := *r.ListPrice / float64(*r.LivingArea)
			fl.PPSF = &ppsf
		}

		// Count price reductions.
		if r.OriginalListPrice != nil && r.ListPrice != nil && *r.ListPrice < *r.OriginalListPrice {
			fl.PriceReductions = 1
			if r.PreviousListPrice != nil && *r.PreviousListPrice != *r.ListPrice && *r.PreviousListPrice != *r.OriginalListPrice {
				fl.PriceReductions = 2
			}
		}

		if len(subjectFloodCodes) > 0 || len(r.FloodZoneCodes) > 0 {
			ann := annotateFloodZone(subjectFloodCodes, r.FloodZoneCodes)
			fl.FloodZone = &ann
		}

		listings = append(listings, fl)
	}
	return listings
}

// --- Market conditions ---

func ppsfFromRows(rows []compRow, method AggregationMethod) float64 {
	var ppsfs []float64
	for _, r := range rows {
		if r.ClosePrice != nil && r.LivingArea != nil && *r.LivingArea > 0 {
			ppsfs = append(ppsfs, *r.ClosePrice/float64(*r.LivingArea))
		}
	}
	return aggregate(ppsfs, method)
}

func computeMarketConditions(soldRows []compRow, soldTotal, compTotal, soldMonthsBack int, method AggregationMethod) *MarketConditions {
	mc := &MarketConditions{}

	// Central tendency sold PPSF.
	var ppsfs []float64
	for _, r := range soldRows {
		if r.ClosePrice != nil && r.LivingArea != nil && *r.LivingArea > 0 {
			ppsfs = append(ppsfs, *r.ClosePrice/float64(*r.LivingArea))
		}
	}
	if v := aggregate(ppsfs, method); v > 0 {
		rounded := math.Round(v*100) / 100
		mc.MedianSoldPPSF = &rounded
	}

	// Central tendency sold DOM.
	var doms []float64
	for _, r := range soldRows {
		if r.DaysOnMarket != nil {
			doms = append(doms, float64(*r.DaysOnMarket))
		}
	}
	if v := aggregate(doms, method); v > 0 {
		dom := int(v)
		mc.MedianSoldDOM = &dom
	}

	// Average list-to-sale ratio.
	var ratioSum float64
	var ratioCount int
	for _, r := range soldRows {
		if r.ClosePrice != nil && r.OriginalListPrice != nil && *r.OriginalListPrice > 0 {
			ratioSum += *r.ClosePrice / *r.OriginalListPrice
			ratioCount++
		}
	}
	if ratioCount > 0 {
		ratio := math.Round(ratioSum/float64(ratioCount)*1000) / 1000
		mc.AvgListToSaleRatio = &ratio
	}

	// Months of inventory: active_count / (sold_count / months).
	if soldMonthsBack > 0 && soldTotal > 0 && compTotal > 0 {
		absorptionRate := float64(soldTotal) / float64(soldMonthsBack)
		moi := math.Round(float64(compTotal)/absorptionRate*10) / 10
		mc.MonthsOfInventory = &moi
	}

	return mc
}

// --- Adjustment grid ---

func computeAdjustmentGrid(subject *resolvedSubject, comp compRow, medianPPSF float64, filters *FiltersInput) *AdjustmentGrid {
	if comp.ClosePrice == nil {
		return nil
	}

	var lines []AdjustmentLine
	var absSum float64

	// GLA adjustment: (subject_sqft - comp_sqft) * median_ppsf.
	if subject.LivingAreaSqft != nil && comp.LivingArea != nil && medianPPSF > 0 {
		diff := float64(*subject.LivingAreaSqft - *comp.LivingArea)
		adj := math.Round(diff * medianPPSF)
		sizeWord := "larger"
		if diff < 0 {
			sizeWord = "smaller"
		}
		reasoning := fmt.Sprintf("Subject is %s sqft %s (%s vs %s sqft); adjusted %s at $%.2f/sqft median PPSF",
			itoa(int(math.Abs(diff))), sizeWord,
			itoa(*subject.LivingAreaSqft), itoa(*comp.LivingArea),
			formatDollarsSigned(adj), medianPPSF)
		lines = append(lines, AdjustmentLine{
			Feature:    "gla",
			SubjectVal: *subject.LivingAreaSqft,
			CompVal:    *comp.LivingArea,
			Adjustment: adj,
			Reasoning:  reasoning,
		})
		absSum += math.Abs(adj)
	}

	// Pool adjustment.
	if subject.Pool != nil && comp.PoolPrivateYn != nil && *subject.Pool != *comp.PoolPrivateYn {
		adj := filters.adjPoolValue()
		var reasoning string
		if *subject.Pool {
			reasoning = fmt.Sprintf("Subject has pool; comp does not; adjusted %s", formatDollarsSigned(adj))
		} else {
			adj = -adj
			reasoning = fmt.Sprintf("Comp has pool; subject does not; adjusted %s", formatDollarsSigned(adj))
		}
		lines = append(lines, AdjustmentLine{
			Feature:    "pool",
			SubjectVal: *subject.Pool,
			CompVal:    *comp.PoolPrivateYn,
			Adjustment: adj,
			Reasoning:  reasoning,
		})
		absSum += math.Abs(adj)
	}

	// Garage adjustment.
	if subject.GarageSpaces != nil && comp.GarageSpaces != nil {
		diff := *subject.GarageSpaces - *comp.GarageSpaces
		if diff != 0 {
			adj := float64(diff) * filters.adjGaragePerSpace()
			moreOrFewer := "more"
			if diff < 0 {
				moreOrFewer = "fewer"
			}
			reasoning := fmt.Sprintf("Subject has %s %s garage space(s) (%s vs %s); adjusted %s at %s/space",
				itoa(int(math.Abs(float64(diff)))), moreOrFewer,
				itoa(*subject.GarageSpaces), itoa(*comp.GarageSpaces),
				formatDollarsSigned(adj), formatDollars(filters.adjGaragePerSpace()))
			lines = append(lines, AdjustmentLine{
				Feature:    "garage",
				SubjectVal: *subject.GarageSpaces,
				CompVal:    *comp.GarageSpaces,
				Adjustment: adj,
				Reasoning:  reasoning,
			})
			absSum += math.Abs(adj)
		}
	}

	// Waterfront adjustment.
	if subject.Waterfront != nil && comp.WaterfrontYn != nil && *subject.Waterfront != *comp.WaterfrontYn {
		adj := filters.adjWaterfrontValue()
		var reasoning string
		if *subject.Waterfront {
			reasoning = fmt.Sprintf("Subject is waterfront; comp is not; adjusted %s", formatDollarsSigned(adj))
		} else {
			adj = -adj
			reasoning = fmt.Sprintf("Comp is waterfront; subject is not; adjusted %s", formatDollarsSigned(adj))
		}
		lines = append(lines, AdjustmentLine{
			Feature:    "waterfront",
			SubjectVal: *subject.Waterfront,
			CompVal:    *comp.WaterfrontYn,
			Adjustment: adj,
			Reasoning:  reasoning,
		})
		absSum += math.Abs(adj)
	}

	// Year built adjustment.
	if subject.YearBuilt != nil && comp.YearBuilt != nil {
		diff := *subject.YearBuilt - *comp.YearBuilt
		if diff != 0 {
			adj := float64(diff) * filters.adjYearBuiltPerYear()
			ageWord := "newer"
			if diff < 0 {
				ageWord = "older"
			}
			reasoning := fmt.Sprintf("Subject is %s years %s (%s vs %s); adjusted %s at %s/year",
				itoa(int(math.Abs(float64(diff)))), ageWord,
				itoa(*subject.YearBuilt), itoa(*comp.YearBuilt),
				formatDollarsSigned(adj), formatDollars(filters.adjYearBuiltPerYear()))
			lines = append(lines, AdjustmentLine{
				Feature:    "year_built",
				SubjectVal: *subject.YearBuilt,
				CompVal:    *comp.YearBuilt,
				Adjustment: adj,
				Reasoning:  reasoning,
			})
			absSum += math.Abs(adj)
		}
	}

	// Lot size adjustment.
	if subject.LotSizeAcres != nil && comp.LotSizeAcres != nil {
		diff := *subject.LotSizeAcres - *comp.LotSizeAcres
		if diff != 0 {
			adj := math.Round(diff * filters.adjLotPerAcre())
			sizeWord := "larger"
			if diff < 0 {
				sizeWord = "smaller"
			}
			reasoning := fmt.Sprintf("Subject lot is %.2f acres %s (%.2f vs %.2f acres); adjusted %s at %s/acre",
				math.Abs(diff), sizeWord,
				*subject.LotSizeAcres, *comp.LotSizeAcres,
				formatDollarsSigned(adj), formatDollars(filters.adjLotPerAcre()))
			lines = append(lines, AdjustmentLine{
				Feature:    "lot_size",
				SubjectVal: *subject.LotSizeAcres,
				CompVal:    *comp.LotSizeAcres,
				Adjustment: adj,
				Reasoning:  reasoning,
			})
			absSum += math.Abs(adj)
		}
	}

	// Compute totals.
	net := 0.0
	for _, l := range lines {
		net += l.Adjustment
	}

	adjustedPrice := *comp.ClosePrice + net
	grossPct := 0.0
	if *comp.ClosePrice > 0 {
		grossPct = absSum / *comp.ClosePrice * 100
	}

	return &AdjustmentGrid{
		Lines:          lines,
		NetAdjustment:  math.Round(net),
		AdjustedPrice:  math.Round(adjustedPrice),
		GrossAdjPct:    math.Round(grossPct*10) / 10,
		HighAdjWarning: grossPct > 25,
	}
}

// --- Helpers ---

func (h *Handler) trackCompsEvent(ctx context.Context, mode, scopeType string, soldCount, compCount, failedCount int, durationMs int64) {
	if h.Analytics == nil {
		return
	}
	props := analytics.CompsProperties(mode, scopeType, soldCount, compCount, failedCount, durationMs)
	h.Analytics.CaptureWithCorrelation(ctx, analytics.EventAPICompsExecuted, props)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeTimedErrorResponse(w http.ResponseWriter, status int, message string, start time.Time) {
	writeJSON(w, status, RunCompsResponse{
		Success:  false,
		Error:    message,
		Metadata: Metadata{ProcessingMs: time.Since(start).Milliseconds()},
	})
}
