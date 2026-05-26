package fema

import (
	"testing"
	"time"
)

func TestEffectiveFloodZoneCode(t *testing.T) {
	mls := strPtr("MLS-X")
	fema := strPtr("AE")
	now := time.Now()

	if got := EffectiveFloodZoneCode(mls, fema, &now); got == nil || *got != "AE" {
		t.Fatalf("enriched: got %v", got)
	}
	if got := EffectiveFloodZoneCode(mls, fema, nil); got == nil || *got != "MLS-X" {
		t.Fatalf("not enriched: got %v", got)
	}
	empty := ""
	if got := EffectiveFloodZoneCode(mls, &empty, &now); got == nil || *got != "MLS-X" {
		t.Fatalf("empty fema: got %v", got)
	}
}

func strPtr(s string) *string { return &s }
