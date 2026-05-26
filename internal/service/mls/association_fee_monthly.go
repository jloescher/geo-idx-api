package mls

import "math"

// sparkEstimatedMonthlyFallback is BeachesMLS declared monthly HOA when fee frequencies are absent.
const sparkEstimatedMonthlyFallback = "Financial_sp_Information_co_Estimated_sp_Monthly_sp_Assoc_sp_Recurring_sp_Fee3"

var monthlyMultipliers = map[string]float64{
	"Monthly":       1.0,
	"Annually":      1.0 / 12.0,
	"Semi-Annually": 1.0 / 6.0,
	"Quarterly":     1.0 / 3.0,
	"Weekly":        52.0 / 12.0,
	"Daily":         365.0 / 12.0,
	"One Time":      0.0,
}

// AssociationFeeMonthlyNormalizer converts RESO association fees to estimated monthly total.
type AssociationFeeMonthlyNormalizer struct{}

func NewAssociationFeeMonthlyNormalizer() *AssociationFeeMonthlyNormalizer {
	return &AssociationFeeMonthlyNormalizer{}
}

// FromResoRow sums AssociationFee/AssociationFee2 with frequency multipliers.
func (n *AssociationFeeMonthlyNormalizer) FromResoRow(row map[string]any, allowDeclaredFallback bool) *float64 {
	sum := round2(
		n.componentMonthly(row["AssociationFee"], row["AssociationFeeFrequency"]) +
			n.componentMonthly(row["AssociationFee2"], row["AssociationFee2Frequency"]),
	)
	if sum > 0 {
		return &sum
	}
	if !allowDeclaredFallback {
		return nil
	}
	if v, ok := numericValue(row[sparkEstimatedMonthlyFallback]); ok && v > 0 {
		f := round2(v)
		return &f
	}
	return nil
}

func (n *AssociationFeeMonthlyNormalizer) componentMonthly(amount, frequency any) float64 {
	amt, ok := numericValue(amount)
	if !ok {
		return 0
	}
	freq := stringValue(frequency)
	if freq == "" {
		return 0
	}
	mult, ok := monthlyMultipliers[freq]
	if !ok {
		return 0
	}
	return round2(amt * mult)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
