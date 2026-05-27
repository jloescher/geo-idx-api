package comps

import (
	"context"
	"math"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func (e *Engine) runHomeValueMode(ctx context.Context, domainSlug string, feed mls.FeedDefinition, req RunRequest, subject SubjectProfile, resp RunResponse) (RunResponse, error) {
	hv := req.HomeValueParams
	max := 25
	if hv.MaxComps != nil {
		max = *hv.MaxComps
	}
	f := req.Filters
	if hv.SoldMonthsBack != nil {
		m := *hv.SoldMonthsBack
		f.SoldMonthsBack = &m
	}
	sold, err := e.fetchSoldComps(ctx, domainSlug, feed, subject, req.Scope, f, max)
	if err != nil {
		return RunResponse{}, err
	}
	rates := extractMarketRates(sold, 6)
	sold, grids := applyURARGrid(subject, sold, rates)
	recon := reconcileBPO(subject, sold, grids, rates)
	reno := renovationCredits(subject, rates)
	renoTotal := sumRenovationCredits(reno)
	estimate := recon.PointEstimate + renoTotal
	low := recon.Low + renoTotal*0.9
	high := recon.High + renoTotal*1.1
	if estimate <= 0 {
		estimate = medianPrice(sold)
		low = round2(estimate * 0.95)
		high = round2(estimate * 1.05)
	}
	publicSold := FilterCompRecordsForPublicHomeValue(sold, feed.Dataset)
	resp.SoldComps = publicSold
	resp.HomeValueResult = map[string]any{
		"estimate":               round2(estimate),
		"low":                    round2(low),
		"high":                   round2(high),
		"confidence":             math.Max(homeValueConfidence(len(sold)), recon.Confidence),
		"comparable_count":       len(publicSold),
		"condition_applied":      subject.Condition != "",
		"condition_rating":       subject.Condition,
		"market_rates":           rates,
		"renovation_credits":     reno,
		"renovation_credit_total": renoTotal,
		"bpo_method":             rates.Method,
	}
	resp.MarketConditions = marketConditions(sold, monthsBackFilters(f, hv.SoldMonthsBack))
	return resp, nil
}

func (e *Engine) runAppraiserSim(ctx context.Context, domainSlug string, feed mls.FeedDefinition, req RunRequest, subject SubjectProfile, resp RunResponse) (RunResponse, error) {
	f := req.Filters
	sold, err := e.fetchSoldComps(ctx, domainSlug, feed, subject, req.Scope, f, 12)
	if err != nil {
		return RunResponse{}, err
	}
	sold = applyAdjustments(subject, sold, f)
	indicated := medianPrice(sold)
	if len(sold) > 0 {
		sum := 0.0
		for _, c := range sold {
			p := c.AdjustedPrice
			if p == 0 {
				p = c.ClosePrice
			}
			sum += p
		}
		indicated = round2(sum / float64(len(sold)))
	}
	resp.SoldComps = sold
	resp.SimulationResult = map[string]any{
		"indicated_value": indicated,
		"bpo_low":         round2(indicated * 0.97),
		"bpo_high":        round2(indicated * 1.03),
		"risk_score":      appraiserRisk(len(sold)),
		"risk_band":       riskBand(len(sold)),
		"comp_count":      len(sold),
	}
	return resp, nil
}

func homeValueConfidence(n int) float64 {
	switch {
	case n >= 8:
		return 80
	case n >= 5:
		return 65
	case n >= 3:
		return 50
	default:
		return 30
	}
}

func appraiserRisk(n int) float64 {
	if n >= 6 {
		return 25
	}
	if n >= 3 {
		return 45
	}
	return 70
}

func riskBand(n int) string {
	if n >= 6 {
		return "low"
	}
	if n >= 3 {
		return "moderate"
	}
	return "high"
}
