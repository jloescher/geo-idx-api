package domain

// FeedDefinition describes an MLS catalog feed (provider-agnostic).
type FeedDefinition struct {
	Code     string `json:"code"`
	Provider string `json:"provider"` // bridge | spark
	Dataset  string `json:"dataset"`  // stellar | beaches
}

// DatasetSlug returns the mirror/upstream dataset slug with optional fallback.
func DatasetSlug(feed FeedDefinition, fallback string) string {
	if feed.Dataset != "" {
		return feed.Dataset
	}
	return fallback
}
