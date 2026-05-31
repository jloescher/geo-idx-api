package search

import (
	"encoding/json"
)

// parseSearchBodyFromUpstream normalizes RESO or Web API listing payloads into SearchResult.
func parseSearchBodyFromUpstream(body []byte, datasetSlug, leg string) (SearchResult, error) {
	switch leg {
	case "web":
		if results, ok := parseWebListingsBody(body); ok {
			return SearchResult{Results: filterSearchResultsForPublic(results, datasetSlug)}, nil
		}
	}
	return parseSearchBody(body, datasetSlug)
}

func parseWebListingsBody(body []byte) ([]json.RawMessage, bool) {
	var bridgeEnvelope struct {
		Bundle []json.RawMessage `json:"bundle"`
	}
	if err := json.Unmarshal(body, &bridgeEnvelope); err == nil && len(bridgeEnvelope.Bundle) > 0 {
		return bridgeEnvelope.Bundle, true
	}
	var sparkEnvelope struct {
		D struct {
			Success bool `json:"Success"`
			Results []struct {
				StandardFields json.RawMessage `json:"StandardFields"`
			} `json:"Results"`
		} `json:"D"`
	}
	if err := json.Unmarshal(body, &sparkEnvelope); err == nil && len(sparkEnvelope.D.Results) > 0 {
		out := make([]json.RawMessage, 0, len(sparkEnvelope.D.Results))
		for _, r := range sparkEnvelope.D.Results {
			if len(r.StandardFields) > 0 {
				out = append(out, r.StandardFields)
			}
		}
		if len(out) > 0 {
			return out, true
		}
	}
	return nil, false
}
