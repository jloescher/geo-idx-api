package dashboard_test

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/dashboard"
)

func TestStaleReservedAfter(t *testing.T) {
	tests := []struct {
		name     string
		timeout  int
		wantSecs int
	}{
		{"default hour uses half", 3600, 1800},
		{"short timeout floors at 10m", 60, 600},
		{"zero uses default hour half", 0, 1800},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dashboard.StaleReservedAfter(tc.timeout)
			if got != tc.wantSecs {
				t.Fatalf("got %d want %d", got, tc.wantSecs)
			}
		})
	}
}

func TestCacheStatus(t *testing.T) {
	tests := []struct {
		total, rate float64
		want        string
	}{
		{0, 0, "no_data"},
		{10, 50, "healthy"},
	}
	for _, tc := range tests {
		got := dashboard.CacheStatus(int64(tc.total), tc.rate)
		if got != tc.want {
			t.Fatalf("total=%v rate=%v got %q want %q", tc.total, tc.rate, got, tc.want)
		}
	}
}

func TestQueueStatus(t *testing.T) {
	tests := []struct {
		pending, stale, failed int64
		want                   string
	}{
		{0, 0, 0, "healthy"},
		{0, 0, 1, "stale"},
		{501, 0, 0, "stale"},
		{10, 1, 0, "stale"},
	}
	for _, tc := range tests {
		got := dashboard.QueueStatus(tc.pending, tc.stale, tc.failed)
		if got != tc.want {
			t.Fatalf("pending=%d stale=%d failed=%d got %q want %q", tc.pending, tc.stale, tc.failed, got, tc.want)
		}
	}
}

func TestSyncPipelineStatus(t *testing.T) {
	healthy := dashboard.SyncPipelineStatus(nil)
	if healthy != "healthy" {
		t.Fatalf("empty got %q", healthy)
	}
	stale := dashboard.SyncPipelineStatus([]repository.ReplicaPageStatusCount{
		{Status: "failed", Count: 1},
	})
	if stale != "stale" {
		t.Fatalf("failed row got %q", stale)
	}
}

func TestInfraStatus(t *testing.T) {
	if dashboard.InfraStatus(true) != "healthy" {
		t.Fatal("leader active should be healthy")
	}
	if dashboard.InfraStatus(false) != "critical" {
		t.Fatal("no leader should be critical")
	}
}

func TestBuildIncidents(t *testing.T) {
	inc := dashboard.BuildIncidents(dashboard.IncidentInput{
		InfraStatus:           "critical",
		SchedulerLeaderActive: false,
		TotalFailed:           3,
		TotalStaleReserved:    1,
		SyncPipelineStatus:    "stale",
	})
	if len(inc) != 4 {
		t.Fatalf("want 4 incidents got %d", len(inc))
	}
}

func TestReplicaPageChipStatus(t *testing.T) {
	if dashboard.ReplicaPageChipStatus("pending", 10) != "healthy" {
		t.Fatal("low pending should be healthy")
	}
	if dashboard.ReplicaPageChipStatus("pending", 501) != "stale" {
		t.Fatal("high pending should be stale")
	}
}
