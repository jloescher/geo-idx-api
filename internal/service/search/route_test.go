package search

import (
	"strings"
	"testing"
)

func TestDecideRoutePostgresOnlyDefault(t *testing.T) {
	if DecideRoute(SearchRequest{}) != RoutePostgresOnly {
		t.Fatal("expected mirror-only default")
	}
}

func TestDecideRouteUpstreamClosed(t *testing.T) {
	req := SearchRequest{Statuses: []string{"Closed"}}
	if DecideRoute(req) != RouteUpstreamOnly {
		t.Fatal("closed-only should use live upstream")
	}
}

func TestDecideRouteSplit(t *testing.T) {
	req := SearchRequest{Statuses: []string{"Active", "Closed"}}
	if DecideRoute(req) != RouteSplit {
		t.Fatal("mixed statuses should split")
	}
}

func TestBuildODataFilterStatuses(t *testing.T) {
	req := SearchRequest{Statuses: []string{"Active", "Closed"}}
	f := buildODataFilter(req)
	if f == "" || !strings.Contains(f, "StandardStatus") {
		t.Fatalf("filter %q", f)
	}
}
