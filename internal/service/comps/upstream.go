package comps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func (e *Engine) fetchSoldComps(
	ctx context.Context,
	feed mls.FeedDefinition,
	subject SubjectProfile,
	scope ScopeInput,
	filters FiltersInput,
	limit int,
) ([]CompRecord, error) {
	if limit <= 0 {
		limit = 25
	}
	endpoint, err := mlspoxy.LiveResoPropertyEndpoint(e.cfg, feed)
	if err != nil {
		return nil, err
	}

	months := 12
	if filters.SoldMonthsBack != nil && *filters.SoldMonthsBack > 0 {
		months = *filters.SoldMonthsBack
	}
	cutoff := time.Now().AddDate(0, -months, 0).UTC().Format("2006-01-02T15:04:05Z")

	u, err := url.Parse(endpoint.PropertyURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	var parts []string
	parts = append(parts, "StandardStatus eq 'Closed'")
	parts = append(parts, fmt.Sprintf("CloseDate ge %s", cutoff))
	scopeParts := geoFilterParts(subject, scope)
	parts = append(parts, scopeParts...)
	q.Set("$filter", strings.Join(parts, " and "))
	q.Set("$orderby", "CloseDate desc")
	q.Set("$top", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	body, status, err := e.getReso(ctx, u.String(), endpoint.Bearer)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("live MLS sold comps status %d for feed %q", status, feed.Code)
	}
	return parseSoldCompRecords(body, subject, scope)
}

func (e *Engine) fetchRentalComps(
	ctx context.Context,
	feed mls.FeedDefinition,
	subject SubjectProfile,
	scope ScopeInput,
	limit int,
) ([]CompRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	endpoint, err := mlspoxy.LiveResoPropertyEndpoint(e.cfg, feed)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(endpoint.PropertyURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	parts := []string{
		"PropertyType eq 'Residential Lease'",
		"StandardStatus eq 'Closed'",
	}
	parts = append(parts, geoFilterParts(subject, scope)...)
	q.Set("$filter", strings.Join(parts, " and "))
	q.Set("$orderby", "CloseDate desc")
	q.Set("$top", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	body, status, err := e.getReso(ctx, u.String(), endpoint.Bearer)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("live MLS rental comps status %d for feed %q", status, feed.Code)
	}
	closed, err := parseRentalCompRecords(body, subject, scope)
	if err != nil {
		return nil, err
	}

	// Active/Pending rental inventory comes from the mirror for all MLS feeds.
	active, err := e.findMirrorComps(ctx, feed.Dataset, subject, []string{"active", "pending"}, scope, FiltersInput{}, limit, true)
	if err != nil {
		return closed, err
	}
	return append(closed, active...), nil
}

func geoFilterParts(subject SubjectProfile, scope ScopeInput) []string {
	if scope.Type == "zip" && len(scope.PostalCodes) > 0 {
		var zips []string
		for _, z := range scope.PostalCodes {
			zips = append(zips, "'"+strings.ReplaceAll(z, "'", "''")+"'")
		}
		return []string{"PostalCode in (" + strings.Join(zips, ",") + ")"}
	}
	radius := 5.0
	if scope.RadiusMiles != nil && *scope.RadiusMiles > 0 {
		radius = *scope.RadiusMiles
	}
	deg := radius / 69.0
	return []string{
		fmt.Sprintf("Latitude ge %f", subject.Lat-deg),
		fmt.Sprintf("Latitude le %f", subject.Lat+deg),
		fmt.Sprintf("Longitude ge %f", subject.Lng-deg),
		fmt.Sprintf("Longitude le %f", subject.Lng+deg),
	}
}

func (e *Engine) getReso(ctx context.Context, rawURL, bearer string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Accept", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func parseSoldCompRecords(body []byte, subject SubjectProfile, scope ScopeInput) ([]CompRecord, error) {
	var envelope struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	var out []CompRecord
	for _, raw := range envelope.Value {
		c := parseProperty(raw)
		if c.ListingKey == subject.ListingKey {
			continue
		}
		if c.ClosePrice <= 0 {
			continue
		}
		c.DistanceMiles = haversineMiles(subject.Lat, subject.Lng, c.Lat, c.Lng)
		if scope.Type != "zip" && scope.RadiusMiles != nil && c.DistanceMiles > *scope.RadiusMiles*1.15 {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func parseRentalCompRecords(body []byte, subject SubjectProfile, scope ScopeInput) ([]CompRecord, error) {
	var envelope struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	var out []CompRecord
	for _, raw := range envelope.Value {
		c := parseProperty(raw)
		if c.ListingKey == subject.ListingKey {
			continue
		}
		price := c.ClosePrice
		if price == 0 {
			price = c.ListPrice
		}
		if price <= 0 {
			continue
		}
		if strings.EqualFold(strFromRaw(raw, "LeaseAmountFrequency"), "Annually") && price > 0 {
			price /= 12
		}
		c.ClosePrice = price
		c.ListPrice = price
		c.DistanceMiles = haversineMiles(subject.Lat, subject.Lng, c.Lat, c.Lng)
		if scope.Type != "zip" && scope.RadiusMiles != nil && c.DistanceMiles > *scope.RadiusMiles*1.15 {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func strFromRaw(raw json.RawMessage, key string) string {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	return str(m[key])
}
