package search

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type searchRequestWire struct {
	Statuses               []string        `json:"statuses"`
	ActiveOnly             json.RawMessage `json:"active_only"`
	MinPrice               json.RawMessage `json:"min_price"`
	MaxPrice               json.RawMessage `json:"max_price"`
	BedsMin                json.RawMessage `json:"beds_min"`
	MinBeds                json.RawMessage `json:"min_beds"`
	BedsMax                json.RawMessage `json:"beds_max"`
	MaxBeds                json.RawMessage `json:"max_beds"`
	BathsMin               json.RawMessage `json:"baths_min"`
	MinBaths               json.RawMessage `json:"min_baths"`
	LivingAreaMin          json.RawMessage `json:"living_area_min"`
	MinSqft                json.RawMessage `json:"min_sqft"`
	LivingAreaMax          json.RawMessage `json:"living_area_max"`
	MaxSqft                json.RawMessage `json:"max_sqft"`
	LotSizeAcresMin        json.RawMessage `json:"lot_size_acres_min"`
	YearBuiltMin           json.RawMessage `json:"year_built_min"`
	PropertyType           *string         `json:"property_type"`
	PropertySubType        *string         `json:"property_sub_type"`
	City                   *string         `json:"city"`
	CountyOrParish         *string         `json:"county_or_parish"`
	RemarksQuery           *string         `json:"remarks_query"`
	PostalCode             *string         `json:"postal_code"`
	PoolPrivate            json.RawMessage `json:"pool_private"`
	Waterfront             json.RawMessage `json:"waterfront"`
	Lat                    json.RawMessage `json:"lat"`
	Lng                    json.RawMessage `json:"lng"`
	RadiusMiles            json.RawMessage `json:"radius_miles"`
	PriceReducedWithinDays json.RawMessage `json:"price_reduced_within_days"`
	LowRiskFloodzone       json.RawMessage `json:"low_risk_floodzone"`
	MinMonthlyFees         json.RawMessage `json:"min_monthly_fees"`
	MaxMonthlyFees         json.RawMessage `json:"max_monthly_fees"`
	Skip                   json.RawMessage `json:"skip"`
	Limit                  json.RawMessage `json:"limit"`
	Page                   json.RawMessage `json:"page"`
}

