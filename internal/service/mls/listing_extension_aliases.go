package mls

import "strings"

// providerExtensionAliasCanonical returns the typed-column RESO key when extKey duplicates indexed data.
func providerExtensionAliasCanonical(extKey string) string {
	switch extKey {
	case BeachesSparkFloodZoneField:
		return "FloodZoneCode"
	}
	if strings.HasSuffix(extKey, "_FloodZoneCode") {
		return "FloodZoneCode"
	}
	if strings.HasSuffix(extKey, "_TotalMonthlyFees") {
		return "EstimatedTotalMonthlyFees"
	}
	return ""
}

// resolvedProviderExtensionKeys lists upstream extension field names copied into typed mirror columns.
func resolvedProviderExtensionKeys(provider MirrorProvider, datasetUpper string) []string {
	keys := []string{BeachesSparkFloodZoneField, sparkEstimatedMonthlyFallback}
	if provider == MirrorProviderBridge && datasetUpper != "" {
		prefix := strings.ToUpper(datasetUpper)
		keys = append(keys, prefix+"_FloodZoneCode", prefix+"_TotalMonthlyFees")
	}
	return keys
}

func stripResolvedProviderExtensionsFromCustom(out map[string]any, provider MirrorProvider, datasetUpper string) {
	for _, k := range resolvedProviderExtensionKeys(provider, datasetUpper) {
		delete(out, k)
	}
}

// publicListingTypedKeys are RESO keys emitted from mirror typed columns in BuildPublicListingJSON.
func publicListingTypedKeys() map[string]struct{} {
	keys := []string{
		"ListingKey", "ListingId", "StandardStatus", "ListPrice",
		"BedroomsTotal", "BathroomsTotalDecimal", "LivingArea", "LotSizeAcres",
		"YearBuilt", "StoriesTotal", "City", "CountyOrParish", "PostalCode", "StateOrProvince",
		"PropertyType", "PropertySubType", "OnMarketDate", "CloseDate",
		"ModificationTimestamp", "PriceChangeTimestamp", "PreviousListPrice",
		"FloodZoneCode", "EstimatedTotalMonthlyFees", "Latitude", "Longitude",
		"WaterfrontYN", "PoolPrivateYN", "DockYN", "NewConstructionYN", "GarageYN",
		"AssociationYN", "SpaYN", "FireplaceYN", "SeniorCommunityYN",
		"SubdivisionName", "ElementarySchool", "MiddleOrJuniorSchool", "HighSchool",
		"SpecialListingConditions",
		"StreetNumber", "StreetName", "ListAgentMlsId", "ListOfficeMlsId",
	}
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		set[k] = struct{}{}
	}
	return set
}

func shouldSkipCustomMergeKey(root map[string]any, key string) bool {
	if _, exists := root[key]; exists {
		return true
	}
	if canonical := providerExtensionAliasCanonical(key); canonical != "" {
		if _, exists := root[canonical]; exists {
			return true
		}
	}
	if _, isTyped := publicListingTypedKeys()[key]; isTyped {
		if _, exists := root[key]; exists {
			return true
		}
	}
	if _, isMapped := mappedScalarFieldNames()[key]; isMapped {
		if _, exists := root[key]; exists {
			return true
		}
	}
	return false
}

func isJSONNull(v any) bool {
	return v == nil
}

func omitNullTopLevel(root map[string]any) {
	for k, v := range root {
		if isJSONNull(v) {
			delete(root, k)
		}
	}
}
