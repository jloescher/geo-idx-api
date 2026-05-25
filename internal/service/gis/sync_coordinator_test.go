package gis_test

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/queue"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	"github.com/quantyralabs/idx-api/internal/service/gis"
)

func TestPlanBootstrapActions_freshDB(t *testing.T) {
	actions := gis.PlanBootstrapActions(gisrepo.LayerCounts{})
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].JobType != queue.TypeGISInitialSync {
		t.Fatalf("expected %s, got %s", queue.TypeGISInitialSync, actions[0].JobType)
	}
}

func TestPlanBootstrapActions_partialDB(t *testing.T) {
	counts := gisrepo.LayerCounts{
		Parcels:  0,
		Cities:   920,
		Counties: 67,
		Zips:     0,
	}
	actions := gis.PlanBootstrapActions(counts)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	types := map[string]bool{actions[0].JobType: true, actions[1].JobType: true}
	if !types[queue.TypeGISMonthlyParcelRefresh] || !types[queue.TypeGISZipSync] {
		t.Fatalf("unexpected actions: %+v", actions)
	}
}

func TestLayerCountsNeedsBootstrap(t *testing.T) {
	if !(gisrepo.LayerCounts{Cities: 1}).NeedsBootstrap() {
		t.Fatal("expected needs bootstrap when parcels and zips empty")
	}
	if (gisrepo.LayerCounts{Parcels: 1, Cities: 1, Counties: 1, Zips: 1}).NeedsBootstrap() {
		t.Fatal("expected no bootstrap when all layers populated")
	}
}

func TestLayerCountsAllEmpty(t *testing.T) {
	if !(gisrepo.LayerCounts{}).AllEmpty() {
		t.Fatal("expected all empty")
	}
	if (gisrepo.LayerCounts{Zips: 1}).AllEmpty() {
		t.Fatal("expected not all empty")
	}
}

func TestBoundariesAllEmpty(t *testing.T) {
	if !gis.BoundariesAllEmpty(0, 0, 0) {
		t.Fatal("expected all boundary layers empty")
	}
	if gis.BoundariesAllEmpty(1, 0, 0) {
		t.Fatal("expected not empty when cities present")
	}
	if gis.BoundariesAllEmpty(0, 1, 0) {
		t.Fatal("expected not empty when counties present")
	}
	if gis.BoundariesAllEmpty(0, 0, 1) {
		t.Fatal("expected not empty when zips present")
	}
}
