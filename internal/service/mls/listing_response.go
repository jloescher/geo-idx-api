package mls

import (
	"encoding/json"
	"strings"
	"time"
)

// MirrorListingColumns is the SELECT column list for mirror-backed API reads (excludes raw_data).
const MirrorListingColumns = `
	dataset_slug, listing_key, mls_listing_id, standard_status, list_price,
	bedrooms_total, bathrooms_total_decimal, living_area, lot_size_acres,
	year_built, stories_total, city, county_or_parish, postal_code, state_or_province,
	property_type, property_sub_type, on_market_date, close_date,
	modification_timestamp, price_change_timestamp, previous_list_price,
	flood_zone_code, estimated_total_monthly_fees,
	fema_flood_zone_code, flood_zone_sfha_tf, low_risk_flood_zone_yn,
	flood_zone_updated_at, fema_failure_reason, fema_attempted_at,
	latitude, longitude,
	waterfront_yn, pool_private_yn, dock_yn, new_construction_yn, garage_yn,
	association_yn, spa_yn, fireplace_yn, senior_community_yn,
	subdivision_name, elementary_school, middle_or_junior_school, high_school,
	special_listing_conditions,
	street_number, street_name, list_agent_mls_id, list_office_mls_id,
	garage_spaces, mls_area_major, days_on_market, tax_annual_amount,
	heating_yn, cooling_yn, carport_yn, attached_garage_yn,
	internet_consumer_comment_yn, internet_address_display_yn,
	internet_entire_listing_display_yn, internet_automated_valuation_display_yn,
	idx_participation_yn, idx_office_participation_yn,
	unparsed_address, public_remarks,
	media, unit, room, open_house, custom_fields`

// MirrorListingSearchColumns is a lean SELECT for POST /search (skips heavy navigation JSONB).
const MirrorListingSearchColumns = `
	dataset_slug, listing_key, mls_listing_id, standard_status, list_price,
	bedrooms_total, bathrooms_total_decimal, living_area, lot_size_acres,
	year_built, stories_total, city, county_or_parish, postal_code, state_or_province,
	property_type, property_sub_type, on_market_date, close_date,
	modification_timestamp, price_change_timestamp, previous_list_price,
	flood_zone_code, estimated_total_monthly_fees,
	fema_flood_zone_code, flood_zone_sfha_tf, low_risk_flood_zone_yn,
	flood_zone_updated_at, fema_failure_reason, fema_attempted_at,
	latitude, longitude,
	waterfront_yn, pool_private_yn, dock_yn, new_construction_yn, garage_yn,
	association_yn, spa_yn, fireplace_yn, senior_community_yn,
	subdivision_name, elementary_school, middle_or_junior_school, high_school,
	special_listing_conditions,
	street_number, street_name, list_agent_mls_id, list_office_mls_id,
	garage_spaces, mls_area_major, days_on_market, tax_annual_amount,
	heating_yn, cooling_yn, carport_yn, attached_garage_yn,
	internet_consumer_comment_yn, internet_address_display_yn,
	internet_entire_listing_display_yn, internet_automated_valuation_display_yn,
	idx_participation_yn, idx_office_participation_yn,
	unparsed_address, public_remarks`

