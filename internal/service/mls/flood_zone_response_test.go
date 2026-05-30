package mls

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildFloodZoneResponseEnriched(t *testing.T) {
	mlsCode := "X"
	femaCode := "AE"
	sfha := "T"
	updated := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	row := MirrorListingRow{
		FloodZoneCode:      &mlsCode,
		FEMAFloodZoneCode:  &femaCode,
		FloodZoneSFHA_TF:   &sfha,
		LowRiskFloodZoneYN: false,
		FloodZoneUpdatedAt: &updated,
	}

	out := BuildFloodZoneResponse(row)
	if out.Status != FloodZoneStatusEnriched {
		t.Fatalf("status = %q", out.Status)
	}
	if out.Source != "fema" {
		t.Fatalf("source = %q", out.Source)
	}
	if out.EffectiveCode == nil || *out.EffectiveCode != "AE" {
		t.Fatalf("effective = %+v", out.EffectiveCode)
	}
}

func TestBuildFloodZoneResponseMLSFallback(t *testing.T) {
	mlsCode := "X"
	updated := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	reason := "no_nfhl_feature"
	row := MirrorListingRow{
		FloodZoneCode:      &mlsCode,
		LowRiskFloodZoneYN: true,
		FloodZoneUpdatedAt: &updated,
		FEMAFailureReason:  &reason,
	}

	out := BuildFloodZoneResponse(row)
	if out.Status != FloodZoneStatusMLSFallback {
		t.Fatalf("status = %q", out.Status)
	}
	if out.Reason != FloodZoneReasonNFHLNoPolygon {
		t.Fatalf("reason = %q", out.Reason)
	}
}

func TestBuildPublicListingJSONIncludesFloodZoneObject(t *testing.T) {
	mlsCode := "X"
	row := MirrorListingRow{
		ListingKey:        "stellar:abc",
		ListPrice:         100,
		FloodZoneCode:     &mlsCode,
		LowRiskFloodZoneYN: true,
	}
	out := BuildPublicListingJSON(row)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	fz, ok := m["flood_zone"].(map[string]any)
	if !ok {
		t.Fatalf("flood_zone = %#v", m["flood_zone"])
	}
	if fz["mls_code"] != "X" {
		t.Fatalf("mls_code = %#v", fz["mls_code"])
	}
	if fz["low_risk"] != true {
		t.Fatalf("low_risk = %#v", fz["low_risk"])
	}
}
