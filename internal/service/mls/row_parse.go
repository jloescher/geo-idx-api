package mls

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func numericValue(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}

func int16Ptr(v any) *int16 {
	f, ok := numericValue(v)
	if !ok {
		return nil
	}
	i := int16(math.Round(f))
	return &i
}

func int32Ptr(v any) *int32 {
	f, ok := numericValue(v)
	if !ok {
		return nil
	}
	i := int32(math.Round(f))
	return &i
}

func float64Ptr(v any) *float64 {
	f, ok := numericValue(v)
	if !ok {
		return nil
	}
	return &f
}

func boolPtr(v any) *bool {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case bool:
		return &t
	default:
		return nil
	}
}

func datePtr(v any) *time.Time {
	s := stringValue(v)
	if s == "" {
		return nil
	}
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			utc := t.UTC()
			d := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
			return &d
		}
	}
	return nil
}

func timestampPtr(v any) *time.Time {
	s := stringValue(v)
	if s == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.999Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			utc := t.UTC()
			return &utc
		}
	}
	return nil
}

func resolveLatLng(row map[string]any) (lat, lng float64, ok bool) {
	if coords, okMap := row["Coordinates"].(map[string]any); okMap {
		if pair, okArr := coords["coordinates"].([]any); okArr && len(pair) >= 2 {
			lngV, lngOK := numericValue(pair[0])
			latV, latOK := numericValue(pair[1])
			if lngOK && latOK && !math.IsNaN(latV) && !math.IsNaN(lngV) {
				return latV, lngV, true
			}
		}
	}
	latV, latOK := numericValue(row["Latitude"])
	lngV, lngOK := numericValue(row["Longitude"])
	if latOK && lngOK && !math.IsNaN(latV) && !math.IsNaN(lngV) {
		return latV, lngV, true
	}
	return 0, 0, false
}

func datasetSlugFromListingKey(listingKey, fallbackDataset string) string {
	if idx := strings.Index(listingKey, ":"); idx > 0 {
		return strings.ToLower(listingKey[:idx])
	}
	return strings.ToLower(fallbackDataset)
}

func specialListingConditions(row map[string]any) json.RawMessage {
	raw, ok := row["SpecialListingConditions"]
	if !ok {
		return json.RawMessage("[]")
	}
	switch t := raw.(type) {
	case []any:
		var out []string
		for _, item := range t {
			if s := stringValue(item); s != "" {
				out = append(out, s)
			}
		}
		b, _ := json.Marshal(out)
		return b
	default:
		b, err := json.Marshal(raw)
		if err != nil {
			return json.RawMessage("[]")
		}
		return b
	}
}

func standardResoFieldNames() map[string]struct{} {
	names := []string{
		"ListingKey", "ListingId", "StandardStatus", "ListPrice", "ClosePrice", "PreviousListPrice",
		"PriceChangeTimestamp", "ModificationTimestamp", "BridgeModificationTimestamp",
		"BedroomsTotal", "BathroomsTotalDecimal", "BathroomsTotalInteger", "LivingArea", "BuildingAreaTotal", "LotSizeAcres",
		"AssociationFee", "AssociationFeeFrequency", "AssociationFee2", "AssociationFee2Frequency",
		"YearBuilt", "StoriesTotal", "City", "CountyOrParish", "PostalCode", "StateOrProvince",
		"PropertyType", "PropertySubType", "OnMarketDate", "CloseDate", "Latitude", "Longitude",
		"Coordinates", "WaterfrontYN", "PoolPrivateYN", "DockYN", "NewConstructionYN", "GarageYN",
		"AssociationYN", "SpaYN", "FireplaceYN", "SeniorCommunityYN", "SpecialListingConditions",
		"SubdivisionName", "ElementarySchool", "MiddleOrJuniorSchool", "HighSchool",
		"StreetNumber", "StreetName", "ListAgentMlsId", "ListOfficeMlsId", "MlsStatus",
	}
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}
