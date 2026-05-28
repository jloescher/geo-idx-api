package dashboard_test

import (
	"strings"
	"testing"

	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/dashboard"
)

func TestStaleReservedIncidentDetail(t *testing.T) {
	detail := dashboard.StaleReservedIncidentDetail(dashboard.IncidentInput{
		StaleReservedAfterSecs: 1800,
		StaleInFlight: []repository.InFlightJob{
			{JobID: 42, Queue: "spark-sync-persist", JobType: "spark.persist_chunk", AgeSeconds: 1900},
		},
	})
	if !strings.Contains(detail, "spark.persist_chunk") || !strings.Contains(detail, "id 42") {
		t.Fatalf("expected job detail in incident copy, got %q", detail)
	}
}

func TestSchedulerLeaderIncidentDetail(t *testing.T) {
	incidents := dashboard.BuildIncidents(dashboard.IncidentInput{
		InfraStatus:           "critical",
		SchedulerLeaderActive: false,
		SchedulerLastEnqueue:  "2026-05-28T12:00:00Z",
	})
	if len(incidents) == 0 {
		t.Fatal("expected incident")
	}
	if !strings.Contains(incidents[0].Detail, "HAProxy") {
		t.Fatalf("expected HAProxy hint, got %q", incidents[0].Detail)
	}
}
