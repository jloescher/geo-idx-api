package dashboard_test

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/dashboard"
)

func TestListingDatasetStatus(t *testing.T) {
	tests := []struct {
		name             string
		isCurrent        bool
		hasActiveReplica bool
		want             string
	}{
		{"steady", true, false, "healthy"},
		{"catching up with page", false, true, "catching_up"},
		{"stale idle", false, false, "stale"},
		{"current ignores active page", true, true, "healthy"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dashboard.ListingDatasetStatus(tc.isCurrent, tc.hasActiveReplica)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