// UnmarshalJSON accepts documented search fields with OpenAPI/Laravel-style aliases:
// string-encoded numbers, empty strings as omitted, min_beds/max_beds, and page.limit/skip.
func (r *SearchRequest) UnmarshalJSON(data []byte) error {
	var w searchRequestWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}

	var err error
	out := SearchRequest{
		Statuses:        filterNonEmptyStrings(w.Statuses),
		PropertyType:    w.PropertyType,
		PropertySubType: w.PropertySubType,
		City:            trimStringPtr(w.City),
		CountyOrParish:  trimStringPtr(w.CountyOrParish),
		RemarksQuery:    trimStringPtr(w.RemarksQuery),
		PostalCode:      trimStringPtr(w.PostalCode),
	}
	if out.ActiveOnly, err = parseFlexBool(w.ActiveOnly); err != nil {
		return err
	}
	if out.MinPrice, err = parseFlexFloat(w.MinPrice); err != nil {
		return err
	}
	if out.MaxPrice, err = parseFlexFloat(w.MaxPrice); err != nil {
		return err
	}
	if out.BedsMin, err = firstFlexInt(w.BedsMin, w.MinBeds); err != nil {
		return err
	}
	if out.BedsMax, err = firstFlexInt(w.BedsMax, w.MaxBeds); err != nil {
		return err
	}
	if out.BathsMin, err = firstFlexFloat(w.BathsMin, w.MinBaths); err != nil {
		return err
	}
	if out.LivingAreaMin, err = firstFlexInt(w.LivingAreaMin, w.MinSqft); err != nil {
		return err
	}
	if out.LivingAreaMax, err = firstFlexInt(w.LivingAreaMax, w.MaxSqft); err != nil {
		return err
	}
	if out.LotSizeAcresMin, err = parseFlexFloat(w.LotSizeAcresMin); err != nil {
		return err
	}
	if out.YearBuiltMin, err = parseFlexInt(w.YearBuiltMin); err != nil {
		return err
	}
	if out.PoolPrivate, err = parseFlexBool(w.PoolPrivate); err != nil {
		return err
	}
	if out.Waterfront, err = parseFlexBool(w.Waterfront); err != nil {
		return err
	}
	if out.Lat, err = parseFlexFloat(w.Lat); err != nil {
		return err
	}
	if out.Lng, err = parseFlexFloat(w.Lng); err != nil {
		return err
	}
	if out.RadiusMiles, err = parseFlexFloat(w.RadiusMiles); err != nil {
		return err
	}
	if out.PriceReducedWithinDays, err = parseFlexInt(w.PriceReducedWithinDays); err != nil {
		return err
	}
	if out.LowRiskFloodzone, err = parseFlexBool(w.LowRiskFloodzone); err != nil {
		return err
	}
	if out.MinMonthlyFees, err = parseFlexFloat(w.MinMonthlyFees); err != nil {
		return err
	}
	if out.MaxMonthlyFees, err = parseFlexFloat(w.MaxMonthlyFees); err != nil {
		return err
	}
	if out.Skip, err = parseFlexIntDefault(w.Skip, 0); err != nil {
		return err
	}
	if out.Limit, err = parseFlexIntDefault(w.Limit, 0); err != nil {
		return err
	}
	if len(w.Page) > 0 {
		var p struct {
			Limit json.RawMessage `json:"limit"`
			Skip  json.RawMessage `json:"skip"`
		}
		if err := json.Unmarshal(w.Page, &p); err != nil {
			return fmt.Errorf("page: %w", err)
		}
		if lim, err := parseFlexInt(p.Limit); err != nil {
			return fmt.Errorf("page.limit: %w", err)
		} else if lim != nil {
			out.Limit = *lim
		}
		if sk, err := parseFlexInt(p.Skip); err != nil {
			return fmt.Errorf("page.skip: %w", err)
		} else if sk != nil {
			out.Skip = *sk
		}
	}
	*r = out
	return nil
}

func firstFlexInt(primary, alias json.RawMessage) (*int, error) {
	if len(primary) > 0 && strings.TrimSpace(string(primary)) != "" && strings.TrimSpace(string(primary)) != `""` {
		return parseFlexInt(primary)
	}
	return parseFlexInt(alias)
}

func firstFlexFloat(primary, alias json.RawMessage) (*float64, error) {
	if len(primary) > 0 && strings.TrimSpace(string(primary)) != "" && strings.TrimSpace(string(primary)) != `""` {
		return parseFlexFloat(primary)
	}
	return parseFlexFloat(alias)
}

func trimStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func filterNonEmptyStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func parseFlexInt(raw json.RawMessage) (*int, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	s := strings.TrimSpace(string(raw))
	if s == "null" || s == `""` {
		return nil, nil
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return &n, nil
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		i := int(f)
		return &i, nil
	}
	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return nil, err
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return nil, nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func parseFlexIntDefault(raw json.RawMessage, def int) (int, error) {
	n, err := parseFlexInt(raw)
	if err != nil {
		return 0, err
	}
	if n == nil {
		return def, nil
	}
	return *n, nil
}

func parseFlexFloat(raw json.RawMessage) (*float64, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	s := strings.TrimSpace(string(raw))
	if s == "null" || s == `""` {
		return nil, nil
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return &f, nil
	}
	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return nil, err
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseFlexBool(raw json.RawMessage) (*bool, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	s := strings.TrimSpace(string(raw))
	if s == "null" || s == `""` {
		return nil, nil
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return &b, nil
	}
	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return nil, err
	}
	str = strings.TrimSpace(strings.ToLower(str))
	if str == "" {
		return nil, nil
	}
	switch str {
	case "true", "1", "yes":
		v := true
		return &v, nil
	case "false", "0", "no":
		v := false
		return &v, nil
	default:
		return nil, fmt.Errorf("invalid boolean %q", str)
	}
}
