package search

import (
	"strings"
)

// ListingAddress holds formatted address data.
type ListingAddress struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Full       string `json:"full"`
}

func buildListingAddress(parts addressParts) ListingAddress {
	line1 := buildStreetLine(parts)
	cityStateZip := buildCityStateZip(parts)
	full := line1
	if line1 != "" && cityStateZip != "" {
		full = line1 + ", " + cityStateZip
	} else if cityStateZip != "" {
		full = cityStateZip
	}
	return ListingAddress{
		Line1:      line1,
		Line2:      cityStateZip,
		City:       parts.City,
		State:      parts.State,
		PostalCode: parts.PostalCode,
		Full:       full,
	}
}

type addressParts struct {
	StreetNumber string
	StreetDirPre string
	StreetName   string
	StreetSuffix string
	StreetDirSuf string
	UnitNumber   string
	City         string
	State        string
	PostalCode   string
}

func buildStreetLine(parts addressParts) string {
	segments := []string{}
	appendSegment := func(value string) {
		if value != "" {
			segments = append(segments, value)
		}
	}
	appendSegment(parts.StreetNumber)
	appendSegment(parts.StreetDirPre)
	appendSegment(parts.StreetName)
	appendSegment(parts.StreetSuffix)
	appendSegment(parts.StreetDirSuf)
	line := strings.Join(segments, " ")
	unit := strings.TrimSpace(parts.UnitNumber)
	if unit == "" {
		return line
	}
	unitLower := strings.ToLower(unit)
	if strings.HasPrefix(unitLower, "#") || strings.HasPrefix(unitLower, "unit") || strings.HasPrefix(unitLower, "apt") {
		return strings.TrimSpace(line + " " + unit)
	}
	return strings.TrimSpace(line + " #" + unit)
}

func buildCityStateZip(parts addressParts) string {
	city := strings.TrimSpace(parts.City)
	state := strings.TrimSpace(parts.State)
	postal := strings.TrimSpace(parts.PostalCode)
	line := strings.Builder{}
	if city != "" {
		line.WriteString(city)
	}
	if state != "" {
		if line.Len() > 0 {
			line.WriteString(", ")
		}
		line.WriteString(state)
	}
	if postal != "" {
		if line.Len() > 0 {
			line.WriteString(" ")
		}
		line.WriteString(postal)
	}
	return strings.TrimSpace(line.String())
}
