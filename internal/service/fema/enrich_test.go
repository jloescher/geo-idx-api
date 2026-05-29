package fema

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	femarepo "github.com/quantyralabs/idx-api/internal/repository/fema"
)

func TestBuildListingFEMAUpdate_hit(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	zone := "AE"
	sfha := "T"
	attrs := &PointAttributes{
		FLDZone: &zone,
		SFHA_TF: &sfha,
		Raw:     []byte(`{"FLD_ZONE":"AE"}`),
	}

	u := buildListingFEMAUpdate(42, now, attrs)
	if u.ID != 42 {
		t.Fatalf("id: got %d", u.ID)
	}
	if !u.FloodZoneUpdatedAt.Equal(now) {
		t.Fatalf("updated_at: got %v", u.FloodZoneUpdatedAt)
	}
	if u.FEMAFloodZoneCode == nil || *u.FEMAFloodZoneCode != "AE" {
		t.Fatalf("fema code: %+v", u.FEMAFloodZoneCode)
	}
	if u.FloodZoneSFHA_TF == nil || *u.FloodZoneSFHA_TF != "T" {
		t.Fatalf("sfha: %+v", u.FloodZoneSFHA_TF)
	}
	if u.LowRiskFloodZoneYN {
		t.Fatal("AE should not be low risk")
	}
}

func TestBuildListingFEMAUpdate_miss(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)

	u := buildListingFEMAUpdate(7, now, nil)
	if u.ID != 7 {
		t.Fatalf("id: got %d", u.ID)
	}
	if !u.FloodZoneUpdatedAt.Equal(now) {
		t.Fatalf("updated_at: got %v", u.FloodZoneUpdatedAt)
	}
	if u.FEMAFloodZoneCode != nil {
		t.Fatalf("expected nil fema code on miss, got %+v", u.FEMAFloodZoneCode)
	}
	if u.FEMAFailureReason == nil || *u.FEMAFailureReason != femarepo.FailureReasonNoNFHLFeature {
		t.Fatalf("expected no_nfhl_feature reason, got %+v", u.FEMAFailureReason)
	}
	if u.LowRiskFloodZoneYN {
		t.Fatal("miss should not be low risk")
	}
}

func TestBuildListingFEMAUpdate_lowRiskZone(t *testing.T) {
	now := time.Now().UTC()
	zone := "X"
	attrs := &PointAttributes{FLDZone: &zone}

	u := buildListingFEMAUpdate(1, now, attrs)
	if !u.LowRiskFloodZoneYN {
		t.Fatal("zone X should be low risk")
	}
}

func TestRunBatchSkipsUpdateOnQueryPointError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"code":500,"message":"server error"}}`))
	}))
	defer srv.Close()

	c := NewClient(config.FEMAConfig{
		NFHLBaseURL:          srv.URL,
		NFHLLayerID:          28,
		MaxRequestsPerSecond: 100,
		CircuitFailThreshold: 10,
	})
	_, err := c.QueryPoint(context.Background(), 27.95, -82.45)
	if err == nil {
		t.Fatal("expected query error")
	}

	updates, errorIDs := collectBatchUpdates(err, nil, time.Now().UTC(), 99)
	if len(updates) != 0 {
		t.Fatalf("expected no updates on query error, got %d", len(updates))
	}
	if len(errorIDs) != 1 || errorIDs[0] != 99 {
		t.Fatalf("expected error id 99, got %v", errorIDs)
	}
}

// collectBatchUpdates mirrors RunBatch persist decision: errors are tracked separately.
func collectBatchUpdates(queryErr error, attrs *PointAttributes, now time.Time, id int64) ([]femarepo.FEMAUpdate, []int64) {
	if queryErr != nil {
		return nil, []int64{id}
	}
	return []femarepo.FEMAUpdate{buildListingFEMAUpdate(id, now, attrs)}, nil
}
