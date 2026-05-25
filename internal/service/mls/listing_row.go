package mls

import (
	"encoding/json"
	"strings"
	"time"
)

// RowAction is how persist should treat a RESO row.
type RowAction string

const (
	RowActionUpsert RowAction = "upsert"
	RowActionDelete RowAction = "delete"
	RowActionSkip   RowAction = "skip"
)

// ListingRecord is a normalized row for listings mirror upsert.
type ListingRecord struct {
	DatasetSlug               string
	ListingKey                string
	MlsListingID              *string
	StandardStatus            *string
	ListPrice                 *float64
	BedroomsTotal             *int16
	BathroomsTotalDecimal     *float64
	LivingArea                *float64
	LotSizeAcres              *float64
	YearBuilt                 *int16
	StoriesTotal              *int16
	City                      *string
	CountyOrParish            *string
	PostalCode                *string
	StateOrProvince           *string
	PropertyType              *string
	PropertySubType           *string
	OnMarketDate              *time.Time
	CloseDate                 *time.Time
	ModificationTimestamp     *time.Time
	PriceChangeTimestamp      *time.Time
	PreviousListPrice         *float64
	FloodZoneCode             *string
	LowRiskFloodZoneYN        bool
	EstimatedTotalMonthlyFees *float64
	Latitude                  *float64
	Longitude                 *float64
	WaterfrontYN              *bool
	PoolPrivateYN             *bool
	DockYN                    *bool
	NewConstructionYN         *bool
	GarageYN                  *bool
	AssociationYN             *bool
	SpaYN                     *bool
	FireplaceYN               *bool
	SeniorCommunityYN         *bool
	SubdivisionName           *string
	ElementarySchool          *string
	MiddleOrJuniorSchool      *string
	HighSchool                *string
	SpecialListingConditions  json.RawMessage
	RawData                   json.RawMessage
	Media                     json.RawMessage
	Unit                      json.RawMessage
	Room                      json.RawMessage
	OpenHouse                 json.RawMessage
	HasMedia                  bool
	HasUnit                   bool
	HasRoom                   bool
	HasOpenHouse              bool
	CustomFields              json.RawMessage
	StreetNumber              *string
	StreetName                *string
	ListAgentMlsID            *string
	ListOfficeMlsID           *string
}

