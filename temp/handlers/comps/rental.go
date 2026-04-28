package comps

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/mls"
)

const (
	rentalMaxClosedResults = 50
	rentalMaxActiveResults = 30
	rentalLookbackMonths   = 12
)

// handleRentalCashflow is the handler pipeline for mode=rent_hold_cashflow.
func (h *Handler) handleRentalCashflow(w http.ResponseWriter, r *http.Request, req RunCompsRequest, subject *resolvedSubject, start time.Time) {
	ctx := r.Context()
	params := req.RentalParams

	// Expand small radius for wider candidate pool (Go-side tiering narrows).
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

	// Build rental-specific filters (wide tolerances, no sub-type match).
	filters := buildRentalFilters()
	filterClause, filterArgs := buildFilterClauses(subject, &filters)

	normDistMeters := 16093.44 // 10mi default
	if rentalScope.RadiusMiles != nil {
		normDistMeters = *rentalScope.RadiusMiles * 1609.344
	}

	simArgs := buildSimilarityArgs(subject, normDistMeters)
	soldSince := time.Now().AddDate(0, -rentalLookbackMonths, 0)

	// Parallel queries: closed leased + active rentals.
	g, gCtx := errgroup.WithContext(ctx)

	var closedRows []compRow
	var closedTotal int
	g.Go(func() error {
		var err error
		closedRows, closedTotal, err = h.queryComps(gCtx, "CompsRentalClosedBase", simArgs, scope, filterClause, filterArgs, subject.ListingID, soldSince, rentalMaxClosedResults)
		return err
	})

	var activeRows []compRow
	var activeTotal int
	g.Go(func() error {
		var err error
		activeRows, activeTotal, err = h.queryComps(gCtx, "CompsRentalActiveBase", simArgs, scope, filterClause, filterArgs, subject.ListingID, time.Time{}, rentalMaxActiveResults)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Printf("comps: rental query failed: %v", err)
		writeTimedErrorResponse(w, http.StatusInternalServerError, "Internal server error", start)
		return
	}

	// Go-side flood zone partitioning.
	if req.Filters.matchFlood() && len(subject.FloodZoneCodes) > 0 {
		closedRows, _ = partitionByFloodZone(subject.FloodZoneCodes, closedRows)
		activeRows, _ = partitionByFloodZone(subject.FloodZoneCodes, activeRows)
	}

	// Go-side tiered filtering & ranking.
	closedRanked, activeRanked, warnings := filterAndRankRentalComps(closedRows, activeRows, subject, params)

	// Rent estimate (V2: kernel similarity, decay, winsorization, blended).
	rentEst := estimateRentV2(closedRanked, activeRanked, subject, params.RentWeighting)
	if rentEst.CompCount < minClosedComps {
		warnings = append(warnings, fmt.Sprintf("Only %d closed rental comps found; rent estimate may be unreliable", rentEst.CompCount))
	}
	if rentEst.ClosedEffectiveShare > 0 && rentEst.ClosedEffectiveShare < params.RentWeighting.preferClosedMinShare() {
		warnings = append(warnings, fmt.Sprintf("Closed comp weight share %.0f%% is below preferred minimum %.0f%%",
			rentEst.ClosedEffectiveShare*100, params.RentWeighting.preferClosedMinShare()*100))
	}

	// Loan & cash flow analysis (investor view).
	loan := computeLoanSummary(params.Financing)
	monthly := computeMonthlyBreakdown(rentEst.Recommended, params, params.Ownership, loan)
	annual := computeAnnualMetrics(monthly, loan)
	scenarios := computeScenarios(rentEst.Low, rentEst.Recommended, rentEst.High, params, params.Ownership, loan)
	flags := computeScenarioFlags(scenarios)

	// Qualifying view (self-sufficiency haircut).
	dscrCfg := params.DSCRConfig
	qualPct := dscrCfg.rentQualifyingPercent()
	qualRent := rentEst.Recommended * qualPct
	qualifyingMonthly := computeQualifyingMonthlyBreakdown(rentEst.Recommended, qualRent, params, params.Ownership, loan)

	// DSCR overlay.
	dscrNOI := qualifyingMonthly.NOI
	if dscrCfg.qualifyUses() == "investor_view" {
		dscrNOI = monthly.NOI
	}
	dscrOverlay := computeDSCROverlay(dscrNOI, loan, params.Financing, dscrCfg)
	selfSuff := computeSelfSufficiency(qualRent, loan, params.Ownership, dscrCfg)

	// Map comps for response.
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

	fallbackUsed := ""
	for _, warnMsg := range warnings {
		if strings.Contains(warnMsg, "family-match") {
			fallbackUsed = "family_match"
		} else if strings.Contains(warnMsg, "cross-family") {
			fallbackUsed = "cross_family"
		} else if strings.Contains(warnMsg, "Expanded") {
			fallbackUsed = "expanded_radius"
		}
	}

	h.trackCompsEvent(ctx, req.Mode, req.Scope.Type, len(closedRanked), len(activeRanked), 0, processingMs)

	investorMonthly := monthly
	writeJSON(w, http.StatusOK, RunCompsResponse{
		Success: true,
		Subject: &subjectResp,
		RentalResult: &RentalResult{
			SubjectSummary:    subjectResp,
			RentEstimate:      rentEst,
			LoanSummary:       loan,
			Monthly:           monthly,
			Annual:            annual,
			Scenarios:         scenarios,
			ScenarioFlags:     flags,
			Warnings:          warnings,
			RentalComps:       rentalComps,
			InvestorMonthly:   &investorMonthly,
			QualifyingMonthly: &qualifyingMonthly,
			DSCROverlay:       &dscrOverlay,
			SelfSufficiency:   &selfSuff,
			Metadata: RentalMetadata{
				TotalClosedCandidates: closedTotal,
				TotalActiveCandidates: activeTotal,
				ScopeApplied:          req.Scope.Type,
				RadiusMiles:           radiusMiles,
				ProcessingMs:          processingMs,
				FallbackUsed:          fallbackUsed,
			},
		},
		Metadata: Metadata{ProcessingMs: processingMs},
	})
}

// --- Rental helpers ---

// buildRentalFilters returns SQL-side filter defaults for rental mode.
// Intentionally wide — Go-side tiered filtering handles final selection.
func buildRentalFilters() FiltersInput {
	falseVal := false
	livingAreaPct := 30
	bedsTol := 1
	bathsTol := 1
	yearBuiltTol := 25
	lotSizePct := 75
	return FiltersInput{
		LivingAreaPct:        &livingAreaPct,
		BedsTolerance:        &bedsTol,
		BathsTolerance:       &bathsTol,
		MatchPool:            &falseVal,
		MatchHOA:             &falseVal,
		MatchSeniorCommunity: &falseVal,
		MatchFlood:           &falseVal,
		MatchPropertySubType: &falseVal,
		MatchWaterfront:      &falseVal,
		YearBuiltTolerance:   &yearBuiltTol,
		LotSizePct:           &lotSizePct,
	}
}

// validateRentalParams validates the rental-specific inputs.
func validateRentalParams(req *RunCompsRequest) error {
	if req.RentalParams == nil {
		return fmt.Errorf("rental_params is required for rent_hold_cashflow mode")
	}
	f := req.RentalParams.Financing
	if f.PurchasePrice <= 0 {
		return fmt.Errorf("financing.purchase_price must be positive")
	}
	if f.LoanTermYears <= 0 {
		return fmt.Errorf("financing.loan_term_years must be positive")
	}
	if f.DownPaymentAmount == nil && f.DownPaymentPercent == nil {
		return fmt.Errorf("either financing.down_payment_amount or financing.down_payment_percent is required")
	}
	return nil
}

// estimateRent computes the weighted rent estimate from closed comps.
func estimateRent(closedComps []rankedRentalComp, activeComps []rankedRentalComp) RentEstimate {
	var rents []float64
	var weights []float64
	for _, c := range closedComps {
		if c.rent != nil {
			rents = append(rents, *c.rent)
			weights = append(weights, c.finalScore)
		}
	}

	if len(rents) == 0 {
		return RentEstimate{}
	}

	// Weighted average.
	var sumRW, sumW float64
	for i, r := range rents {
		sumRW += r * weights[i]
		sumW += weights[i]
	}
	recommended := sumRW / sumW

	// Percentiles from sorted rents.
	sort.Float64s(rents)
	low := percentile(rents, 25)
	high := percentile(rents, 75)

	// Active median.
	var activeRents []float64
	for _, c := range activeComps {
		if c.rent != nil {
			activeRents = append(activeRents, *c.rent)
		}
	}
	var activeMedian *float64
	if len(activeRents) > 0 {
		sort.Float64s(activeRents)
		m := round2(percentile(activeRents, 50))
		activeMedian = &m
	}

	return RentEstimate{
		Recommended:     round2(recommended),
		Low:             round2(low),
		High:            round2(high),
		ActiveMedian:    activeMedian,
		CompCount:       len(rents),
		ActiveCompCount: len(activeRents),
	}
}

// percentile returns the p-th percentile from a sorted slice using linear interpolation.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := p / 100.0 * float64(len(sorted)-1)
	lower := int(idx)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	frac := idx - float64(lower)
	return sorted[lower] + frac*(sorted[upper]-sorted[lower])
}

