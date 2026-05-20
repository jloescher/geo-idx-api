package mls

import (
	"strings"
	"time"
)

// UsesBridgeModificationOData reports whether incremental sync should prefer BridgeModificationTimestamp.
func UsesBridgeModificationOData(datasetSlug string) bool {
	switch strings.ToLower(strings.TrimSpace(datasetSlug)) {
	case "beaches":
		return false
	default:
		return true
	}
}

// ModificationODataField returns the RESO property name for incremental $filter/$orderby.
func ModificationODataField(datasetSlug string) string {
	if UsesBridgeModificationOData(datasetSlug) {
		return "BridgeModificationTimestamp"
	}
	return "ModificationTimestamp"
}

// ResolveModificationTimestamp picks the canonical stored timestamp for a mirror row.
func ResolveModificationTimestamp(datasetSlug string, row map[string]any) *time.Time {
	if UsesBridgeModificationOData(datasetSlug) {
		if ts := timestampPtr(row["BridgeModificationTimestamp"]); ts != nil {
			return ts
		}
	}
	return timestampPtr(row["ModificationTimestamp"])
}

// MaxModificationFromRow is the canonical modification time for cursor advancement.
func MaxModificationFromRow(datasetSlug string, row map[string]any) *time.Time {
	return ResolveModificationTimestamp(datasetSlug, row)
}
