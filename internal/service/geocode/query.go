package geocode

import (
	"fmt"
	"regexp"
	"strings"
)

// usAddressTail matches ", FL 34951" or ", FL 34951-1234" at end of unparsed line.
var usAddressTail = regexp.MustCompile(`,\s*[A-Za-z]{2}\s+\d{5}(-\d{4})?\s*$`)

// BuildGeocodeQuery builds a single-line address for Google Geocoding API.
// Beaches often ships a complete UnparsedAddress; Stellar often ships street-only plus typed city/state/ZIP.
func BuildGeocodeQuery(
	dataset string,
	unparsed string,
	city, state, postal *string,
	streetNum, streetName *string,
) (query string, ok bool) {
	_ = dataset
	unparsed = strings.TrimSpace(unparsed)
	if unparsed != "" && usAddressTail.MatchString(unparsed) {
		return normalizeWS(unparsed), true
	}
	line1 := unparsed
	if line1 == "" {
		var parts []string
		if streetNum != nil {
			parts = append(parts, strings.TrimSpace(*streetNum))
		}
		if streetName != nil {
			parts = append(parts, strings.TrimSpace(*streetName))
		}
		line1 = strings.TrimSpace(strings.Join(parts, " "))
	}
	cityS := strPtr(city)
	stateS := strPtr(state)
	postalS := strPtr(postal)
	if line1 == "" && cityS == "" && postalS == "" {
		return "", false
	}
	if line1 != "" && (cityS != "" || postalS != "") {
		tail := composeCityStatePostal(cityS, stateS, postalS)
		if tail != "" {
			return normalizeWS(line1 + ", " + tail), true
		}
	}
	if line1 != "" {
		return normalizeWS(line1), cityS != "" || postalS != ""
	}
	tail := composeCityStatePostal(cityS, stateS, postalS)
	if tail == "" {
		return "", false
	}
	return normalizeWS(tail), true
}

func composeCityStatePostal(city, state, postal string) string {
	if city == "" && postal == "" {
		return ""
	}
	if state != "" && postal != "" {
		if city != "" {
			return fmt.Sprintf("%s, %s %s", city, state, postal)
		}
		return fmt.Sprintf("%s %s", state, postal)
	}
	if city != "" && postal != "" {
		return fmt.Sprintf("%s %s", city, postal)
	}
	if city != "" {
		return city
	}
	return postal
}

func strPtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func normalizeWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
