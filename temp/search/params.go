package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPageLimit = 24
	maxPageLimit     = 400
)

// ValidationError reports request validation issues.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "invalid request"
}

func newValidationError() *ValidationError {
	return &ValidationError{Fields: make(map[string]string)}
}

// SearchRequest is the normalized request used by the search pipeline.
type SearchRequest struct {
	Params          SearchParams
	LocationFilters LocationFilters
	ListingTypes    []string
	Page            PageParams
	IncludeRTO      bool
	// CurrencyTickers requests digital asset/fiat price quotes for the specified tickers.
	// Examples: ["BTC", "ETH", "SOL", "BNB", "XRP", "USD", "EUR"] - digital asset symbols or fiat currencies.
	CurrencyTickers []string `json:"currency_tickers"`
}

// PageParams defines paging input.
type PageParams struct {
	Limit        IntParam `json:"limit"`
	Cursor       string   `json:"cursor"`
	OverallLimit IntParam `json:"overall_limit"`
	Returned     IntParam `json:"returned"`
}

// SearchParams defines the supported search parameters.
type SearchParams struct {
	MinPrice               FloatParam `json:"min_price"`
	MaxPrice               FloatParam `json:"max_price"`
	MinBeds                IntParam   `json:"min_beds"`
	MaxBeds                IntParam   `json:"max_beds"`
	MinBaths               IntParam   `json:"min_baths"`
	MaxBaths               IntParam   `json:"max_baths"`
	MinSqft                FloatParam `json:"min_sqft"`
	MaxSqft                FloatParam `json:"max_sqft"`
	MinLotSizeAcres        FloatParam `json:"min_lot_size_acres"`
	MaxLotSizeAcres        FloatParam `json:"max_lot_size_acres"`
	MinYearBuilt           IntParam   `json:"min_year_built"`
	MaxYearBuilt           IntParam   `json:"max_year_built"`
	MinDOM                 IntParam   `json:"min_days_on_market"`
	MaxDOM                 IntParam   `json:"max_days_on_market"`
	PriceReducedWithinDays IntParam   `json:"price_reduced_within_days"`
	PoolPrivate            BoolParam  `json:"pool_private"`
	Waterfront             BoolParam  `json:"waterfront"`
	Dock                   BoolParam  `json:"dock"`
	NewConstruction        BoolParam  `json:"new_construction"`
	HasPhotos              BoolParam  `json:"has_photos"`
	ActiveOnly             BoolParam  `json:"active_only"`
	// New SEO filters (Phase 11)
	ActiveAdultCommunity     BoolParam   `json:"active_adult_community"`
	Association              BoolParam   `json:"association"`
	Fireplace                BoolParam   `json:"fireplace"`
	Spa                      BoolParam   `json:"spa"`
	ForLease                 BoolParam   `json:"for_lease"`
	Garage                   BoolParam   `json:"garage"`
	MinMonthlyFees           FloatParam  `json:"min_monthly_fees"`
	MaxMonthlyFees           FloatParam  `json:"max_monthly_fees"`
	MinStories               IntParam    `json:"min_stories"`
	MaxStories               IntParam    `json:"max_stories"`
	Statuses                 []string    `json:"statuses"`
	PropertySubType          []string    `json:"property_sub_types"`
	SpecialListingConditions []string    `json:"special_listing_conditions"`
	Sort                     string      `json:"sort"`
	SortDir                  string      `json:"sort_dir"`
	Geo                      *GeoFilters `json:"geo"`
}

// GeoFilters defines geo-based filters.
type GeoFilters struct {
	Distance *DistanceFilter `json:"distance"`
	BBox     *BBoxFilter     `json:"bbox"`
	Polygon  *PolygonFilter  `json:"polygon"`
}

// DistanceFilter defines a radius filter.
type DistanceFilter struct {
	Lat         FloatParam `json:"lat"`
	Lng         FloatParam `json:"lng"`
	RadiusMiles FloatParam `json:"radius_miles"`
}

// BBoxFilter defines a bounding box filter.
type BBoxFilter struct {
	West  FloatParam `json:"west"`
	South FloatParam `json:"south"`
	East  FloatParam `json:"east"`
	North FloatParam `json:"north"`
}

// PolygonFilter defines a polygon filter.
type PolygonFilter struct {
	Points []GeoPoint `json:"points"`
}

// GeoPoint is a lat/lng pair.
type GeoPoint struct {
	Lat FloatParam `json:"lat"`
	Lng FloatParam `json:"lng"`
}

