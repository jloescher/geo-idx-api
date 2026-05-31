package odata

import "strings"

// FeedDefaults holds dataset-specific OData query defaults for live search.
type FeedDefaults struct {
	OrderByField        string
	IDXParticipationAnd string // empty when not applicable
}

// ForDataset returns RESO OData defaults for a dataset slug.
func ForDataset(datasetSlug string) FeedDefaults {
	if strings.EqualFold(datasetSlug, "stellar") {
		return FeedDefaults{
			OrderByField:        "BridgeModificationTimestamp desc",
			IDXParticipationAnd: "(IDXParticipationYN ne false or IDXParticipationYN eq null)",
		}
	}
	return FeedDefaults{
		OrderByField: "ModificationTimestamp desc",
	}
}
