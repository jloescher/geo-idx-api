package mls

import "encoding/json"

// ExtractMediaJSON returns RESO Media[] when present on the row.
func ExtractMediaJSON(row map[string]any) (json.RawMessage, bool) {
	return ExtractJSONBField(row, "Media")
}

// StripMediaFromRaw removes Media from the stored property JSON blob.
func StripMediaFromRaw(raw json.RawMessage) json.RawMessage {
	return StripJSONBKeysFromRaw(raw, []string{"Media"})
}

// MergeListingJSON attaches Media to raw_data for API responses (legacy helper).
func MergeListingJSON(raw, media json.RawMessage) json.RawMessage {
	return MergeMirrorListing(raw, ExpandedPayload{Media: media, HasMedia: len(media) > 0}, nil)
}
