package comps

import (
	"encoding/json"
	"strings"
)

// Condition ratings used for home_value effective age (bpo_grid.effectiveYearBuilt).
const (
	conditionPoor      = "poor"
	conditionFair      = "fair"
	conditionGood      = "good"
	conditionExcellent = "excellent"
)

// DeriveConditionFromProperty infers a quality rating from RESO Property fields.
// Returns "" when no confident match (condition adjustment is skipped).
// Revenue impact: home_value accuracy for listing_id path without owner-entered condition.
func DeriveConditionFromProperty(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	if c := conditionFromPropertyCondition(propertyConditionValues(m)); c != "" {
		return c
	}
	remarks := strings.ToLower(str(m["PublicRemarks"]))
	if remarks == "" {
		remarks = strings.ToLower(str(m["PropertyDescription"]))
	}
	return conditionFromRemarks(remarks)
}

func propertyConditionValues(m map[string]any) []string {
	v, ok := m["PropertyCondition"]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		return []string{s}
	case []any:
		var out []string
		for _, item := range t {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	default:
		return nil
	}
}

// conditionFromPropertyCondition maps RESO construction-state values (Stellar).
// Only "Fixer" maps to a quality tier per comps-api.md.
func conditionFromPropertyCondition(values []string) string {
	for _, v := range values {
		n := strings.ToLower(strings.TrimSpace(v))
		if n == "fixer" || strings.Contains(n, "fixer") {
			return conditionPoor
		}
	}
	return ""
}

func conditionFromRemarks(remarks string) string {
	remarks = strings.ToLower(strings.TrimSpace(remarks))
	if remarks == "" {
		return ""
	}
	// Order: excellent and poor phrases before ambiguous good/fair terms.
	for _, phrase := range excellentRemarkPhrases {
		if strings.Contains(remarks, phrase) {
			return conditionExcellent
		}
	}
	for _, phrase := range poorRemarkPhrases {
		if strings.Contains(remarks, phrase) {
			return conditionPoor
		}
	}
	for _, phrase := range fairRemarkPhrases {
		if strings.Contains(remarks, phrase) {
			return conditionFair
		}
	}
	for _, phrase := range goodRemarkPhrases {
		if strings.Contains(remarks, phrase) {
			return conditionGood
		}
	}
	return ""
}

var excellentRemarkPhrases = []string{
	"mint condition",
	"turnkey",
	"completely renovated",
	"fully renovated",
	"like new",
	"no expense spared",
	"model perfect",
	"showcase",
}

var poorRemarkPhrases = []string{
	"fixer upper",
	"fixer-upper",
	"needs major work",
	"major work needed",
	"tear down",
	"teardown",
	"investor special",
	"distressed",
	"needs complete renovation",
	"contractor special",
}

var fairRemarkPhrases = []string{
	"needs some tlc",
	"needs tlc",
	"as-is",
	"as is",
	"needs updating",
	"handyman special",
	"estate sale",
	"bring your contractor",
}

var goodRemarkPhrases = []string{
	"well maintained",
	"well-maintained",
	"move-in ready",
	"move in ready",
	"immaculate",
	"pride of ownership",
	"updated throughout",
}

// applyDerivedCondition sets subject.Condition from listing JSON when the request did not supply one.
func applyDerivedCondition(sub *SubjectProfile, in SubjectInput) {
	if in.Condition != nil && strings.TrimSpace(*in.Condition) != "" {
		return
	}
	if strings.TrimSpace(sub.Condition) != "" {
		return
	}
	if c := DeriveConditionFromProperty(sub.Raw); c != "" {
		sub.Condition = c
	}
}