// MirrorListingRow is a listings mirror row for public API assembly (no raw_data).
type MirrorListingRow struct {
	DatasetSlug                         string
	ListingKey                          string
	MlsListingID                        *string
	StandardStatus                      *string
	ListPrice                           float64
	BedroomsTotal                       *int16
	BathroomsTotalDecimal               *float64
	LivingArea                          *float64
	LotSizeAcres                        *float64
	YearBuilt                           *int16
	StoriesTotal                        *int16
	City                                *string
	CountyOrParish                      *string
	PostalCode                          *string
	StateOrProvince                     *string
	PropertyType                        *string
	PropertySubType                     *string
	OnMarketDate                        *time.Time
	CloseDate                           *time.Time
	ModificationTimestamp               *time.Time
	PriceChangeTimestamp                *time.Time
	PreviousListPrice                   *float64
	FloodZoneCode                       *string
	FEMAFloodZoneCode                   *string
	FloodZoneSFHA_TF                    *string
	LowRiskFloodZoneYN                  bool
	FloodZoneUpdatedAt                  *time.Time
	FEMAFailureReason                   *string
	FEMAAttemptedAt                     *time.Time
	EstimatedTotalMonthlyFees           *float64
	Latitude                            *float64
	Longitude                           *float64
	WaterfrontYN                        *bool
	PoolPrivateYN                       *bool
	DockYN                              *bool
	NewConstructionYN                   *bool
	GarageYN                            *bool
	AssociationYN                       *bool
	SpaYN                               *bool
	FireplaceYN                         *bool
	SeniorCommunityYN                   *bool
	SubdivisionName                     *string
	ElementarySchool                    *string
	MiddleOrJuniorSchool                *string
	HighSchool                          *string
	SpecialListingConditions            json.RawMessage
	StreetNumber                        *string
	StreetName                          *string
	ListAgentMlsID                      *string
	ListOfficeMlsID                     *string
	GarageSpaces                        *float64
	MLSAreaMajor                        *string
	DaysOnMarket                        *int32
	TaxAnnualAmount                     *float64
	HeatingYN                           *bool
	CoolingYN                           *bool
	CarportYN                           *bool
	AttachedGarageYN                    *bool
	InternetConsumerCommentYN           *bool
	InternetAddressDisplayYN            *bool
	InternetEntireListingDisplayYN      *bool
	InternetAutomatedValuationDisplayYN *bool
	IDXParticipationYN                  *bool
	IDXOfficeParticipationYN            *bool
	UnparsedAddress                     *string
	PublicRemarks                       *string
	Media                               json.RawMessage
	Unit                                json.RawMessage
	Room                                json.RawMessage
	OpenHouse                           json.RawMessage
	CustomFields                        json.RawMessage
}

// ScanMirrorListingRow reads mirror listing columns from a database row (order matches MirrorListingColumns).
func ScanMirrorListingRow(scan func(dest ...any) error) (MirrorListingRow, error) {
	var row MirrorListingRow
	var media, unit, room, openHouse, custom, special []byte
	err := scan(
		&row.DatasetSlug, &row.ListingKey, &row.MlsListingID, &row.StandardStatus, &row.ListPrice,
		&row.BedroomsTotal, &row.BathroomsTotalDecimal, &row.LivingArea, &row.LotSizeAcres,
		&row.YearBuilt, &row.StoriesTotal, &row.City, &row.CountyOrParish, &row.PostalCode, &row.StateOrProvince,
		&row.PropertyType, &row.PropertySubType, &row.OnMarketDate, &row.CloseDate,
		&row.ModificationTimestamp, &row.PriceChangeTimestamp, &row.PreviousListPrice,
		&row.FloodZoneCode, &row.FEMAFloodZoneCode, &row.FloodZoneSFHA_TF, &row.LowRiskFloodZoneYN,
		&row.FloodZoneUpdatedAt, &row.FEMAFailureReason, &row.FEMAAttemptedAt,
		&row.EstimatedTotalMonthlyFees,
		&row.Latitude, &row.Longitude,
		&row.WaterfrontYN, &row.PoolPrivateYN, &row.DockYN, &row.NewConstructionYN, &row.GarageYN,
		&row.AssociationYN, &row.SpaYN, &row.FireplaceYN, &row.SeniorCommunityYN,
		&row.SubdivisionName, &row.ElementarySchool, &row.MiddleOrJuniorSchool, &row.HighSchool,
		&special,
		&row.StreetNumber, &row.StreetName, &row.ListAgentMlsID, &row.ListOfficeMlsID,
		&row.GarageSpaces, &row.MLSAreaMajor, &row.DaysOnMarket, &row.TaxAnnualAmount,
		&row.HeatingYN, &row.CoolingYN, &row.CarportYN, &row.AttachedGarageYN,
		&row.InternetConsumerCommentYN, &row.InternetAddressDisplayYN,
		&row.InternetEntireListingDisplayYN, &row.InternetAutomatedValuationDisplayYN,
		&row.IDXParticipationYN, &row.IDXOfficeParticipationYN,
		&row.UnparsedAddress, &row.PublicRemarks,
		&media, &unit, &room, &openHouse, &custom,
	)
	if err != nil {
		return MirrorListingRow{}, err
	}
	row.SpecialListingConditions = json.RawMessage(special)
	row.Media = json.RawMessage(media)
	row.Unit = json.RawMessage(unit)
	row.Room = json.RawMessage(room)
	row.OpenHouse = json.RawMessage(openHouse)
	row.CustomFields = json.RawMessage(custom)
	return row, nil
}

