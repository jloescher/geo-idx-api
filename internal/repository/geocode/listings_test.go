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

func TestPendingSelectionRecoveryColumnIsNeverSQLNull(t *testing.T) {
	if !strings.Contains(selectPendingBaseSQL, "COALESCE(fema_failure_reason = 'insufficient_coords', false) AS recovery") {
		t.Fatal("recovery must COALESCE to false when fema_failure_reason is NULL (pgx cannot scan NULL into bool)")
	}
}

func TestPendingSelectionIncludesInsufficientCoordsRecovery(t *testing.T) {
	if !strings.Contains(selectPendingBaseSQL, "fema_failure_reason = 'insufficient_coords'") {
		t.Fatal("pending selection must include FEMA insufficient_coords recovery rows")
	}
	if !strings.Contains(countPendingBaseSQL, "fema_failure_reason = 'insufficient_coords'") {
		t.Fatal("pending count must include FEMA insufficient_coords recovery rows")
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
	if !strings.Contains(applyCoordsSQL, "flood_zone_updated_at = NULL") {
		t.Fatal("apply coords should clear FEMA watermark on recovery")
	}
	if !strings.Contains(applyCoordsSQL, "fema_failure_reason = 'insufficient_coords'") {
		t.Fatal("apply coords should allow insufficient_coords recovery overwrite")
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
	if !strings.Contains(markFailedAttemptSQL, "fema_failure_reason = 'insufficient_coords'") {
		t.Fatal("failed attempt should allow insufficient_coords recovery rows")
	}
}