// LocationFilters defines geo ref-id filters.
type LocationFilters struct {
	StateRefIDs            IDList      `json:"state_ref_ids"`
	CountyRefIDs           IDList      `json:"county_ref_ids"`
	CityRefIDs             IDList      `json:"city_ref_ids"`
	SubdivisionRefIDs      IDList      `json:"subdivision_ref_ids"`
	PostalCodeRefIDs       IDList      `json:"postal_code_ref_ids"`
	ElementarySchoolRefIDs IDList      `json:"elementary_school_ref_ids"`
	MiddleSchoolRefIDs     IDList      `json:"middle_school_ref_ids"`
	HighSchoolRefIDs       IDList      `json:"high_school_ref_ids"`
	FocusAreas             []FocusArea `json:"focus_areas"`
}

// FocusArea identifies a geographic focus region.
type FocusArea struct {
	Type  string `json:"type"`
	ID    *int64 `json:"id"`
	RefID *int64 `json:"ref_id"`
}

// IntParam supports ints encoded as number or string.
type IntParam struct {
	Value *int
}

// FloatParam supports floats encoded as number or string.
type FloatParam struct {
	Value *float64
}

// BoolParam supports bools encoded as bool, number, or string.
type BoolParam struct {
	Value *bool
}

// IDList supports arrays of int64 encoded as numbers or strings.
type IDList struct {
	Values []int64
}

// ParseSearchPayload parses and validates the request payload.
func ParseSearchPayload(data []byte) (SearchRequest, error) {
	var raw rawRequest
	if err := decodeStrict(data, &raw); err != nil {
		return SearchRequest{}, err
	}

	if raw.Context != nil && (len(raw.Params) > 0 || len(raw.LocationFilters) > 0 || len(raw.ListingTypes) > 0) {
		v := newValidationError()
		v.Fields["context"] = "cannot be combined with top-level params"
		return SearchRequest{}, v
	}

	paramsBytes := raw.Params
	filtersBytes := raw.LocationFilters
	listingTypes := raw.ListingTypes
	currencyTickers := raw.CurrencyTickers
	if raw.Context != nil {
		paramsBytes = raw.Context.Params
		filtersBytes = raw.Context.LocationFilters
		listingTypes = raw.Context.ListingTypes
		currencyTickers = raw.Context.CurrencyTickers
	}

	params, err := parseParams(paramsBytes)
	if err != nil {
		return SearchRequest{}, err
	}
	filters, err := parseLocationFilters(filtersBytes)
	if err != nil {
		return SearchRequest{}, err
	}
	page, err := parsePage(raw.Page)
	if err != nil {
		return SearchRequest{}, err
	}

	listingTypes = normalizeStrings(listingTypes)
	currencyTickers = normalizeStrings(currencyTickers)
	filters.normalizeFocusAreas()

	req := SearchRequest{
		Params:          params,
		LocationFilters: filters,
		ListingTypes:    listingTypes,
		Page:            page,
		IncludeRTO:      raw.IncludeRTO,
		CurrencyTickers: currencyTickers,
	}
	if err := req.Validate(); err != nil {
		return SearchRequest{}, err
	}
	return req, nil
}

// Validate checks request-level rules.
func (r *SearchRequest) Validate() error {
	v := newValidationError()
	page := r.Page

	limit := page.Limit.IntOr(defaultPageLimit)
	if limit < 1 || limit > maxPageLimit {
		v.Fields["page.limit"] = fmt.Sprintf("must be between 1 and %d", maxPageLimit)
	}

	overall := page.OverallLimit.IntOr(0)
	returned := page.Returned.IntOr(0)
	if overall < 0 {
		v.Fields["page.overall_limit"] = "must be >= 0"
	}
	if returned < 0 {
		v.Fields["page.returned"] = "must be >= 0"
	}
	if overall > 0 && returned > overall {
		v.Fields["page.returned"] = "cannot exceed overall_limit"
	}

	sort := strings.TrimSpace(r.Params.Sort)
	if sort != "" && !isSortAllowed(sort) {
		v.Fields["params.sort"] = "unsupported sort option"
	}

	sortDir := strings.TrimSpace(strings.ToLower(r.Params.SortDir))
	if sortDir != "" && sortDir != "asc" && sortDir != "desc" {
		v.Fields["params.sort_dir"] = "must be 'asc' or 'desc'"
	}

	if r.Params.Geo != nil {
		validateGeoFilters(r.Params.Geo, v)
	}

	if len(v.Fields) > 0 {
		return v
	}
	return nil
}