// ScanMirrorListingSearchRow reads lean search columns (order matches MirrorListingSearchColumns).
func ScanMirrorListingSearchRow(scan func(dest ...any) error) (MirrorListingRow, error) {
	var row MirrorListingRow
	var special []byte
	err := scan(
		&row.DatasetSlug, &row.ListingKey, &row.MlsListingID, &row.StandardStatus, &row.ListPrice,
		&row.BedroomsTotal, &row.BathroomsTotalDecimal, &row.LivingArea, &row.LotSizeAcres,
		&row.YearBuilt, &row.StoriesTotal, &row.City, &row.CountyOrParish, &row.PostalCode, &row.StateOrProvince,
		&row.PropertyType, &row.PropertySubType, &row.OnMarketDate, &row.CloseDate,
		&row.ModificationTimestamp, &row.PriceChangeTimestamp, &row.PreviousListPrice,
		&row.FloodZoneCode, &row.FEMAFloodZoneCode, &row.FloodZoneSFHA_TF, &row.LowRiskFloodZoneYN,
		&row.FloodZoneUpdatedAt, &row.FEMAFailureReason, &row.FEMAAttemptedAt,
		&row.EstimatedTotalMonthlyFees,
		&row.Latitude, &row.Longitude,
		&row.WaterfrontYN, &row.PoolPrivateYN, &row.DockYN, &row.NewConstructionYN, &row.GarageYN,
		&row.AssociationYN, &row.SpaYN, &row.FireplaceYN, &row.SeniorCommunityYN,
		&row.SubdivisionName, &row.ElementarySchool, &row.MiddleOrJuniorSchool, &row.HighSchool,
		&special,
		&row.StreetNumber, &row.StreetName, &row.ListAgentMlsID, &row.ListOfficeMlsID,
		&row.GarageSpaces, &row.MLSAreaMajor, &row.DaysOnMarket, &row.TaxAnnualAmount,
		&row.HeatingYN, &row.CoolingYN, &row.CarportYN, &row.AttachedGarageYN,
		&row.InternetConsumerCommentYN, &row.InternetAddressDisplayYN,
		&row.InternetEntireListingDisplayYN, &row.InternetAutomatedValuationDisplayYN,
		&row.IDXParticipationYN, &row.IDXOfficeParticipationYN,
		&row.UnparsedAddress, &row.PublicRemarks,
	)
	if err != nil {
		return MirrorListingRow{}, err
	}
	row.SpecialListingConditions = json.RawMessage(special)
	return row, nil
}

type listingJSONOpts struct {
	includeExpanded bool
}

// BuildPublicListingJSON assembles a flat RESO Property from typed columns, navigation JSONB, and custom_fields.
func BuildPublicListingJSON(row MirrorListingRow) json.RawMessage {
	root := buildPublicListingMap(row, listingJSONOpts{includeExpanded: true})
	SanitizePublicListingMap(root)
	omitNullTopLevel(root)
	out, err := json.Marshal(root)
	if err != nil {
		return json.RawMessage("{}")
	}
	return out
}