// mapRentalComps converts ranked comps to the API response type.
func mapRentalComps(ranked []rankedRentalComp, subjectFloodCodes []string) []RentalComp {
	comps := make([]RentalComp, 0, len(ranked))
	for _, rc := range ranked {
		r := rc.row
		c := RentalComp{
			Address:         buildAddress(r.StreetNumber, r.StreetDirPrefix, r.StreetName, r.StreetSuffix, r.StreetDirSuffix, r.UnitNumber, r.City, r.State, r.PostalCode),
			PropertySubType: r.PropertySubType,
			MatchQuality:    rc.matchQuality,
			SimilarityScore: round2(rc.finalScore * 100),
			Rent:            rc.rent,
			RentSource:      rc.rentSource,
			Sqft:            r.LivingArea,
			Bedrooms:        r.BedroomsTotal,
			Bathrooms:       r.BathroomsTotal,
			YearBuilt:       r.YearBuilt,
			DistanceMiles:   round2(rc.distanceMiles),
			DOM:             r.DaysOnMarket,
			Pool:            r.PoolPrivateYn,
			Waterfront:      r.WaterfrontYn,
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

		if rc.rawWeight > 0 {
			c.WeightBreakdown = &CompWeightBreakdown{
				RentPerSqft:      round4(rc.rentPerSqft),
				KernelSimilarity: round4(rc.kernelSim),
				DistanceDecay:    round4(rc.distanceDecay),
				RecencyDecay:     round4(rc.recencyDecay),
				StatusMultiplier: round2(rc.statusMult),
				RawWeight:        round4(rc.rawWeight),
				NormalizedWeight: round4(rc.normWeight),
			}
		}

		if rc.isClosedLeased {
			c.StatusLabel = "Leased/Closed"
			if r.CloseDate != nil {
				d := r.CloseDate.Format("2006-01-02")
				c.CloseDate = &d
			}
		} else {
			c.StatusLabel = "Active"
		}

		if len(subjectFloodCodes) > 0 || len(r.FloodZoneCodes) > 0 {
			ann := annotateFloodZone(subjectFloodCodes, r.FloodZoneCodes)
			c.FloodZone = &ann
		}

		comps = append(comps, c)
	}
	return comps
}
