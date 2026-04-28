package comps

import (
	"regexp"
	"strings"
)

// KeywordResult holds computed keyword scores and matched phrases.
type KeywordResult struct {
	DisrepairScore     float64
	GoodConditionScore float64
	DisrepairMatches   []string
	GoodCondMatches    []string
}

// computeKeywordScores evaluates user-provided keywords against public remarks.
// For modes D/E, returns neutral (0.5) scores with no matches.
func computeKeywordScores(remarks string, keywords map[string][]Keyword, mode string) KeywordResult {
	if mode == "D" || mode == "E" {
		return KeywordResult{
			DisrepairScore:     0.5,
			GoodConditionScore: 0.5,
			DisrepairMatches:   []string{},
			GoodCondMatches:    []string{},
		}
	}

	result := KeywordResult{
		DisrepairMatches: []string{},
		GoodCondMatches:  []string{},
	}

	if remarks == "" || len(keywords) == 0 {
		return result
	}

	lower := strings.ToLower(remarks)

	result.DisrepairScore, result.DisrepairMatches = scoreCategory(lower, remarks, keywords["disrepair"])
	result.GoodConditionScore, result.GoodCondMatches = scoreCategory(lower, remarks, keywords["good_condition"])

	return result
}

// scoreCategory computes a weighted score for a single keyword category.
// Returns (score, matchedPhrases).
func scoreCategory(lowerRemarks, rawRemarks string, keywords []Keyword) (float64, []string) {
	if len(keywords) == 0 {
		return 0, []string{}
	}

	var totalWeight float64
	var matchedWeight float64
	var matched []string

	for _, kw := range keywords {
		totalWeight += kw.Weight
		if matchKeyword(lowerRemarks, rawRemarks, kw) {
			matchedWeight += kw.Weight
			matched = append(matched, kw.Phrase)
		}
	}

	if matched == nil {
		matched = []string{}
	}

	if totalWeight == 0 {
		return 0, matched
	}

	score := matchedWeight / totalWeight
	if score > 1.0 {
		score = 1.0
	}
	return score, matched
}

// matchKeyword checks if a single keyword matches the remarks text.
func matchKeyword(lowerRemarks, rawRemarks string, kw Keyword) bool {
	if kw.IsRegex {
		re, err := regexp.Compile("(?i)" + kw.Phrase)
		if err != nil {
			return false
		}
		return re.MatchString(rawRemarks)
	}
	return strings.Contains(lowerRemarks, strings.ToLower(kw.Phrase))
}