func validateGeoFilters(geo *GeoFilters, v *ValidationError) {
	if geo.Distance != nil {
		lat, ok := geo.Distance.Lat.Float64()
		if !ok {
			v.Fields["params.geo.distance.lat"] = "required"
		} else if lat < -90 || lat > 90 {
			v.Fields["params.geo.distance.lat"] = "must be between -90 and 90"
		}

		lng, ok := geo.Distance.Lng.Float64()
		if !ok {
			v.Fields["params.geo.distance.lng"] = "required"
		} else if lng < -180 || lng > 180 {
			v.Fields["params.geo.distance.lng"] = "must be between -180 and 180"
		}

		radius, ok := geo.Distance.RadiusMiles.Float64()
		if !ok {
			v.Fields["params.geo.distance.radius_miles"] = "required"
		} else if radius <= 0 {
			v.Fields["params.geo.distance.radius_miles"] = "must be > 0"
		}
	}

	if geo.BBox != nil {
		west, okW := geo.BBox.West.Float64()
		if !okW {
			v.Fields["params.geo.bbox.west"] = "required"
		}
		east, okE := geo.BBox.East.Float64()
		if !okE {
			v.Fields["params.geo.bbox.east"] = "required"
		}
		south, okS := geo.BBox.South.Float64()
		if !okS {
			v.Fields["params.geo.bbox.south"] = "required"
		}
		north, okN := geo.BBox.North.Float64()
		if !okN {
			v.Fields["params.geo.bbox.north"] = "required"
		}
		if okW && okE && east <= west {
			v.Fields["params.geo.bbox.east"] = "must be greater than west"
		}
		if okS && okN && north <= south {
			v.Fields["params.geo.bbox.north"] = "must be greater than south"
		}
	}

	if geo.Polygon != nil {
		if len(geo.Polygon.Points) < 3 {
			v.Fields["params.geo.polygon.points"] = "must include at least 3 points"
		}
		for i, pt := range geo.Polygon.Points {
			lat, ok := pt.Lat.Float64()
			if !ok || lat < -90 || lat > 90 {
				v.Fields[fmt.Sprintf("params.geo.polygon.points[%d].lat", i)] = "invalid latitude"
			}
			lng, ok := pt.Lng.Float64()
			if !ok || lng < -180 || lng > 180 {
				v.Fields[fmt.Sprintf("params.geo.polygon.points[%d].lng", i)] = "invalid longitude"
			}
		}
	}
}

func parseParams(data json.RawMessage) (SearchParams, error) {
	if isNullJSON(data) {
		return SearchParams{}, nil
	}
	var params SearchParams
	if err := decodeStrict(data, &params); err != nil {
		return SearchParams{}, annotateDecodeError(err, "params")
	}
	params.Statuses = normalizeStrings(params.Statuses)
	params.PropertySubType = normalizeStrings(params.PropertySubType)
	params.SpecialListingConditions = normalizeStrings(params.SpecialListingConditions)
	params.Sort = strings.TrimSpace(params.Sort)
	params.SortDir = strings.TrimSpace(params.SortDir)
	return params, nil
}

func parseLocationFilters(data json.RawMessage) (LocationFilters, error) {
	if isNullJSON(data) {
		return LocationFilters{}, nil
	}
	var filters LocationFilters
	if err := decodeStrict(data, &filters); err != nil {
		return LocationFilters{}, annotateDecodeError(err, "location_filters")
	}
	return filters, nil
}

func parsePage(data json.RawMessage) (PageParams, error) {
	if isNullJSON(data) {
		return PageParams{}, nil
	}
	var page PageParams
	if err := decodeStrict(data, &page); err != nil {
		return PageParams{}, annotateDecodeError(err, "page")
	}
	return page, nil
}

type rawRequest struct {
	Params          json.RawMessage `json:"params"`
	LocationFilters json.RawMessage `json:"location_filters"`
	ListingTypes    []string        `json:"listing_types"`
	Context         *rawContext     `json:"context"`
	Page            json.RawMessage `json:"page"`
	IncludeRTO      bool            `json:"include_rto"`
	CurrencyTickers []string        `json:"currency_tickers"`
}

type rawContext struct {
	Params          json.RawMessage `json:"params"`
	LocationFilters json.RawMessage `json:"location_filters"`
	ListingTypes    []string        `json:"listing_types"`
	CurrencyTickers []string        `json:"currency_tickers"`
}

func decodeStrict(data []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("extra data after JSON")
	}
	return nil
}

