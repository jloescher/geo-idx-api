package comps

import (
	"sort"
	"strings"
)

// Flood zone group constants for FEMA code classification.
const (
	FloodGroupA = "A" // A, AE, AH, AO (high risk inland)
	FloodGroupV = "V" // V, VE (high risk coastal)
	FloodGroupX = "X" // X, X500, B, C (low/moderate risk)
)

// FloodZoneAnnotation is a per-comp annotation for flood zone match quality.
type FloodZoneAnnotation struct {
	SubjectCodes []string `json:"subject_codes"`
	CompCodes    []string `json:"comp_codes"`
	MatchQuality string   `json:"match_quality"` // "exact", "partial_review", "no_match", "unknown"
	ReviewFlag   bool     `json:"review_flag"`   // true for partial_review
}

// classifyFloodZoneCode maps a single FEMA flood zone code to its group.
// Returns "" for unknown or empty codes.
func classifyFloodZoneCode(code string) string {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return ""
	}
	if strings.HasPrefix(code, "V") {
		return FloodGroupV
	}
	if strings.HasPrefix(code, "X") || code == "B" || code == "C" {
		return FloodGroupX
	}
	if strings.HasPrefix(code, "A") {
		return FloodGroupA
	}
	return ""
}

// classifyFloodZoneCodes returns deduplicated, sorted groups for a set of codes.
func classifyFloodZoneCodes(codes []string) []string {
	seen := make(map[string]bool)
	for _, code := range codes {
		g := classifyFloodZoneCode(code)
		if g != "" {
			seen[g] = true
		}
	}
	groups := make([]string, 0, len(seen))
	for g := range seen {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	return groups
}

// floodZoneGroupsOverlap checks if two group sets share any group.
func floodZoneGroupsOverlap(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, g := range a {
		set[g] = true
	}
	for _, g := range b {
		if set[g] {
			return true
		}
	}
	return false
}

// floodZoneMatchQuality determines match quality between subject and comp groups.
func floodZoneMatchQuality(subjectGroups, compGroups []string) string {
	if len(subjectGroups) == 0 || len(compGroups) == 0 {
		return "unknown"
	}
	if !floodZoneGroupsOverlap(subjectGroups, compGroups) {
		return "no_match"
	}
	// Check if sets are identical.
	if len(subjectGroups) == len(compGroups) {
		match := true
		set := make(map[string]bool, len(subjectGroups))
		for _, g := range subjectGroups {
			set[g] = true
		}
		for _, g := range compGroups {
			if !set[g] {
				match = false
				break
			}
		}
		if match {
			return "exact"
		}
	}
	return "partial_review"
}

// annotateFloodZone builds a flood zone annotation for a single comp.
func annotateFloodZone(subjectCodes, compCodes []string) FloodZoneAnnotation {
	subjectGroups := classifyFloodZoneCodes(subjectCodes)
	compGroups := classifyFloodZoneCodes(compCodes)
	quality := floodZoneMatchQuality(subjectGroups, compGroups)
	return FloodZoneAnnotation{
		SubjectCodes: subjectCodes,
		CompCodes:    compCodes,
		MatchQuality: quality,
		ReviewFlag:   quality == "partial_review",
	}
}

// partitionByFloodZone splits comps into matched vs unmatched based on flood zone groups.
// If subject has no codes, all comps are returned unchanged.
// If fewer than 3 comps match, all comps are returned with a warning.
func partitionByFloodZone(subjectCodes []string, comps []compRow) ([]compRow, string) {
	if len(subjectCodes) == 0 {
		return comps, ""
	}
	subjectGroups := classifyFloodZoneCodes(subjectCodes)
	if len(subjectGroups) == 0 {
		return comps, ""
	}

	var matched []compRow
	for _, c := range comps {
		compGroups := classifyFloodZoneCodes(c.FloodZoneCodes)
		if len(compGroups) == 0 {
			// Unknown flood zone — include, don't penalize missing data.
			matched = append(matched, c)
			continue
		}
		if floodZoneGroupsOverlap(subjectGroups, compGroups) {
			matched = append(matched, c)
		}
	}

	if len(matched) >= 3 {
		return matched, ""
	}
	return comps, "No flood zone group matches found; all comps returned — review flood zone compatibility"
}

// parseFloodZoneCodes splits a comma-separated flood zone code string into a slice.
func parseFloodZoneCodes(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var codes []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			codes = append(codes, p)
		}
	}
	return codes
}
