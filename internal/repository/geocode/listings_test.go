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

func TestPendingSelectionSkipsBadAddressFlaggedRows(t *testing.T) {
	if !strings.Contains(selectPendingBaseSQL, "geocode_bad_address_yn IS FALSE") {
		t.Fatal("pending selection must skip rows flagged as bad addresses")
	}
	if !strings.Contains(countPendingBaseSQL, "geocode_bad_address_yn IS FALSE") {
		t.Fatal("pending count must skip rows flagged as bad addresses")
	}
}

func TestApplyCoordsClearsFailureFlags(t *testing.T) {
	if !strings.Contains(applyCoordsSQL, "geocode_bad_address_yn = FALSE") {
		t.Fatal("apply coords should clear bad-address flag")
	}
	if !strings.Contains(applyCoordsSQL, "geocode_failure_reason = NULL") {
		t.Fatal("apply coords should clear failure reason")
	}
	if !strings.Contains(applyCoordsSQL, "geocode_attempt_count = COALESCE(geocode_attempt_count, 0) + 1") {
		t.Fatal("apply coords should increment attempt count")
	}
}

func TestMarkFailedAttemptTracksAttemptMetadata(t *testing.T) {
	if !strings.Contains(markFailedAttemptSQL, "geocode_failed_at = NOW()") {
		t.Fatal("failed attempt should set geocode_failed_at")
	}
	if !strings.Contains(markFailedAttemptSQL, "geocode_failure_reason = $2") {
		t.Fatal("failed attempt should persist failure reason")
	}
	if !strings.Contains(markFailedAttemptSQL, "geocode_bad_address_yn = $4") {
		t.Fatal("failed attempt should persist bad-address marker")
	}
}
