package comps

import (
	"encoding/json"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// FilterCompRecordsForPublicHomeValue removes non-IDX / non-display comps from public sold_comps.
func FilterCompRecordsForPublicHomeValue(comps []CompRecord, datasetSlug string) []CompRecord {
	out := make([]CompRecord, 0, len(comps))
	for _, c := range comps {
		if len(c.Property) == 0 {
			continue
		}
		var root map[string]any
		if err := json.Unmarshal(c.Property, &root); err != nil {
			continue
		}
		flags := mls.IDXFlagsFromMap(root, datasetSlug)
		if mls.IsListingPublicCompliant(flags) {
			out = append(out, c)
		}
	}
	return out
}