func buildPublicListingMap(row MirrorListingRow, opts listingJSONOpts) map[string]any {
	root := map[string]any{}

	putString(root, "ListingKey", row.ListingKey)
	putStringPtr(root, "ListingId", row.MlsListingID)
	putStringPtr(root, "StandardStatus", row.StandardStatus)
	root["ListPrice"] = row.ListPrice
	putInt16Ptr(root, "BedroomsTotal", row.BedroomsTotal)
	putFloatPtr(root, "BathroomsTotalDecimal", row.BathroomsTotalDecimal)
	putFloatPtr(root, "LivingArea", row.LivingArea)
	putFloatPtr(root, "LotSizeAcres", row.LotSizeAcres)
	putInt16Ptr(root, "YearBuilt", row.YearBuilt)
	putInt16Ptr(root, "StoriesTotal", row.StoriesTotal)
	putStringPtr(root, "City", row.City)
	putStringPtr(root, "CountyOrParish", row.CountyOrParish)
	putStringPtr(root, "PostalCode", row.PostalCode)
	putStringPtr(root, "StateOrProvince", row.StateOrProvince)
	putStringPtr(root, "PropertyType", row.PropertyType)
	putStringPtr(root, "PropertySubType", row.PropertySubType)
	putDatePtr(root, "OnMarketDate", row.OnMarketDate)
	putDatePtr(root, "CloseDate", row.CloseDate)
	putTimestampPtr(root, "ModificationTimestamp", row.ModificationTimestamp)
	putTimestampPtr(root, "PriceChangeTimestamp", row.PriceChangeTimestamp)
	putFloatPtr(root, "PreviousListPrice", row.PreviousListPrice)
	putStringPtr(root, "FloodZoneCode", row.FloodZoneCode)
	floodZone := BuildFloodZoneResponse(row)
	root["flood_zone"] = floodZone
	putFloatPtr(root, "EstimatedTotalMonthlyFees", row.EstimatedTotalMonthlyFees)
	putFloatPtr(root, "Latitude", row.Latitude)
	putFloatPtr(root, "Longitude", row.Longitude)
	putBoolPtr(root, "WaterfrontYN", row.WaterfrontYN)
	putBoolPtr(root, "PoolPrivateYN", row.PoolPrivateYN)
	putBoolPtr(root, "DockYN", row.DockYN)
	putBoolPtr(root, "NewConstructionYN", row.NewConstructionYN)
	putBoolPtr(root, "GarageYN", row.GarageYN)
	putBoolPtr(root, "AssociationYN", row.AssociationYN)
	putBoolPtr(root, "SpaYN", row.SpaYN)
	putBoolPtr(root, "FireplaceYN", row.FireplaceYN)
	putBoolPtr(root, "SeniorCommunityYN", row.SeniorCommunityYN)
	putStringPtr(root, "SubdivisionName", row.SubdivisionName)
	putStringPtr(root, "ElementarySchool", row.ElementarySchool)
	putStringPtr(root, "MiddleOrJuniorSchool", row.MiddleOrJuniorSchool)
	putStringPtr(root, "HighSchool", row.HighSchool)
	putStringPtr(root, "StreetNumber", row.StreetNumber)
	putStringPtr(root, "StreetName", row.StreetName)
	putStringPtr(root, "ListAgentMlsId", row.ListAgentMlsID)
	putStringPtr(root, "ListOfficeMlsId", row.ListOfficeMlsID)
	putFloatPtr(root, "GarageSpaces", row.GarageSpaces)
	putStringPtr(root, "MLSAreaMajor", row.MLSAreaMajor)
	putInt32Ptr(root, "DaysOnMarket", row.DaysOnMarket)
	putFloatPtr(root, "TaxAnnualAmount", row.TaxAnnualAmount)
	putBoolPtr(root, "HeatingYN", row.HeatingYN)
	putBoolPtr(root, "CoolingYN", row.CoolingYN)
	putBoolPtr(root, "CarportYN", row.CarportYN)
	putBoolPtr(root, "AttachedGarageYN", row.AttachedGarageYN)
	putBoolPtr(root, "InternetConsumerCommentYN", row.InternetConsumerCommentYN)
	putBoolPtr(root, "InternetAddressDisplayYN", row.InternetAddressDisplayYN)
	putBoolPtr(root, "InternetEntireListingDisplayYN", row.InternetEntireListingDisplayYN)
	putBoolPtr(root, "InternetAutomatedValuationDisplayYN", row.InternetAutomatedValuationDisplayYN)
	putBoolPtr(root, "IDXParticipationYN", row.IDXParticipationYN)
	putBoolPtr(root, "IDXOfficeParticipationYN", row.IDXOfficeParticipationYN)
	putStringPtr(root, "UnparsedAddress", row.UnparsedAddress)
	putStringPtr(root, "PublicRemarks", row.PublicRemarks)
	attachRawJSON(root, "SpecialListingConditions", row.SpecialListingConditions)

	if opts.includeExpanded {
		payloads := ExpandedPayload{
			Media:        row.Media,
			Unit:         row.Unit,
			Room:         row.Room,
			OpenHouse:    row.OpenHouse,
			HasMedia:     len(row.Media) > 0,
			HasUnit:      len(row.Unit) > 0,
			HasRoom:      len(row.Room) > 0,
			HasOpenHouse: len(row.OpenHouse) > 0,
		}
		attachJSONB(root, "Media", payloads.Media, payloads.HasMedia)
		attachJSONB(root, "Unit", payloads.Unit, payloads.HasUnit)
		attachJSONB(root, "Room", payloads.Room, payloads.HasRoom)
		attachJSONB(root, "OpenHouse", payloads.OpenHouse, payloads.HasOpenHouse)
		flatMergeCustomFieldsPublic(root, row.CustomFields)
	}
	return root
}

// BuildPublicListingJSONForSearch builds JSON and applies public search visibility rules.
func BuildPublicListingJSONForSearch(row MirrorListingRow) (json.RawMessage, bool) {
	root := buildPublicListingMap(row, listingJSONOpts{includeExpanded: false})
	flags := IDXFlagsFromMirrorRow(row)
	if !ApplyPublicListingVisibility(root, flags, VisibilityPublicSearch) {
		return nil, false
	}
	SanitizePublicListingMap(root)
	omitNullTopLevel(root)
	out, err := json.Marshal(root)
	if err != nil {
		return json.RawMessage("{}"), true
	}
	return out, true
}

