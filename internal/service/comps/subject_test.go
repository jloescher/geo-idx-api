package comps

import "testing"

func TestMergeSubjectAliases_ListingToMLS(t *testing.T) {
	in := SubjectInput{Type: "listing", ListingID: "TB123"}
	out := mergeSubjectAliases(in)
	if out.Type != "mls" {
		t.Fatalf("type=%q want mls", out.Type)
	}
}
