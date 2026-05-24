package dashboard_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/quantyralabs/idx-api/internal/service/dashboard"
)

func TestGISMetricJSONFreshnessFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	m := dashboard.GISMetric{
		ParcelsTotal:        100,
		ParcelsLastSyncedAt: &now,
		ZipsTotal:           900,
		ZipsLastSyncedAt:    &now,
		Status:              "healthy",
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["parcels_last_synced_at"]; !ok {
		t.Fatal("missing parcels_last_synced_at")
	}
	if _, ok := decoded["zips_last_synced_at"]; !ok {
		t.Fatal("missing zips_last_synced_at")
	}
}
