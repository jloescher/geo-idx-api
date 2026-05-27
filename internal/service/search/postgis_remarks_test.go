package search

import (
	"strings"
	"testing"
)

func TestPublicListingComplianceSQL(t *testing.T) {
	if !strings.Contains(publicListingComplianceSQL, "internet_entire_listing_display_yn") {
		t.Fatal("missing display gate")
	}
	if !strings.Contains(publicListingComplianceSQL, "idx_participation_yn") {
		t.Fatal("missing IDX gate")
	}
}