// BuildListingRecord maps a RESO Property row to mirror columns.
func BuildListingRecord(
	replicationDataset string,
	provider MirrorProvider,
	row map[string]any,
	raw json.RawMessage,
	resolver *ResoFieldResolver,
	expandKeys []string,
) (ListingRecord, RowAction) {
	listingKey := stringValue(row["ListingKey"])
	if listingKey == "" {
		return ListingRecord{}, RowActionSkip
	}

	datasetSlug := datasetSlugFromListingKey(listingKey, replicationDataset)
	statusNorm := strings.ToLower(strings.TrimSpace(stringValue(row["StandardStatus"])))
	if statusNorm != "active" && statusNorm != "pending" {
		return ListingRecord{DatasetSlug: datasetSlug, ListingKey: listingKey}, RowActionDelete
	}

	datasetUpper := strings.ToUpper(replicationDataset)
	lat, lng, hasCoords := resolveLatLng(row)
	var latPtr, lngPtr *float64
	if hasCoords {
		latPtr = &lat
		lngPtr = &lng
	}

	var mlsID *string
	if s := stringValue(row["ListingId"]); s != "" {
		mlsID = &s
	}

	if len(expandKeys) == 0 {
		expandKeys = ParseExpandKeys("")
	}

	payloads := ExtractExpandedPayloads(row, provider, expandKeys)
	storedRaw := StripJSONBKeysFromRaw(raw, StripKeysForProvider(provider, expandKeys))
	custom := BuildCustomFields(row, provider, expandKeys)

	stdStatus := stringValue(row["StandardStatus"])
	var stdPtr *string
	if stdStatus != "" {
		stdPtr = &stdStatus
	}

	floodZoneCode := resolver.ResolveFloodZoneCode(row, provider, datasetUpper)

	listPrice, hasPrice := ResolveListPrice(row)
	if !hasPrice {
		return ListingRecord{}, RowActionSkip
	}

	rec := ListingRecord{
		DatasetSlug:               datasetSlug,
		ListingKey:                  listingKey,
		MlsListingID:                mlsID,
		StandardStatus:              stdPtr,
		ListPrice:                   &listPrice,
		BedroomsTotal:               BoundedInt16Ptr(row["BedroomsTotal"]),
		BathroomsTotalDecimal:       BathroomsTotal(row),
		LivingArea:                  ResolveLivingAreaSqft(row),
		LotSizeAcres:                ClampNumeric12_4Ptr(row["LotSizeAcres"]),
		YearBuilt:                   BoundedInt16Ptr(row["YearBuilt"]),
		StoriesTotal:                BoundedInt16Ptr(row["StoriesTotal"]),
		City:                        optionalString(row["City"]),
		CountyOrParish:              optionalString(row["CountyOrParish"]),
		PostalCode:                  optionalString(row["PostalCode"]),
		StateOrProvince:             optionalString(row["StateOrProvince"]),
		PropertyType:                optionalString(row["PropertyType"]),
		PropertySubType:             optionalString(row["PropertySubType"]),
		OnMarketDate:                datePtr(row["OnMarketDate"]),
		CloseDate:                   datePtr(row["CloseDate"]),
		ModificationTimestamp:       ResolveModificationTimestamp(datasetSlug, row),
		PriceChangeTimestamp:        timestampPtr(row["PriceChangeTimestamp"]),
		PreviousListPrice:           ClampNumeric14_2Ptr(row["PreviousListPrice"]),
		FloodZoneCode:               floodZoneCode,
		LowRiskFloodZoneYN:          false, // set by FEMA enrichment (fema_flood_zone_code), not MLS
		EstimatedTotalMonthlyFees:   resolver.ResolveEstimatedTotalMonthlyFees(row, provider, datasetUpper),
		Latitude:                    latPtr,
		Longitude:                   lngPtr,
		WaterfrontYN:                boolPtr(row["WaterfrontYN"]),
		PoolPrivateYN:               boolPtr(row["PoolPrivateYN"]),
		DockYN:                      boolPtr(row["DockYN"]),
		NewConstructionYN:           boolPtr(row["NewConstructionYN"]),
		GarageYN:                    boolPtr(row["GarageYN"]),
		AssociationYN:               boolPtr(row["AssociationYN"]),
		SpaYN:                       boolPtr(row["SpaYN"]),
		FireplaceYN:                 boolPtr(row["FireplaceYN"]),
		SeniorCommunityYN:           boolPtr(row["SeniorCommunityYN"]),
		SubdivisionName:             optionalString(row["SubdivisionName"]),
		ElementarySchool:            optionalString(row["ElementarySchool"]),
		MiddleOrJuniorSchool:        optionalString(row["MiddleOrJuniorSchool"]),
		HighSchool:                  optionalString(row["HighSchool"]),
		SpecialListingConditions:    specialListingConditions(row),
		RawData:                     storedRaw,
		Media:                       payloads.Media,
		Unit:                        payloads.Unit,
		Room:                        payloads.Room,
		OpenHouse:                   payloads.OpenHouse,
		HasMedia:                    payloads.HasMedia,
		HasUnit:                     payloads.HasUnit,
		HasRoom:                     payloads.HasRoom,
		HasOpenHouse:                payloads.HasOpenHouse,
		CustomFields:                custom,
		StreetNumber:                optionalString(row["StreetNumber"]),
		StreetName:                  optionalString(row["StreetName"]),
		ListAgentMlsID:              optionalString(row["ListAgentMlsId"]),
		ListOfficeMlsID:             optionalString(row["ListOfficeMlsId"]),
	}
	return rec, RowActionUpsert
}

func optionalString(v any) *string {
	s := stringValue(v)
	if s == "" {
		return nil
	}
	return &s
}
