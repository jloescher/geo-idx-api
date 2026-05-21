package comps

import (
	"context"
	"fmt"
	"strings"

	"github.com/quantyralabs/idx-api/internal/service/geocode"
)

func (e *Engine) resolveHomeValueSubject(ctx context.Context, dataset string, in SubjectInput) (SubjectProfile, error) {
	if strings.TrimSpace(in.ListingID) != "" {
		in.Type = "mls"
		sub, err := e.subjectFromMLS(ctx, dataset, in)
		if err != nil {
			return SubjectProfile{}, err
		}
		applyDerivedCondition(&sub, in)
		return sub, nil
	}
	if in.Address != nil && strings.TrimSpace(*in.Address) != "" {
		g := geocode.NewGeocoder(e.cfg.Geocode.GoogleMapsAPIKey, e.cfg.Geocode.HTTPTimeout)
		lat, lng, err := g.Geocode(ctx, *in.Address)
		if err != nil {
			return SubjectProfile{}, err
		}
		in.Lat = &lat
		in.Lng = &lng
		in.Type = "off_market"
		return subjectFromOffMarket(in)
	}
	if in.Lat != nil && in.Lng != nil {
		in.Type = "off_market"
		return subjectFromOffMarket(in)
	}
	return SubjectProfile{}, fmt.Errorf("home_value requires listing_id or address with geocodable details")
}
