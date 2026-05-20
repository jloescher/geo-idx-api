package mls_test

import (
	"math"
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func TestAssociationFeeMonthlyNormalizer_Frequencies(t *testing.T) {
	n := mls.NewAssociationFeeMonthlyNormalizer()
	tests := []struct {
		name     string
		fee      any
		freq     any
		want     float64
		fallback bool
	}{
		{"Monthly", 100.0, "Monthly", 100.0, true},
		{"Annually", 1200.0, "Annually", 100.0, true},
		{"Semi-Annually", 600.0, "Semi-Annually", 100.0, true},
		{"Quarterly", 300.0, "Quarterly", 100.0, true},
		{"Weekly", 52.0, "Weekly", round2(52.0 * 52.0 / 12.0), true},
		{"Daily", 12.0, "Daily", 365.0, true},
		{"One Time", 500.0, "One Time", 0, true},
		{"null frequency", 100.0, nil, 0, true},
		{"null fee", nil, "Monthly", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := n.FromResoRow(map[string]any{
				"AssociationFee":          tc.fee,
				"AssociationFeeFrequency": tc.freq,
			}, tc.fallback)
			if tc.want == 0 {
				if got != nil {
					t.Fatalf("got %v want nil", got)
				}
				return
			}
			if got == nil || math.Abs(*got-tc.want) > 0.01 {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestAssociationFeeMonthlyNormalizer_Annual(t *testing.T) {
	n := mls.NewAssociationFeeMonthlyNormalizer()
	got := n.FromResoRow(map[string]any{
		"AssociationFee":          1200.0,
		"AssociationFeeFrequency": "Annually",
	}, true)
	if got == nil || *got != 100.0 {
		t.Fatalf("got %v want 100", got)
	}
}

func TestAssociationFeeMonthlyNormalizer_NoFallbackWhenDisabled(t *testing.T) {
	n := mls.NewAssociationFeeMonthlyNormalizer()
	got := n.FromResoRow(map[string]any{
		"Financial_sp_Information_co_Estimated_sp_Monthly_sp_Assoc_sp_Recurring_sp_Fee3": 500.0,
	}, false)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestAssociationFeeMonthlyNormalizer_ZeroSumReturnsNil(t *testing.T) {
	n := mls.NewAssociationFeeMonthlyNormalizer()
	got := n.FromResoRow(map[string]any{
		"AssociationFee":          100.0,
		"AssociationFeeFrequency": "One Time",
	}, true)
	if got != nil {
		t.Fatalf("expected nil for zero sum, got %v", got)
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
