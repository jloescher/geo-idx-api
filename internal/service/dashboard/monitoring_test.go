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
		ParcelsTotal:         100,
		ParcelsLastSyncedAt:  &now,
		ParcelsStatus:        "healthy",
		CitiesTotal:          920,
		CitiesLastSyncedAt:   &now,
		CitiesStatus:         "healthy",
		CountiesTotal:        67,
		CountiesLastSyncedAt: &now,
		CountiesStatus:       "healthy",
		ZipsTotal:            900,
		ZipsLastSyncedAt:     &now,
		ZipsStatus:           "healthy",
		BoundaryStaleDays:    90,
		Status:               "healthy",
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"parcels_last_synced_at", "parcels_status",
		"cities_last_synced_at", "cities_status",
		"counties_last_synced_at", "counties_status",
		"zips_last_synced_at", "zips_status",
		"boundary_stale_days",
	} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing %s", key)
		}
	}
}

func TestBoundaryLayerStatus(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	staleDays := 90

	tests := []struct {
		name     string
		synced   *time.Time
		total    int64
		want     string
	}{
		{"zero rows", nil, 0, "unknown"},
		{"no timestamp", nil, 100, "unknown"},
		{"healthy recent", ptr(now.AddDate(0, 0, -30)), 920, "healthy"},
		{"healthy edge 89d", ptr(now.AddDate(0, 0, -89)), 920, "healthy"},
		{"stale 91d", ptr(now.AddDate(0, 0, -91)), 920, "stale"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dashboard.BoundaryLayerStatus(tc.synced, tc.total, staleDays, now)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func ptr(t time.Time) *time.Time { return &t }