// SanitizePublicListingMap removes internal and OData metadata keys from a Property object.
func SanitizePublicListingMap(root map[string]any) map[string]any {
	if root == nil {
		return map[string]any{}
	}
	for k := range root {
		if isForbiddenPublicKey(k) {
			delete(root, k)
		}
	}
	return root
}

// SanitizeUpstreamPropertyJSON strips @odata.* and applies public search visibility rules.
func SanitizeUpstreamPropertyJSON(body json.RawMessage) json.RawMessage {
	return SanitizeUpstreamPropertyJSONWithDataset(body, "")
}

// SanitizeUpstreamPropertyJSONForInternal strips @odata.* only (full MLS payload for comps/BPO).
func SanitizeUpstreamPropertyJSONForInternal(body json.RawMessage) json.RawMessage {
	if len(body) == 0 {
		return body
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return body
	}
	NormalizeBridgeExpandKeys(root)
	stripBridgeNavigationAliases(root)
	SanitizePublicListingMap(root)
	omitNullTopLevel(root)
	out, err := json.Marshal(root)
	if err != nil {
		return body
	}
	return out
}

// SanitizeUpstreamPropertyJSONWithDataset is SanitizeUpstreamPropertyJSON with dataset for IDX gates.
func SanitizeUpstreamPropertyJSONWithDataset(body json.RawMessage, datasetSlug string) json.RawMessage {
	if len(body) == 0 {
		return body
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return body
	}
	NormalizeBridgeExpandKeys(root)
	stripBridgeNavigationAliases(root)
	SanitizePublicListingMap(root)
	flags := IDXFlagsFromMap(root, datasetSlug)
	ApplyPublicListingVisibility(root, flags, VisibilityPublicSearch)
	omitNullTopLevel(root)
	out, err := json.Marshal(root)
	if err != nil {
		return body
	}
	return out
}

func stripBridgeNavigationAliases(root map[string]any) {
	aliases := []struct{ bridge, canonical string }{
		{"Rooms", "Room"},
		{"UnitTypes", "Unit"},
		{"OpenHouses", "OpenHouse"},
	}
	for _, a := range aliases {
		if _, has := root[a.canonical]; has {
			delete(root, a.bridge)
		}
	}
}

func isForbiddenPublicKey(k string) bool {
	if k == "raw_data" || k == "custom_fields" {
		return true
	}
	return strings.HasPrefix(k, "@odata")
}

func flatMergeCustomFieldsPublic(root map[string]any, customFields json.RawMessage) {
	if len(customFields) == 0 {
		return
	}
	var custom map[string]any
	if err := json.Unmarshal(customFields, &custom); err != nil {
		return
	}
	nav := navigationCollectionSet()
	for k, v := range custom {
		if isForbiddenPublicKey(k) {
			continue
		}
		if _, isNav := nav[k]; isNav {
			continue
		}
		if isJSONNull(v) {
			continue
		}
		if shouldSkipCustomMergeKey(root, k) {
			continue
		}
		root[k] = v
	}
}

func putString(m map[string]any, key, val string) {
	if val != "" {
		m[key] = val
	}
}

func putStringPtr(m map[string]any, key string, val *string) {
	if val != nil && *val != "" {
		m[key] = *val
	}
}

func putFloatPtr(m map[string]any, key string, val *float64) {
	if val != nil {
		m[key] = *val
	}
}

func putInt16Ptr(m map[string]any, key string, val *int16) {
	if val != nil {
		m[key] = *val
	}
}

func putInt32Ptr(m map[string]any, key string, val *int32) {
	if val != nil {
		m[key] = *val
	}
}

func putBoolPtr(m map[string]any, key string, val *bool) {
	if val != nil {
		m[key] = *val
	}
}

func putDatePtr(m map[string]any, key string, val *time.Time) {
	if val != nil {
		m[key] = val.Format("2006-01-02")
	}
}

func putTimestampPtr(m map[string]any, key string, val *time.Time) {
	if val != nil {
		m[key] = val.UTC().Format(time.RFC3339)
	}
}

func attachRawJSON(root map[string]any, key string, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var val any
	if err := json.Unmarshal(raw, &val); err != nil {
		return
	}
	if isJSONNull(val) {
		return
	}
	root[key] = val
}
