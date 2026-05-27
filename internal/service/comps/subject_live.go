package comps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// subjectFromLiveClosed resolves subject from live RESO when not present in AP mirror.
// This supports buyer flows for recently closed properties by ListingId.
func (e *Engine) subjectFromLiveClosed(ctx context.Context, dataset, listingRef string, in SubjectInput) (SubjectProfile, error) {
	feed := mls.FeedDefinitionFromCode(e.cfg, dataset)
	endpoint, err := mlspoxy.LiveResoPropertyEndpoint(e.cfg, feed)
	if err != nil {
		return SubjectProfile{}, err
	}
	u, err := url.Parse(endpoint.PropertyURL)
	if err != nil {
		return SubjectProfile{}, err
	}
	ref := strings.TrimSpace(listingRef)
	if ref == "" {
		return SubjectProfile{}, fmt.Errorf("subject.listing_id is required for mls subject")
	}
	refEscaped := strings.ReplaceAll(ref, "'", "''")
	q := u.Query()
	q.Set("$filter", fmt.Sprintf("StandardStatus eq 'Closed' and (ListingId eq '%s' or ListingKey eq '%s')", refEscaped, refEscaped))
	q.Set("$orderby", "CloseDate desc")
	q.Set("$top", "1")
	u.RawQuery = q.Encode()

	body, status, err := e.getReso(ctx, u.String(), endpoint.Bearer)
	if err != nil {
		return SubjectProfile{}, err
	}
	if status >= 400 {
		return SubjectProfile{}, fmt.Errorf("live MLS closed subject lookup status %d for feed %q", status, feed.Code)
	}
	var envelope struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return SubjectProfile{}, err
	}
	if len(envelope.Value) == 0 {
		return SubjectProfile{}, fmt.Errorf("subject listing not found in mirror or live closed feed: %s", ref)
	}
	raw := envelope.Value[0]
	comp := parseProperty(raw)
	sub := SubjectProfile{
		Type:         "mls",
		ListingKey:   comp.ListingKey,
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
		ListPrice:    comp.ListPrice,
		Raw:          raw,
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
