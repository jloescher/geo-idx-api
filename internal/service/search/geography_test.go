package search

import "testing"

func TestMergeGeographyLikePatterns(t *testing.T) {
	patterns := mergeGeographyLikePatterns(
		"Tampa",
		"pinellas",
		[]string{"Tampa", "Tampa Bay"},
		[]string{"Pinellas"},
		[]string{"pinellas"},
	)
	if len(patterns) < 3 {
		t.Fatalf("expected multiple patterns, got %v", patterns)
	}
	seen := make(map[string]bool)
	for _, p := range patterns {
		if seen[p] {
			t.Fatalf("duplicate pattern %q", p)
		}
		seen[p] = true
	}
	if !seen["%tampa%"] || !seen["%pinellas%"] {
		t.Fatalf("missing expected patterns: %v", patterns)
	}
}

func TestMergeGeographyLikePatterns_empty(t *testing.T) {
	if len(mergeGeographyLikePatterns("", "", nil, nil, nil)) != 0 {
		t.Fatal("expected no patterns")
	}
}

// TestGeographyFilter_reusesSingleArg documents appendGeographyFilter SQL: $n appears
// twice (city and county LIKE ANY) but must bind only one pattern array — duplicate
// args caused pgx "expected N arguments, got N+1" and HTTP 502 on city search.
func TestGeographyFilter_reusesSingleArg(t *testing.T) {
	patterns := mergeGeographyLikePatterns("Largo", "", nil, nil, nil)
	if len(patterns) != 1 || patterns[0] != "%largo%" {
		t.Fatalf("patterns: %v", patterns)
	}
	args := []any{"stellar"}
	args = append(args, patterns)
	if len(args) != 2 {
		t.Fatalf("want 2 args (dataset + patterns), got %d", len(args))
	}
}
