package comps

import (
	"reflect"
	"testing"
)

func TestClassifyFloodZoneCode(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"A", "A"},
		{"AE", "A"},
		{"AH", "A"},
		{"AO", "A"},
		{"a", "A"},  // lowercase
		{"ae", "A"}, // lowercase
		{"V", "V"},
		{"VE", "V"},
		{"ve", "V"},
		{"X", "X"},
		{"X500", "X"},
		{"B", "X"},
		{"C", "X"},
		{"x", "X"},
		{"", ""},
		{"D", ""},       // unknown zone
		{"  AE  ", "A"}, // whitespace
	}
	for _, tt := range tests {
		got := classifyFloodZoneCode(tt.code)
		if got != tt.want {
			t.Errorf("classifyFloodZoneCode(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestClassifyFloodZoneCodes(t *testing.T) {
	tests := []struct {
		name  string
		codes []string
		want  []string
	}{
		{"single A", []string{"AE"}, []string{"A"}},
		{"mixed AX", []string{"AE", "X"}, []string{"A", "X"}},
		{"dedup", []string{"A", "AE", "AH"}, []string{"A"}},
		{"all groups", []string{"AE", "VE", "X500"}, []string{"A", "V", "X"}},
		{"empty", nil, []string{}},
		{"unknown only", []string{"D", ""}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyFloodZoneCodes(tt.codes)
			if len(got) == 0 && len(tt.want) == 0 {
				return // both empty
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("classifyFloodZoneCodes(%v) = %v, want %v", tt.codes, got, tt.want)
			}
		})
	}
}

func TestFloodZoneMatchQuality(t *testing.T) {
	tests := []struct {
		name    string
		subject []string
		comp    []string
		want    string
	}{
		{"both empty", nil, nil, "unknown"},
		{"subject empty", nil, []string{"A"}, "unknown"},
		{"comp empty", []string{"A"}, nil, "unknown"},
		{"exact single", []string{"A"}, []string{"A"}, "exact"},
		{"exact multi", []string{"A", "X"}, []string{"A", "X"}, "exact"},
		{"no match", []string{"A"}, []string{"V"}, "no_match"},
		{"partial overlap", []string{"A", "X"}, []string{"A"}, "partial_review"},
		{"partial overlap reverse", []string{"A"}, []string{"A", "V"}, "partial_review"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := floodZoneMatchQuality(tt.subject, tt.comp)
			if got != tt.want {
				t.Errorf("floodZoneMatchQuality(%v, %v) = %q, want %q", tt.subject, tt.comp, got, tt.want)
			}
		})
	}
}

func TestAnnotateFloodZone(t *testing.T) {
	t.Run("exact", func(t *testing.T) {
		ann := annotateFloodZone([]string{"AE"}, []string{"A"})
		if ann.MatchQuality != "exact" {
			t.Errorf("got %q, want exact", ann.MatchQuality)
		}
		if ann.ReviewFlag {
			t.Error("ReviewFlag should be false for exact match")
		}
	})
	t.Run("partial_review", func(t *testing.T) {
		ann := annotateFloodZone([]string{"AE", "X"}, []string{"AE"})
		if ann.MatchQuality != "partial_review" {
			t.Errorf("got %q, want partial_review", ann.MatchQuality)
		}
		if !ann.ReviewFlag {
			t.Error("ReviewFlag should be true for partial_review")
		}
	})
	t.Run("no_match", func(t *testing.T) {
		ann := annotateFloodZone([]string{"VE"}, []string{"X"})
		if ann.MatchQuality != "no_match" {
			t.Errorf("got %q, want no_match", ann.MatchQuality)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		ann := annotateFloodZone(nil, []string{"AE"})
		if ann.MatchQuality != "unknown" {
			t.Errorf("got %q, want unknown", ann.MatchQuality)
		}
	})
}

func TestPartitionByFloodZone_SufficientMatches(t *testing.T) {
	comps := []compRow{
		{FloodZoneCodes: []string{"AE"}},
		{FloodZoneCodes: []string{"A"}},
		{FloodZoneCodes: []string{"AH"}},
		{FloodZoneCodes: []string{"VE"}}, // no match
	}
	result, warning := partitionByFloodZone([]string{"AE"}, comps)
	if warning != "" {
		t.Errorf("expected no warning, got %q", warning)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 matched comps, got %d", len(result))
	}
}

func TestPartitionByFloodZone_InsufficientMatches(t *testing.T) {
	comps := []compRow{
		{FloodZoneCodes: []string{"AE"}},
		{FloodZoneCodes: []string{"VE"}},
		{FloodZoneCodes: []string{"VE"}},
		{FloodZoneCodes: []string{"V"}},
	}
	result, warning := partitionByFloodZone([]string{"AE"}, comps)
	if warning == "" {
		t.Error("expected warning when fewer than 3 matches")
	}
	if len(result) != 4 {
		t.Errorf("expected all 4 comps returned, got %d", len(result))
	}
}

func TestPartitionByFloodZone_NoSubjectCodes(t *testing.T) {
	comps := []compRow{
		{FloodZoneCodes: []string{"AE"}},
		{FloodZoneCodes: []string{"VE"}},
	}
	result, warning := partitionByFloodZone(nil, comps)
	if warning != "" {
		t.Errorf("expected no warning, got %q", warning)
	}
	if len(result) != 2 {
		t.Errorf("expected all comps returned, got %d", len(result))
	}
}

func TestPartitionByFloodZone_MixedSubject(t *testing.T) {
	// Subject spans both A and X groups — comps in either group should match.
	comps := []compRow{
		{FloodZoneCodes: []string{"AE"}},   // matches A
		{FloodZoneCodes: []string{"X"}},    // matches X
		{FloodZoneCodes: []string{"VE"}},   // no match
		{FloodZoneCodes: []string{"X500"}}, // matches X
	}
	result, warning := partitionByFloodZone([]string{"AE", "X"}, comps)
	if warning != "" {
		t.Errorf("expected no warning, got %q", warning)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 matched comps, got %d", len(result))
	}
}

func TestPartitionByFloodZone_UnknownCompIncluded(t *testing.T) {
	// Comps with no flood zone codes should be included (not penalized).
	comps := []compRow{
		{FloodZoneCodes: []string{"AE"}},
		{FloodZoneCodes: nil},            // unknown — included
		{FloodZoneCodes: []string{}},     // unknown — included
		{FloodZoneCodes: []string{"VE"}}, // no match
	}
	result, warning := partitionByFloodZone([]string{"AE"}, comps)
	if warning != "" {
		t.Errorf("expected no warning, got %q", warning)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 comps (1 match + 2 unknown), got %d", len(result))
	}
}

func TestParseFloodZoneCodes(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"AE", []string{"AE"}},
		{"AE,X", []string{"AE", "X"}},
		{"AE, X, VE", []string{"AE", "X", "VE"}},
		{" AE , , X ", []string{"AE", "X"}},
	}
	for _, tt := range tests {
		got := parseFloodZoneCodes(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseFloodZoneCodes(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
