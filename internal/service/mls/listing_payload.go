package mls

import (
	"encoding/json"
	"strings"
)

const defaultSyncExpand = "Media,Unit,Room,OpenHouse"

const bridgeNavigationExpandDefault = "OpenHouses,Rooms,UnitTypes"

// BridgeNavigationExpandCSV returns Stellar navigation $expand names (Rooms, UnitTypes, OpenHouses).
// Media is omitted — Bridge inlines Media on full Property; /replication ignores $expand anyway.
func BridgeNavigationExpandCSV(bridgeExpandCSV string) string {
	return strings.Join(BridgeNavigationExpandKeys(bridgeExpandCSV), ",")
}

// BridgeNavigationExpandKeys lists Stellar nav property names from BRIDGE_SYNC_EXPAND.
func BridgeNavigationExpandKeys(bridgeExpandCSV string) []string {
	keys := ParseExpandKeys(bridgeExpandCSV)
	nav := make([]string, 0, 3)
	for _, k := range keys {
		switch k {
		case "OpenHouses", "Rooms", "UnitTypes":
			nav = append(nav, k)
		}
	}
	if len(nav) > 0 {
		return nav
	}
	return ParseExpandKeys(bridgeNavigationExpandDefault)
}

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

// NormalizeBridgeExpandKeys maps Bridge Stellar navigation names to canonical JSONB keys.
// Bridge metadata uses Rooms, UnitTypes, OpenHouses (not Spark's Room, Unit, OpenHouse).
func NormalizeBridgeExpandKeys(row map[string]any) {
	aliases := []struct{ bridge, canonical string }{
		{"Rooms", "Room"},
		{"UnitTypes", "Unit"},
		{"OpenHouses", "OpenHouse"},
	}
	for _, a := range aliases {
		if v, ok := row[a.bridge]; ok {
			if _, has := row[a.canonical]; !has {
				row[a.canonical] = v
			}
		}
	}
}

// navigationCollectionKeys returns all upstream property names for RESO navigation collections.
func navigationCollectionKeys() []string {
	return []string{
		"Media",
		"Unit", "UnitTypes", "Units",
		"Room", "Rooms",
		"OpenHouse", "OpenHouses",
	}
}

func navigationCollectionSet() map[string]struct{} {
	keys := navigationCollectionKeys()
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		set[k] = struct{}{}
	}
	return set
}

func stripNavigationFromMap(m map[string]any) {
	for _, k := range navigationCollectionKeys() {
		delete(m, k)
	}
}

// StripKeysForProvider returns raw_data keys to remove after payload split.
func StripKeysForProvider(provider MirrorProvider, expandKeys []string) []string {
	return navigationStripKeys(provider, expandKeys)
}

// navigationStripKeys is the full key list stripped from raw_data at persist.
func navigationStripKeys(provider MirrorProvider, expandKeys []string) []string {
	seen := make(map[string]struct{}, len(expandKeys)+8)
	var keys []string
	add := func(k string) {
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}
	for _, k := range expandKeys {
		add(k)
	}
	for _, k := range navigationCollectionKeys() {
		add(k)
	}
	_ = provider
	return keys
}

// ExtractExpandedPayloads reads navigation collections from the row using all known aliases.
func ExtractExpandedPayloads(row map[string]any, provider MirrorProvider, expandKeys []string) ExpandedPayload {
	if provider == MirrorProviderBridge {
		NormalizeBridgeExpandKeys(row)
	}
	var p ExpandedPayload
	for _, key := range expandKeys {
		applyNavKeyToPayload(&p, row, key)
	}
	// Always resolve via alias lists so incremental rows without $expand still land in JSONB columns.
	if !p.HasMedia {
		for _, alias := range []string{"Media"} {
			if val, ok := ExtractJSONBField(row, alias); ok {
				p.Media, p.HasMedia = val, true
				break
			}
		}
	}
	if !p.HasUnit {
		for _, alias := range []string{"Unit", "UnitTypes", "Units"} {
			if val, ok := ExtractJSONBField(row, alias); ok {
				p.Unit, p.HasUnit = val, true
				break
			}
		}
	}
	if !p.HasRoom {
		for _, alias := range []string{"Room", "Rooms"} {
			if val, ok := ExtractJSONBField(row, alias); ok {
				p.Room, p.HasRoom = val, true
				break
			}
		}
	}
	if !p.HasOpenHouse {
		for _, alias := range []string{"OpenHouse", "OpenHouses"} {
			if val, ok := ExtractJSONBField(row, alias); ok {
				p.OpenHouse, p.HasOpenHouse = val, true
				break
			}
		}
	}
	return p
}

func applyNavKeyToPayload(p *ExpandedPayload, row map[string]any, key string) {
	val, ok := ExtractJSONBField(row, key)
	if !ok {
		return
	}
	switch key {
	case "Media":
		p.Media, p.HasMedia = val, true
	case "Unit", "UnitTypes", "Units":
		p.Unit, p.HasUnit = val, true
	case "Room", "Rooms":
		p.Room, p.HasRoom = val, true
	case "OpenHouse", "OpenHouses":
		p.OpenHouse, p.HasOpenHouse = val, true
	}
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

// PersistExpandKeys returns expand key names used to split payloads for a provider.
func PersistExpandKeys(provider MirrorProvider, sparkExpand, bridgeExpand string) []string {
	if provider == MirrorProviderSpark {
		return ParseExpandKeys(sparkExpand)
	}
	return ParseExpandKeys(bridgeExpand)
}

// BuildCustomFields collects unmapped RESO keys not stored in typed columns or expanded JSONB.
func BuildCustomFields(row map[string]any, provider MirrorProvider, expandKeys []string, datasetUpper string) json.RawMessage {
	mapped := mappedScalarFieldNames()
	expandSet := navigationCollectionSet()
	for _, k := range navigationStripKeys(provider, expandKeys) {
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
	stripNavigationFromMap(out)
	stripResolvedProviderExtensionsFromCustom(out, provider, datasetUpper)
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
	if val == nil {
		return
	}
	root[key] = val
}

func flatMergeCustomFields(root map[string]any, customFields json.RawMessage) {
	flatMergeCustomFieldsPublic(root, customFields)
}
