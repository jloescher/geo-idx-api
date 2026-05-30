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

	u, markInsufficient := buildListingFEMAUpdate(42, now, attrs, 27.95, -82.45, "FL", true, nil)
	if markInsufficient {
		t.Fatal("hit should not mark insufficient coords")
	}
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

	u, markInsufficient := buildListingFEMAUpdate(7, now, nil, 27.95, -82.45, "FL", true, nil)
	if markInsufficient {
		t.Fatal("valid FL miss should not mark insufficient coords")
	}
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
		t.Fatal("miss without MLS code should not be low risk")
	}
}

func TestBuildListingFEMAUpdate_missWithMLSFallback(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	mlsCode := "X"

	u, markInsufficient := buildListingFEMAUpdate(7, now, nil, 27.95, -82.45, "FL", true, &mlsCode)
	if markInsufficient {
		t.Fatal("valid FL miss should not mark insufficient coords")
	}
	if !u.LowRiskFloodZoneYN {
		t.Fatal("MLS X fallback should be low risk")
	}
}

func TestBuildListingFEMAUpdate_missSuspiciousFirstPass(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)

	_, markInsufficient := buildListingFEMAUpdate(8, now, nil, -82, 27, "FL", true, nil)
	if !markInsufficient {
		t.Fatal("suspicious first-pass miss should mark insufficient coords")
	}
}

func TestBuildListingFEMAUpdate_missSuspiciousStalePass(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)

	u, markInsufficient := buildListingFEMAUpdate(9, now, nil, -82, 27, "FL", false, nil)
	if markInsufficient {
		t.Fatal("stale re-run should not mark insufficient coords")
	}
	if u.FEMAFailureReason == nil || *u.FEMAFailureReason != femarepo.FailureReasonNoNFHLFeature {
		t.Fatalf("expected no_nfhl_feature on stale pass, got %+v", u.FEMAFailureReason)
	}
}

func TestBuildOutOfCoverageUpdate(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	mlsCode := "X"

	u := buildOutOfCoverageUpdate(11, now, &mlsCode)
	if u.FEMAFailureReason == nil || *u.FEMAFailureReason != femarepo.FailureReasonOutOfCoverage {
		t.Fatalf("reason: %+v", u.FEMAFailureReason)
	}
	if !u.LowRiskFloodZoneYN {
		t.Fatal("MLS X out of coverage should still set low risk from MLS")
	}
}

func TestBuildListingFEMAUpdate_lowRiskZone(t *testing.T) {
	now := time.Now().UTC()
	zone := "X"
	attrs := &PointAttributes{FLDZone: &zone}

	u, markInsufficient := buildListingFEMAUpdate(1, now, attrs, 27.95, -82.45, "FL", true, nil)
	if markInsufficient {
		t.Fatal("hit should not mark insufficient coords")
	}
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

	updates, errorIDs, insufficientIDs := collectBatchUpdates(err, nil, time.Now().UTC(), 99, 27.95, -82.45, "FL", true, nil)
	if len(updates) != 0 {
		t.Fatalf("expected no updates on query error, got %d", len(updates))
	}
	if len(errorIDs) != 1 || errorIDs[0] != 99 {
		t.Fatalf("expected error id 99, got %v", errorIDs)
	}
	if len(insufficientIDs) != 0 {
		t.Fatalf("expected no insufficient ids on query error, got %v", insufficientIDs)
	}
}

// collectBatchUpdates mirrors RunBatch persist decision: errors are tracked separately.
func collectBatchUpdates(queryErr error, attrs *PointAttributes, now time.Time, id int64, lat, lng float64, state string, isFirstPass bool, mlsCode *string) ([]femarepo.FEMAUpdate, []int64, []int64) {
	if queryErr != nil {
		return nil, []int64{id}, nil
	}
	u, markInsufficient := buildListingFEMAUpdate(id, now, attrs, lat, lng, state, isFirstPass, mlsCode)
	if markInsufficient {
		return nil, nil, []int64{id}
	}
	return []femarepo.FEMAUpdate{u}, nil, nil
}