func annotateDecodeError(err error, prefix string) error {
	if err == nil {
		return nil
	}
	v := newValidationError()

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = prefix
		} else {
			field = prefix + "." + field
		}
		v.Fields[field] = "invalid type"
		return v
	}

	msg := err.Error()
	if strings.Contains(msg, "unknown field") {
		field := extractUnknownField(msg)
		if field != "" {
			v.Fields[prefix+"."+field] = "unknown field"
			return v
		}
	}

	v.Fields[prefix] = "invalid payload"
	return v
}

func extractUnknownField(msg string) string {
	start := strings.Index(msg, "\"")
	end := strings.LastIndex(msg, "\"")
	if start >= 0 && end > start {
		return msg[start+1 : end]
	}
	return ""
}

func isNullJSON(data []byte) bool {
	trimmed := strings.TrimSpace(string(data))
	return trimmed == "" || trimmed == "null"
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func (f *FocusArea) normalize() (string, int64, bool) {
	if f == nil {
		return "", 0, false
	}
	typeVal := strings.ToLower(strings.TrimSpace(f.Type))
	var id int64
	if f.ID != nil {
		id = *f.ID
	}
	if id == 0 && f.RefID != nil {
		id = *f.RefID
	}
	if typeVal == "" || id <= 0 {
		return "", 0, false
	}
	return typeVal, id, true
}

func (l *LocationFilters) normalizeFocusAreas() {
	for _, area := range l.FocusAreas {
		kind, id, ok := area.normalize()
		if !ok {
			continue
		}
		switch kind {
		case "state":
			l.StateRefIDs.Values = append(l.StateRefIDs.Values, id)
		case "county":
			l.CountyRefIDs.Values = append(l.CountyRefIDs.Values, id)
		case "city":
			l.CityRefIDs.Values = append(l.CityRefIDs.Values, id)
		case "subdivision":
			l.SubdivisionRefIDs.Values = append(l.SubdivisionRefIDs.Values, id)
		case "postal_code", "postal", "zip":
			l.PostalCodeRefIDs.Values = append(l.PostalCodeRefIDs.Values, id)
		case "elementary_school", "elementary":
			l.ElementarySchoolRefIDs.Values = append(l.ElementarySchoolRefIDs.Values, id)
		case "middle_school", "middle", "junior_high":
			l.MiddleSchoolRefIDs.Values = append(l.MiddleSchoolRefIDs.Values, id)
		case "high_school", "high":
			l.HighSchoolRefIDs.Values = append(l.HighSchoolRefIDs.Values, id)
		}
	}
}

func (p IntParam) IntOr(defaultVal int) int {
	if p.Value == nil {
		return defaultVal
	}
	return *p.Value
}

func (p IntParam) Int() (int, bool) {
	if p.Value == nil {
		return 0, false
	}
	return *p.Value, true
}

func (p FloatParam) Float64() (float64, bool) {
	if p.Value == nil {
		return 0, false
	}
	return *p.Value, true
}

func (p BoolParam) BoolOr(defaultVal bool) bool {
	if p.Value == nil {
		return defaultVal
	}
	return *p.Value
}

func (p BoolParam) Bool() (bool, bool) {
	if p.Value == nil {
		return false, false
	}
	return *p.Value, true
}

func (p *IntParam) UnmarshalJSON(data []byte) error {
	if isNullJSON(data) {
		p.Value = nil
		return nil
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "\"") {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			p.Value = nil
			return nil
		}
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer")
		}
		v := int(val)
		p.Value = &v
		return nil
	}

	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		if i, err := num.Int64(); err == nil {
			v := int(i)
			p.Value = &v
			return nil
		}
		if f, err := num.Float64(); err == nil {
			if f != float64(int64(f)) {
				return fmt.Errorf("invalid integer")
			}
			v := int(f)
			p.Value = &v
			return nil
		}
	}

	return fmt.Errorf("invalid integer")
}

func (p *FloatParam) UnmarshalJSON(data []byte) error {
	if isNullJSON(data) {
		p.Value = nil
		return nil
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "\"") {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			p.Value = nil
			return nil
		}
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("invalid number")
		}
		p.Value = &val
		return nil
	}

	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		if f, err := num.Float64(); err == nil {
			p.Value = &f
			return nil
		}
	}

	return fmt.Errorf("invalid number")
}

