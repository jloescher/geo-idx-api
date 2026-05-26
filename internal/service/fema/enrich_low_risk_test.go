package fema

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func TestFEMALowRiskMapping(t *testing.T) {
	x := "X"
	if !mls.ComputeLowRiskFloodZoneYN(&x) {
		t.Fatal("X should be low risk")
	}
	ae := "AE"
	if mls.ComputeLowRiskFloodZoneYN(&ae) {
		t.Fatal("AE should not be low risk")
	}
	if mls.ComputeLowRiskFloodZoneYN(nil) {
		t.Fatal("nil FEMA code should not be low risk")
	}
}
