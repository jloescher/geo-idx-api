package mls_test

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func TestResolveFloodZoneCodeBeaches(t *testing.T) {
	r := mls.NewResoFieldResolver()
	row := map[string]any{
		mls.BeachesSparkFloodZoneField: "X",
		"FloodZoneCode":                "AE",
	}
	got := r.ResolveFloodZoneCode(row, mls.MirrorProviderSpark, "BEACHES")
	if got == nil || *got != "X" {
		t.Fatalf("flood zone: got %v want X", got)
	}
}

func TestResolveFloodZoneCodeStellar(t *testing.T) {
	r := mls.NewResoFieldResolver()
	row := map[string]any{
		"STELLAR_FloodZoneCode": "X",
		"FloodZoneCode":         "AE",
	}
	got := r.ResolveFloodZoneCode(row, mls.MirrorProviderBridge, "STELLAR")
	if got == nil || *got != "X" {
		t.Fatalf("flood zone: got %v", got)
	}
}

func TestResolveEstimatedFeesBeachesAssociation(t *testing.T) {
	r := mls.NewResoFieldResolver()
	row := map[string]any{
		"AssociationFee":           500.22,
		"AssociationFeeFrequency":  "Monthly",
		"AssociationFee2":          312.54,
		"AssociationFee2Frequency": nil,
	}
	got := r.ResolveEstimatedTotalMonthlyFees(row, mls.MirrorProviderSpark, "BEACHES")
	if got == nil || *got != 500.22 {
		t.Fatalf("fees: got %v want 500.22", got)
	}
}

func TestResolveEstimatedFeesBeachesFallbackDeclaredMonthly(t *testing.T) {
	r := mls.NewResoFieldResolver()
	row := map[string]any{
		"AssociationFee":          nil,
		"AssociationFee2":         nil,
		"Financial_sp_Information_co_Estimated_sp_Monthly_sp_Assoc_sp_Recurring_sp_Fee3": 813.0,
	}
	got := r.ResolveEstimatedTotalMonthlyFees(row, mls.MirrorProviderSpark, "BEACHES")
	if got == nil || *got != 813.0 {
		t.Fatalf("fees: got %v want 813", got)
	}
}

func TestResolveEstimatedFeesBeachesSumTwoMonthly(t *testing.T) {
	r := mls.NewResoFieldResolver()
	row := map[string]any{
		"AssociationFee":           100.0,
		"AssociationFeeFrequency":  "Monthly",
		"AssociationFee2":          50.0,
		"AssociationFee2Frequency": "Monthly",
	}
	got := r.ResolveEstimatedTotalMonthlyFees(row, mls.MirrorProviderSpark, "BEACHES")
	if got == nil || *got != 150.0 {
		t.Fatalf("fees: got %v want 150", got)
	}
}

func TestResolveEstimatedFeesStellarAssociationWhenExtensionMissing(t *testing.T) {
	r := mls.NewResoFieldResolver()
	row := map[string]any{
		"AssociationFee":           200.0,
		"AssociationFeeFrequency":  "Monthly",
	}
	got := r.ResolveEstimatedTotalMonthlyFees(row, mls.MirrorProviderBridge, "STELLAR")
	if got == nil || *got != 200.0 {
		t.Fatalf("fees: got %v want 200", got)
	}
}

func TestResolveEstimatedFeesStellarExtension(t *testing.T) {
	r := mls.NewResoFieldResolver()
	ext := 425.50
	row := map[string]any{
		"STELLAR_TotalMonthlyFees": ext,
		"AssociationFee":           100.0,
		"AssociationFeeFrequency":  "Monthly",
	}
	got := r.ResolveEstimatedTotalMonthlyFees(row, mls.MirrorProviderBridge, "STELLAR")
	if got == nil || *got != ext {
		t.Fatalf("fees: got %v want %v", got, ext)
	}
}
