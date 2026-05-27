package search

import (
	"encoding/json"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// filterSearchResultsForPublic drops non-compliant upstream listings and applies visibility.
func filterSearchResultsForPublic(results []json.RawMessage, datasetSlug string) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(results))
	for _, raw := range results {
		sanitized := mls.SanitizeUpstreamPropertyJSONWithDataset(raw, datasetSlug)
		if len(sanitized) == 0 {
			continue
		}
		var root map[string]any
		if err := json.Unmarshal(sanitized, &root); err != nil {
			continue
		}
		flags := mls.IDXFlagsFromMap(root, datasetSlug)
		if !mls.IsListingPublicCompliant(flags) {
			continue
		}
		body, ok := mls.ApplyPublicListingVisibilityJSON(sanitized, flags, mls.VisibilityPublicSearch)
		if !ok || len(body) == 0 {
			continue
		}
		out = append(out, body)
	}
	return out
}
