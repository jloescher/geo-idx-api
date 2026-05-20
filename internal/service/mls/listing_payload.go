package mls

import (
	"encoding/json"
	"strings"
)

const defaultSyncExpand = "Media,Unit,Room,OpenHouse"

// ParseExpandKeys splits a comma-separated OData $expand list (e.g. MLS_SYNC_EXPAND).
func ParseExpandKeys(expandCSV string) []string {
	expandCSV = strings.TrimSpace(expandCSV)
	if expandCSV == "" {
		expandCSV = defaultSyncExpand
	}
	parts := strings.Split(expandCSV, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// ExpandedPayload holds RESO navigation collections split from raw_data at persist.
type ExpandedPayload struct {
	Media     json.RawMessage
	Unit      json.RawMessage
	Room      json.RawMessage
	OpenHouse json.RawMessage
	HasMedia  bool
	HasUnit   bool
	HasRoom   bool
	HasOpenHouse bool
}

// ExtractJSONBField returns one expanded collection when present on the upstream row.
func ExtractJSONBField(row map[string]any, key string) (json.RawMessage, bool) {
	v, ok := row[key]
	if !ok {
		return nil, false
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	return b, true
}

// ExtractExpandedPayloads reads all configured expand keys from the row.
func ExtractExpandedPayloads(row map[string]any, expandKeys []string) ExpandedPayload {
	var p ExpandedPayload
	for _, key := range expandKeys {
		val, ok := ExtractJSONBField(row, key)
		if !ok {
			continue
		}
		switch key {
		case "Media":
			p.Media, p.HasMedia = val, true
		case "Unit":
			p.Unit, p.HasUnit = val, true
		case "Room":
			p.Room, p.HasRoom = val, true
		case "OpenHouse":
			p.OpenHouse, p.HasOpenHouse = val, true
		}
	}
	return p
}

// StripJSONBKeysFromRaw removes expanded keys from the stored property JSON blob.
func StripJSONBKeysFromRaw(raw json.RawMessage, keys []string) json.RawMessage {
	if len(raw) == 0 || len(keys) == 0 {
		return raw
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	changed := false
	for _, key := range keys {
		if _, ok := m[key]; ok {
			delete(m, key)
			changed = true
		}
	}
	if !changed {
		return raw
	}
	b, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return b
}

// BuildCustomFields collects unmapped RESO keys not stored in typed columns or expanded JSONB.
func BuildCustomFields(row map[string]any, expandKeys []string) json.RawMessage {
	mapped := mappedScalarFieldNames()
	expandSet := make(map[string]struct{}, len(expandKeys))
	for _, k := range expandKeys {
		expandSet[k] = struct{}{}
	}

	out := make(map[string]any)
	for k, v := range row {
		if strings.HasPrefix(k, "@odata") {
			continue
		}
		if _, isMapped := mapped[k]; isMapped {
			continue
		}
		if _, isExpand := expandSet[k]; isExpand {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	b, _ := json.Marshal(out)
	return b
}

func mappedScalarFieldNames() map[string]struct{} {
	names := standardResoFieldNames()
	// Fields populated on ListingRecord but not in standardResoFieldNames.
	extra := []string{
		"BathroomsTotalInteger", "BuildingAreaTotal", "AssociationFee", "AssociationFeeFrequency",
		"AssociationFee2", "AssociationFee2Frequency", "MlsStatus", "ClosePrice", "DockYN",
	}
	for _, n := range extra {
		names[n] = struct{}{}
	}
	return names
}

// MergeMirrorListing rebuilds a flat RESO Property object for API responses.
func MergeMirrorListing(raw json.RawMessage, payloads ExpandedPayload, customFields json.RawMessage) json.RawMessage {
	var root map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &root); err != nil {
			root = map[string]any{}
		}
	} else {
		root = map[string]any{}
	}

	attachJSONB(root, "Media", payloads.Media, payloads.HasMedia)
	attachJSONB(root, "Unit", payloads.Unit, payloads.HasUnit)
	attachJSONB(root, "Room", payloads.Room, payloads.HasRoom)
	attachJSONB(root, "OpenHouse", payloads.OpenHouse, payloads.HasOpenHouse)
	flatMergeCustomFields(root, customFields)

	if len(root) == 0 {
		return raw
	}
	out, err := json.Marshal(root)
	if err != nil {
		return raw
	}
	return out
}

func attachJSONB(root map[string]any, key string, payload json.RawMessage, present bool) {
	if !present || len(payload) == 0 {
		return
	}
	var val any
	if err := json.Unmarshal(payload, &val); err != nil {
		return
	}
	root[key] = val
}

func flatMergeCustomFields(root map[string]any, customFields json.RawMessage) {
	if len(customFields) == 0 {
		return
	}
	var custom map[string]any
	if err := json.Unmarshal(customFields, &custom); err != nil {
		return
	}
	for k, v := range custom {
		if _, exists := root[k]; !exists {
			root[k] = v
		}
	}
}
