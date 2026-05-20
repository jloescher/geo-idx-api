package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func TestHydrateReplicaBatchSparkMapsFloodAndFees(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "spark", "beaches_50_listings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Value) == 0 {
		t.Fatal("empty fixture")
	}

	var row map[string]any
	if err := json.Unmarshal(doc.Value[0], &row); err != nil {
		t.Fatal(err)
	}

	resolver := mls.NewResoFieldResolver()
	rec, action := mls.BuildListingRecord("beaches", mls.MirrorProviderSpark, row, doc.Value[0], resolver)
	if action != mls.RowActionUpsert {
		t.Fatalf("action %s", action)
	}
	if rec.EstimatedTotalMonthlyFees == nil || *rec.EstimatedTotalMonthlyFees != 500.22 {
		t.Fatalf("fees %v", rec.EstimatedTotalMonthlyFees)
	}

	// Find a row with flood zone in fixture.
	for _, r := range doc.Value {
		var m map[string]any
		if err := json.Unmarshal(r, &m); err != nil {
			continue
		}
		if m[mls.BeachesSparkFloodZoneField] == nil {
			continue
		}
		rec, action = mls.BuildListingRecord("beaches", mls.MirrorProviderSpark, m, r, resolver)
		if action != mls.RowActionUpsert || rec.FloodZoneCode == nil {
			t.Fatalf("flood row: action=%s flood=%v", action, rec.FloodZoneCode)
		}
		return
	}
	t.Fatal("fixture missing flood zone row")
}
