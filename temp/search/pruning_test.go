package search

import (
	"testing"
)

func TestDetectPartitionGroup(t *testing.T) {
	tests := []struct {
		name     string
		statuses []string
		want     string
	}{
		{"Single Active", []string{"Active"}, "active"},
		{"Multiple Active", []string{"Active", "Pending", "Coming Soon"}, "active"},
		{"Single Sold", []string{"Closed"}, "closed"},
		{"Mixed Active/Sold", []string{"Active", "Closed"}, ""},
		{"Single Other", []string{"Expired"}, "other"},
		{"Mixed Active/Other", []string{"Active", "Expired"}, ""},
		{"Empty", []string{}, ""},
		{"Unknown", []string{"UnknownStatus"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectPartitionGroup(tt.statuses); got != tt.want {
				t.Errorf("detectPartitionGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}
