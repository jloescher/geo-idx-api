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
