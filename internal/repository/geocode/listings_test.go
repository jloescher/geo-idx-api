package geocoderepo

import (
	"strings"
	"testing"
)

func TestActiveGeocodeJobExistsSQLMatchesJobsSchema(t *testing.T) {
	if strings.Contains(activeGeocodeJobExistsSQL, "finished_at") {
		t.Fatal("jobs table has no finished_at; completed rows are deleted from jobs")
	}
	if !strings.Contains(activeGeocodeJobExistsSQL, "reserved_at") {
		t.Fatal("expected reserved_at for pending/in-flight detection")
	}
}
