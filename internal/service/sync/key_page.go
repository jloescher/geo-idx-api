package sync

import (
	"encoding/json"
)

// KeyPageResult is one OData page of listing keys for mirror reconciliation.
type KeyPageResult struct {
	Keys           []string
	NextURL        *string
	Complete       bool
	CollectionSkip int
	UseFallback    bool
	HTTPError      bool
	Forbidden      bool
	HTTPStatus     int
	ODataError     string
	FetchURL       string
}

func listingKeysFromRows(rows []json.RawMessage) []string {
	out := make([]string, 0, len(rows))
	for _, raw := range rows {
		var row struct {
			ListingKey string `json:"ListingKey"`
		}
		if err := json.Unmarshal(raw, &row); err != nil || row.ListingKey == "" {
			continue
		}
		out = append(out, row.ListingKey)
	}
	return out
}

func dedupeListingKeys(keys []string) []string {
	if len(keys) == 0 {
		return keys
	}
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}
