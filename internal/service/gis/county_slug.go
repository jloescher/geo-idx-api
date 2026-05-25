package gis

import "strings"

// CountyNameToSlug normalizes FDOT/census county names to gis_parcels.county slugs.
func CountyNameToSlug(name string) string {
	s := strings.TrimSpace(name)
	s = strings.TrimSuffix(s, " County")
	s = strings.ReplaceAll(s, "St. ", "St ")
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ".", "")
	return s
}

// MLSStellarSlugs is the set of counties covered by Stellar MLS parcel sync.
var MLSStellarSlugs = map[string]bool{
	"alachua": true, "charlotte": true, "desoto": true, "flagler": true,
	"hillsborough": true, "lake": true, "manatee": true, "marion": true,
	"okeechobee": true, "orange": true, "osceola": true, "pasco": true,
	"pinellas": true, "polk": true, "sarasota": true, "sumter": true, "volusia": true,
}

// MLSBeachesSlugs is the set of counties covered by Beaches MLS parcel sync.
var MLSBeachesSlugs = map[string]bool{
	"broward": true, "palm-beach": true, "martin": true, "st-lucie": true, "miami-dade": true,
}
