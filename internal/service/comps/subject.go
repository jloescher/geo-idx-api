package comps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func (e *Engine) resolveSubject(ctx context.Context, dataset string, in SubjectInput) (SubjectProfile, error) {
	in = mergeSubjectAliases(in)
	typ := strings.ToLower(strings.TrimSpace(in.Type))
	if typ == "" {
		typ = "mls"
	}
	switch typ {
	case "mls":
		return e.subjectFromMLS(ctx, dataset, in)
	case "off_market":
		return subjectFromOffMarket(in)
	default:
		return SubjectProfile{}, fmt.Errorf("subject.type must be mls or off_market")
	}
}

func mergeSubjectAliases(in SubjectInput) SubjectInput {
	if in.FloodZoneCode == nil && in.StellarFloodZoneCode != nil {
		in.FloodZoneCode = in.StellarFloodZoneCode
	}
	if in.MonthlyFees == nil && in.StellarTotalMonthlyFees != nil {
		in.MonthlyFees = in.StellarTotalMonthlyFees
	}
	return in
}

func (e *Engine) subjectFromMLS(ctx context.Context, dataset string, in SubjectInput) (SubjectProfile, error) {
	ds, key := parseListingRef(in.ListingID, dataset)
	if key == "" {
		return SubjectProfile{}, fmt.Errorf("subject.listing_id is required for mls subject")
	}
	var raw, media, unit, room, openHouse, custom []byte
	var listPrice *float64
	pool, err := e.db.ReadPool(ctx)
	if err != nil {
		return SubjectProfile{}, err
	}
	err = pool.QueryRow(ctx, `
		SELECT raw_data, media, unit, room, open_house, custom_fields, list_price
		FROM listings WHERE dataset_slug = $1 AND listing_key = $2
	`, ds, key).Scan(&raw, &media, &unit, &room, &openHouse, &custom, &listPrice)
	if err != nil {
		return SubjectProfile{}, fmt.Errorf("subject listing not found in mirror: %s", key)
	}
	merged := mls.MergeMirrorListing(json.RawMessage(raw), mls.ExpandedPayload{
		Media: json.RawMessage(media), Unit: json.RawMessage(unit),
		Room: json.RawMessage(room), OpenHouse: json.RawMessage(openHouse),
		HasMedia: len(media) > 0, HasUnit: len(unit) > 0, HasRoom: len(room) > 0, HasOpenHouse: len(openHouse) > 0,
	}, json.RawMessage(custom))
	comp := parseProperty(merged)
	sub := SubjectProfile{
		Type:         "mls",
		ListingKey:   key,
		ListingID:    in.ListingID,
		Lat:          comp.Lat,
		Lng:          comp.Lng,
		Bedrooms:     comp.Bedrooms,
		Bathrooms:    comp.Bathrooms,
		LivingArea:   comp.LivingArea,
		LotSizeAcres: comp.LotSizeAcres,
		YearBuilt:    comp.YearBuilt,
		GarageSpaces: comp.GarageSpaces,
		PoolPrivate:  comp.PoolPrivate,
		Waterfront:   comp.Waterfront,
		MonthlyFees:  comp.MonthlyFees,
		FloodZone:    comp.FloodZone,
		Raw:          merged,
	}
	if listPrice != nil {
		sub.ListPrice = *listPrice
	}
	if in.Lat != nil {
		sub.Lat = *in.Lat
	}
	if in.Lng != nil {
		sub.Lng = *in.Lng
	}
	applySubjectOverrides(&sub, in)
	if sub.Lat == 0 || sub.Lng == 0 {
		return SubjectProfile{}, fmt.Errorf("subject listing has no coordinates")
	}
	return sub, nil
}

func subjectFromOffMarket(in SubjectInput) (SubjectProfile, error) {
	if in.Lat == nil || in.Lng == nil {
		return SubjectProfile{}, fmt.Errorf("subject.lat and subject.lng are required for off_market")
	}
	sub := SubjectProfile{
		Type: "off_market",
		Lat:  *in.Lat,
		Lng:  *in.Lng,
	}
	applySubjectOverrides(&sub, in)
	if sub.Bedrooms <= 0 || sub.LivingArea <= 0 {
		return SubjectProfile{}, fmt.Errorf("off_market subject requires bedrooms and living_area_sqft")
	}
	return sub, nil
}

func applySubjectOverrides(sub *SubjectProfile, in SubjectInput) {
	if in.Bedrooms != nil {
		sub.Bedrooms = *in.Bedrooms
	}
	if in.Bathrooms != nil {
		sub.Bathrooms = *in.Bathrooms
	}
	if in.LivingAreaSqft != nil {
		sub.LivingArea = *in.LivingAreaSqft
	}
	if in.LotSizeSqft != nil && *in.LotSizeSqft > 0 {
		sub.LotSizeAcres = *in.LotSizeSqft / 43560.0
	}
	if in.GarageSpaces != nil {
		sub.GarageSpaces = *in.GarageSpaces
	}
	if in.MonthlyFees != nil {
		sub.MonthlyFees = *in.MonthlyFees
	}
	if in.FloodZoneCode != nil {
		sub.FloodZone = *in.FloodZoneCode
	}
	if in.SubdivisionName != nil {
		sub.Subdivision = *in.SubdivisionName
	}
	if in.MLSAreaMajor != nil {
		sub.MLSAreaMajor = *in.MLSAreaMajor
	}
	if in.PropertyType != nil {
		sub.PropertyType = *in.PropertyType
	}
	if in.YearBuilt != nil {
		sub.YearBuilt = *in.YearBuilt
	}
	if in.Condition != nil {
		sub.Condition = *in.Condition
	}
	if in.RenovatedKitchenYear != nil {
		sub.RenovatedKitchenYear = *in.RenovatedKitchenYear
	}
	if in.RenovatedBathroomsYear != nil {
		sub.RenovatedBathroomsYear = *in.RenovatedBathroomsYear
	}
	if in.RenovatedHVACYear != nil {
		sub.RenovatedHVACYear = *in.RenovatedHVACYear
	}
}

func parseListingRef(id, defaultDataset string) (dataset, key string) {
	id = strings.TrimSpace(id)
	if i := strings.Index(id, ":"); i > 0 {
		return normalizeDataset(id[:i]), id[i+1:]
	}
	return defaultDataset, id
}

func normalizeDataset(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "bridge_")
	s = strings.TrimPrefix(s, "spark_")
	return s
}
