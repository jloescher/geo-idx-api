package dashboard_test

import (
	"strings"
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/dashboard"
)

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
