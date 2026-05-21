package comps

import (
	"context"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func (e *Engine) runBPOMode(ctx context.Context, feed mls.FeedDefinition, req RunRequest, subject SubjectProfile, resp RunResponse) (RunResponse, error) {
	f := req.Filters
	bp := req.BPOParams
	max := 8
	if bp.MaxComps != nil {
		max = *bp.MaxComps
	}
	if f.MaxSoldComps != nil {
		max = *f.MaxSoldComps
	}
	minReg := 6
	if bp.MinCompsForRegression != nil {
		minReg = *bp.MinCompsForRegression
	}

	sold, err := e.fetchSoldComps(ctx, feed, subject, req.Scope, f, max)
	if err != nil {
		return RunResponse{}, err
	}
	if len(sold) < 3 {
		resp.Warnings = append(resp.Warnings, "insufficient sold comps for BPO; using median PPSF fallback")
	}
	if bp.ExcludeOutlierZScore != nil && *bp.ExcludeOutlierZScore > 0 {
		sold = filterOutlierZScore(sold, *bp.ExcludeOutlierZScore)
	}

	rates := extractMarketRates(sold, minReg)
	resp.Warnings = append(resp.Warnings, rates.Warnings...)

	sold, grids := applyURARGrid(subject, sold, rates)
	for i := range grids {
		grids[i].Weight = round2(compReconcileWeight(subject, sold[i], grids[i], rates))
	}

	recon := reconcileBPO(subject, sold, grids, rates)
	reno := renovationCredits(subject, rates)
	renoTotal := sumRenovationCredits(reno)
	if renoTotal > 0 {
		recon.PointEstimate = round2(recon.PointEstimate + renoTotal)
		recon.Low = round2(recon.Low + renoTotal*0.9)
		recon.High = round2(recon.High + renoTotal*1.1)
	}

	if bp.ConfidenceThreshold != nil && recon.Confidence < *bp.ConfidenceThreshold {
		resp.Warnings = append(resp.Warnings, "BPO confidence below requested threshold")
	}

	resp.SoldComps = sold
	resp.BPOResult = map[string]any{
		"point_estimate":          recon.PointEstimate,
		"indicated_value":         recon.PointEstimate,
		"range":                   map[string]float64{"low": recon.Low, "high": recon.High},
		"confidence":              recon.Confidence,
		"confidence_band":         recon.ConfidenceBand,
		"method":                  rates.Method,
		"reconciliation_summary":  recon.ReconciliationSummary,
		"market_rates":            rates,
		"adjustment_grid":         grids,
		"renovation_credits":      reno,
		"renovation_credit_total": renoTotal,
		"comp_count":              len(sold),
		"grid_lines":              len(urarGridFeatures),
	}
	resp.MarketConditions = marketConditions(sold, monthsBackFilters(f, bp.SoldMonthsBack))
	return resp, nil
}

func monthsBackFilters(f FiltersInput, bpoMonths *int) int {
	if bpoMonths != nil && *bpoMonths > 0 {
		return *bpoMonths
	}
	if f.SoldMonthsBack != nil {
		return *f.SoldMonthsBack
	}
	return 12
}