func (p *BoolParam) UnmarshalJSON(data []byte) error {
	if isNullJSON(data) {
		p.Value = nil
		return nil
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "\"") {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case "true", "t", "yes", "y", "1":
			val := true
			p.Value = &val
			return nil
		case "false", "f", "no", "n", "0":
			val := false
			p.Value = &val
			return nil
		default:
			return fmt.Errorf("invalid boolean")
		}
	}

	if trimmed == "true" || trimmed == "false" {
		var b bool
		if err := json.Unmarshal(data, &b); err != nil {
			return err
		}
		p.Value = &b
		return nil
	}

	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		if i, err := num.Int64(); err == nil {
			val := i != 0
			p.Value = &val
			return nil
		}
	}

	return fmt.Errorf("invalid boolean")
}

func (l *IDList) UnmarshalJSON(data []byte) error {
	if isNullJSON(data) {
		l.Values = nil
		return nil
	}
	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	values := make([]int64, 0, len(raw))
	for _, item := range raw {
		switch v := item.(type) {
		case float64:
			values = append(values, int64(v))
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			parsed, err := strconv.ParseInt(trimmed, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id")
			}
			values = append(values, parsed)
		default:
			return fmt.Errorf("invalid id")
		}
	}
	l.Values = values
	return nil
}

func isSortAllowed(sort string) bool {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "list_price",
		"on_market_date",
		"year_built",
		"living_area",
		"lot_size_acres",
		"bedrooms_total",
		"bathrooms_total",
		"distance":
		return true
	default:
		return false
	}
}

func (p SearchParams) SortKey() string {
	if p.Sort == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(p.Sort))
}

func (p SearchParams) SortDirection() string {
	if p.SortDir == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(p.SortDir))
}

func (p SearchParams) ActiveOnlyOrDefault() bool {
	return p.ActiveOnly.BoolOr(true)
}

func (g GeoFilters) HasDistance() bool {
	if g.Distance == nil {
		return false
	}
	_, okLat := g.Distance.Lat.Float64()
	_, okLng := g.Distance.Lng.Float64()
	_, okRad := g.Distance.RadiusMiles.Float64()
	return okLat && okLng && okRad
}

func (g GeoFilters) DistanceValues() (float64, float64, float64, bool) {
	if g.Distance == nil {
		return 0, 0, 0, false
	}
	lat, okLat := g.Distance.Lat.Float64()
	lng, okLng := g.Distance.Lng.Float64()
	rad, okRad := g.Distance.RadiusMiles.Float64()
	if !okLat || !okLng || !okRad {
		return 0, 0, 0, false
	}
	return lat, lng, rad, true
}

func (g GeoFilters) DistanceCenter() (float64, float64, bool) {
	if g.Distance == nil {
		return 0, 0, false
	}
	lat, okLat := g.Distance.Lat.Float64()
	lng, okLng := g.Distance.Lng.Float64()
	if !okLat || !okLng {
		return 0, 0, false
	}
	return lat, lng, true
}

func (g GeoFilters) BBoxValues() (float64, float64, float64, float64, bool) {
	if g.BBox == nil {
		return 0, 0, 0, 0, false
	}
	west, okW := g.BBox.West.Float64()
	south, okS := g.BBox.South.Float64()
	east, okE := g.BBox.East.Float64()
	north, okN := g.BBox.North.Float64()
	if !okW || !okS || !okE || !okN {
		return 0, 0, 0, 0, false
	}
	return west, south, east, north, true
}

func (g GeoFilters) PolygonWKT() (string, bool) {
	if g.Polygon == nil || len(g.Polygon.Points) < 3 {
		return "", false
	}
	points := make([]string, 0, len(g.Polygon.Points)+1)
	for _, pt := range g.Polygon.Points {
		lat, okLat := pt.Lat.Float64()
		lng, okLng := pt.Lng.Float64()
		if !okLat || !okLng {
			return "", false
		}
		points = append(points, fmt.Sprintf("%f %f", lng, lat))
	}
	if points[0] != points[len(points)-1] {
		points = append(points, points[0])
	}
	return fmt.Sprintf("POLYGON((%s))", strings.Join(points, ", ")), true
}

func (p SearchParams) CursorSortTime(raw any) (time.Time, error) {
	str, ok := raw.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("invalid cursor time")
	}
	parsed, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func (p SearchParams) NormalizeSort(defaultSort string) string {
	if p.Sort == "" {
		return defaultSort
	}
	return strings.ToLower(strings.TrimSpace(p.Sort))
}

func (p SearchParams) NormalizeSortDir(defaultDir string) string {
	if p.SortDir == "" {
		return defaultDir
	}
	return strings.ToLower(strings.TrimSpace(p.SortDir))
}
