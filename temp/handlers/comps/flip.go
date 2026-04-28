package comps

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/mls"
)

const (
	flipMaxSoldResults = 25
	flipSoldLookback   = 12
)

// handleFlipVsHold runs flip and hold analysis in parallel and returns a comparison.
func (h *Handler) handleFlipVsHold(w http.ResponseWriter, r *http.Request, req RunCompsRequest, subject *resolvedSubject, start time.Time) {
	ctx := r.Context()
	params := req.RentalParams
	fp := req.FlipParams

	// Expand rental scope for wider candidate pool.
	rentalScope := req.Scope
	if rentalScope.Type == "radius" {
		radiusMiles := 1.0
		if rentalScope.RadiusMiles != nil {
			radiusMiles = *rentalScope.RadiusMiles
		}
		if radiusMiles < tier2MaxMiles {
			expanded := tier2MaxMiles
			rentalScope.RadiusMiles = &expanded
		}
	}

	scope, err := buildScopeClause(subject, rentalScope)
	if err != nil {
		writeTimedErrorResponse(w, http.StatusBadRequest, err.Error(), start)
		return
	}

	normDistMeters := 16093.44
	if rentalScope.RadiusMiles != nil {
		normDistMeters = *rentalScope.RadiusMiles * 1609.344
	}
	simArgs := buildSimilarityArgs(subject, normDistMeters)

	// Rental filters.
	rentalFilters := buildRentalFilters()
	rentalFilterClause, rentalFilterArgs := buildFilterClauses(subject, &rentalFilters)

	// Sold comp filters (use request filters for ARV).
	soldFilterClause, soldFilterArgs := buildFilterClauses(subject, &req.Filters)

	soldSince := time.Now().AddDate(0, -flipSoldLookback, 0)
	rentalSoldSince := time.Now().AddDate(0, -rentalLookbackMonths, 0)

	// Run all three queries in parallel.
	g, gCtx := errgroup.WithContext(ctx)

	var soldRows []compRow
	var soldTotal int
	g.Go(func() error {
		var err error
		soldRows, soldTotal, err = h.queryComps(gCtx, "CompsSoldBase", simArgs, scope, soldFilterClause, soldFilterArgs, subject.ListingID, soldSince, flipMaxSoldResults)
		return err
	})

	var closedRentalRows []compRow
	var closedRentalTotal int
	g.Go(func() error {
		var err error
		closedRentalRows, closedRentalTotal, err = h.queryComps(gCtx, "CompsRentalClosedBase", simArgs, scope, rentalFilterClause, rentalFilterArgs, subject.ListingID, rentalSoldSince, rentalMaxClosedResults)
		return err
	})

	var activeRentalRows []compRow
	var activeRentalTotal int
	g.Go(func() error {
		var err error
		activeRentalRows, activeRentalTotal, err = h.queryComps(gCtx, "CompsRentalActiveBase", simArgs, scope, rentalFilterClause, rentalFilterArgs, subject.ListingID, time.Time{}, rentalMaxActiveResults)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Printf("comps: flip_vs_hold query failed: %v", err)
		writeTimedErrorResponse(w, http.StatusInternalServerError, "Internal server error", start)
		return
	}

	// Go-side flood zone partitioning.
	if req.Filters.matchFlood() && len(subject.FloodZoneCodes) > 0 {
		soldRows, _ = partitionByFloodZone(subject.FloodZoneCodes, soldRows)
		closedRentalRows, _ = partitionByFloodZone(subject.FloodZoneCodes, closedRentalRows)
		activeRentalRows, _ = partitionByFloodZone(subject.FloodZoneCodes, activeRentalRows)
	}

	var warnings []string

	// --- Flip side ---
	aggMethod, _ := parseAggregationMethod(req.AggregationMethod)
	arv, arvMethod, arvCompCount := computeARV(soldRows, subject, &req.Filters, aggMethod)
	if arv == 0 {
		warnings = append(warnings, "No sold comps available for ARV computation")
	}
	flipSummary := computeFlipSummary(arv, arvMethod, arvCompCount, fp, params)

	// --- Hold side ---
	closedRanked, activeRanked, holdWarnings := filterAndRankRentalComps(closedRentalRows, activeRentalRows, subject, params)
	warnings = append(warnings, holdWarnings...)

	rentEst := estimateRentV2(closedRanked, activeRanked, subject, params.RentWeighting)
	if rentEst.CompCount < minClosedComps {
		warnings = append(warnings, fmt.Sprintf("Only %d closed rental comps found; rent estimate may be unreliable", rentEst.CompCount))
	}

	loan := computeLoanSummary(params.Financing)
	monthly := computeMonthlyBreakdown(rentEst.Recommended, params, params.Ownership, loan)
	annual := computeAnnualMetrics(monthly, loan)

	dscrCfg := params.DSCRConfig
	qualPct := dscrCfg.rentQualifyingPercent()
	qualRent := rentEst.Recommended * qualPct
	qualifyingMonthly := computeQualifyingMonthlyBreakdown(rentEst.Recommended, qualRent, params, params.Ownership, loan)

	dscrNOI := qualifyingMonthly.NOI
	if dscrCfg.qualifyUses() == "investor_view" {
		dscrNOI = monthly.NOI
	}
	dscrOverlay := computeDSCROverlay(dscrNOI, loan, params.Financing, dscrCfg)
	selfSuff := computeSelfSufficiency(qualRent, loan, params.Ownership, dscrCfg)

	holdSummary := HoldSummary{
		RecommendedRent:           rentEst.Recommended,
		MonthlyCashFlowInvestor:   monthly.CashFlow,
		MonthlyCashFlowQualifying: qualifyingMonthly.CashFlow,
		AnnualCashFlow:            annual.AnnualCashFlow,
		CashOnCash:                annual.CashOnCash,
		DSCR:                      dscrOverlay.DSCRValue,
		CapRate:                   annual.CapRate,
		CashInvested:              loan.CashInvested,
		SelfSufficient:            selfSuff.Pass,
		RentalCompCount:           rentEst.CompCount + rentEst.ActiveCompCount,
	}

	comparison := computeFlipVsHoldComparison(flipSummary, holdSummary)

	// Map comps.
	soldComps := mapFlipSoldComps(soldRows, subject, &req.Filters, subject.FloodZoneCodes, aggMethod)
	rentalComps := append(mapRentalComps(closedRanked, subject.FloodZoneCodes), mapRentalComps(activeRanked, subject.FloodZoneCodes)...)

	subjectResp := SubjectResponse{
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

	if warnings == nil {
		warnings = []string{}
	}

	processingMs := time.Since(start).Milliseconds()
	radiusMiles := 0.0
	if rentalScope.RadiusMiles != nil {
		radiusMiles = *rentalScope.RadiusMiles
	}

	holdFallback := ""
	for _, warnMsg := range holdWarnings {
		if strings.Contains(warnMsg, "family-match") {
			holdFallback = "family_match"
		} else if strings.Contains(warnMsg, "cross-family") {
			holdFallback = "cross_family"
		} else if strings.Contains(warnMsg, "Expanded") {
			holdFallback = "expanded_radius"
		}
	}

	h.trackCompsEvent(ctx, req.Mode, req.Scope.Type, len(closedRanked), len(activeRanked), len(soldRows), processingMs)

	writeJSON(w, http.StatusOK, RunCompsResponse{
		Success: true,
		Subject: &subjectResp,
		FlipVsHoldResult: &FlipVsHoldResult{
			SubjectSummary: subjectResp,
			FlipSummary:    flipSummary,
			HoldSummary:    holdSummary,
			Comparison:     comparison,
			Warnings:       warnings,
			SoldComps:      soldComps,
			RentalComps:    rentalComps,
			Metadata: FlipVsHoldMetadata{
				TotalSoldCandidates: soldTotal,
				TotalClosedRentals:  closedRentalTotal,
				TotalActiveRentals:  activeRentalTotal,
				ScopeApplied:        req.Scope.Type,
				RadiusMiles:         radiusMiles,
				ProcessingMs:        processingMs,
				ARVMethod:           arvMethod,
				HoldFallbackUsed:    holdFallback,
			},
		},
		Metadata: Metadata{ProcessingMs: processingMs},
	})
}

// mapFlipSoldComps maps sold comp rows to the SoldComp response type for flip analysis.
func mapFlipSoldComps(rows []compRow, subject *resolvedSubject, filters *FiltersInput, subjectFloodCodes []string, method AggregationMethod) []SoldComp {
	// Compute PPSF for adjustment grids.
	var ppsfs []float64
	for _, r := range rows {
		if r.ClosePrice != nil && r.LivingArea != nil && *r.LivingArea > 0 {
			ppsfs = append(ppsfs, *r.ClosePrice/float64(*r.LivingArea))
		}
	}
	centralPPSF := aggregate(ppsfs, method)

	comps := make([]SoldComp, 0, len(rows))
	for _, r := range rows {
		c := SoldComp{
			Address:         buildAddress(r.StreetNumber, r.StreetDirPrefix, r.StreetName, r.StreetSuffix, r.StreetDirSuffix, r.UnitNumber, r.City, r.State, r.PostalCode),
			SoldPrice:       r.ClosePrice,
			Bedrooms:        r.BedroomsTotal,
			Bathrooms:       r.BathroomsTotal,
			LivingAreaSqft:  r.LivingArea,
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
			DistanceMiles:   round2(r.DistanceMeters / 1609.344),
			SimilarityScore: round2(r.SimilarityScore * 100),
			DOM:             r.DaysOnMarket,
			Adjustments:     computeAdjustmentGrid(subject, r, centralPPSF, filters),
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
			ppsf := round2(*r.ClosePrice / float64(*r.LivingArea))
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

// validateFlipVsHoldParams validates flip_vs_hold mode inputs.
func validateFlipVsHoldParams(req *RunCompsRequest) error {
	if err := validateRentalParams(req); err != nil {
		return err
	}
	if req.FlipParams == nil {
		return fmt.Errorf("flip_params is required for flip_vs_hold mode")
	}
	if req.FlipParams.RepairBudget < 0 {
		return fmt.Errorf("flip_params.repair_budget must be non-negative")
	}
	return nil
}
