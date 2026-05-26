package comps

import (
	"context"
	"math"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func (e *Engine) runRentHold(ctx context.Context, domainSlug string, feed mls.FeedDefinition, req RunRequest, subject SubjectProfile, resp RunResponse) (RunResponse, error) {
	rentals, err := e.fetchRentalComps(ctx, feed, subject, req.Scope, 25)
	if err != nil {
		resp.Warnings = append(resp.Warnings, err.Error())
	} else {
		resp.RentalComps = rentals
	}
	params := ParseRentalParams(req.RentalParams)
	purchase := params["purchase_price"]
	if purchase == 0 {
		purchase = subject.ListPrice
	}
	downPct := params["down_payment_percent"]
	if downPct == 0 {
		downPct = 20
	}
	rate := params["interest_rate"]
	if rate == 0 {
		rate = 7
	}
	term := params["loan_term_years"]
	if term == 0 {
		term = 30
	}
	loan := purchase * (1 - downPct/100)
	monthlyRate := (rate / 100) / 12
	payments := term * 12
	mortgage := 0.0
	if monthlyRate > 0 && payments > 0 {
		factor := math.Pow(1+monthlyRate, payments)
		mortgage = loan * (monthlyRate * factor) / (factor - 1)
	}
	rent := medianRent(rentals)
	tax := params["annual_tax"] / 12
	ins := params["annual_insurance"] / 12
	hoa := params["monthly_hoa"]
	if hoa == 0 {
		hoa = subject.MonthlyFees
	}
	cashflow := rent - mortgage - tax - ins - hoa
	resp.RentalResult = map[string]any{
		"monthly_rent_median": round2(rent),
		"monthly_mortgage":    round2(mortgage),
		"monthly_cashflow":    round2(cashflow),
		"purchase_price":      purchase,
	}
	return resp, nil
}

func (e *Engine) runFlipVsHold(ctx context.Context, domainSlug string, feed mls.FeedDefinition, req RunRequest, subject SubjectProfile, resp RunResponse) (RunResponse, error) {
	rentResp, err := e.runRentHold(ctx, domainSlug, feed, req, subject, resp)
	if err != nil {
		return rentResp, err
	}
	f := req.Filters
	sold, _ := e.fetchSoldComps(ctx, domainSlug, feed, subject, req.Scope, f, 12)
	sold = applyAdjustments(subject, sold, f)
	arV := medianPrice(sold)
	flip := ParseRentalParams(req.FlipParams)
	purchase := flip["purchase_price"]
	if purchase == 0 {
		purchase = subject.ListPrice
	}
	rehab := flip["rehab_budget"]
	holdMonths := flip["holding_months"]
	if holdMonths == 0 {
		holdMonths = 6
	}
	sellPct := flip["closing_costs_sell_pct"]
	if sellPct == 0 {
		sellPct = 7
	}
	sellCosts := arV * (sellPct / 100)
	flipProfit := arV - purchase - rehab - sellCosts
	holdCash := 0.0
	if rentResp.RentalResult != nil {
		if v, ok := rentResp.RentalResult["monthly_cashflow"].(float64); ok {
			holdCash = v * 12
		}
	}
	rec := "hold"
	if flipProfit > holdCash {
		rec = "flip"
	}
	rentResp.FlipVsHoldResult = map[string]any{
		"flip":           map[string]any{"estimated_profit": round2(flipProfit), "arv": arV},
		"hold":           map[string]any{"annual_cashflow": round2(holdCash)},
		"recommendation": rec,
	}
	rentResp.SoldComps = sold
	return rentResp, nil
}

func medianRent(rentals []CompRecord) float64 {
	if len(rentals) == 0 {
		return 0
	}
	var vals []float64
	for _, r := range rentals {
		if r.ClosePrice > 0 {
			vals = append(vals, r.ClosePrice)
		}
	}
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
