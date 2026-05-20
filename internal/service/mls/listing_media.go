package mls

import "encoding/json"

// ExtractMediaJSON returns RESO Media[] when present on the row.
//
// Bridge (Stellar): Media is inline in $select on Property.
// Spark (Beaches): Media is expanded via $expand=Media (see docs/spark/beaches_50_listings.json):
// each item includes MediaKey, MediaURL (cdn.photos.sparkplatform.com), Order, MediaCategory, @odata.id, etc.
//
// The second return value is false when the upstream row omitted Media (incremental sync without photos).
func ExtractMediaJSON(row map[string]any) (json.RawMessage, bool) {
	v, ok := row["Media"]
	if !ok {
		return nil, false
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	return b, true
}

// StripMediaFromRaw removes Media from the stored property JSON blob.
func StripMediaFromRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	if _, ok := m["Media"]; !ok {
		return raw
	}
	delete(m, "Media")
	b, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return b
}

// MergeListingJSON attaches Media to raw_data for API responses (PostGIS mirror leg).
func MergeListingJSON(raw, media json.RawMessage) json.RawMessage {
	if len(media) == 0 {
		return raw
	}
	var mediaVal any
	if err := json.Unmarshal(media, &mediaVal); err != nil {
		return raw
	}
	if len(raw) == 0 {
		out, err := json.Marshal(map[string]any{"Media": mediaVal})
		if err != nil {
			return raw
		}
		return out
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	m["Media"] = mediaVal
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}
