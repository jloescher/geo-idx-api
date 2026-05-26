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

// ODataTimestampLiteral formats a UTC instant for OData $filter comparisons.
// Bridge Stellar rejects datetime'…' wrappers (HTTP 400); use bare ISO-8601 per Bridge RESO examples.
func ODataTimestampLiteral(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// ODataGTFilter builds a greater-than filter, e.g. BridgeModificationTimestamp gt 2025-05-20T00:00:00Z.
func ODataGTFilter(field string, since time.Time) string {
	return field + " gt " + ODataTimestampLiteral(since)
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
