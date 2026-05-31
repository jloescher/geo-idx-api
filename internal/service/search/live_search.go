package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/odata"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/upstream"
	"github.com/quantyralabs/idx-api/internal/service/cache"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// LiveSearch queries live upstream RESO (with Web fallback) for statuses not stored in the mirror.
type LiveSearch struct {
	cfg        config.Config
	proxyCache *cache.ProxyCache
	http       *http.Client
}

func NewLiveSearch(cfg config.Config, proxyCache *cache.ProxyCache) *LiveSearch {
	return &LiveSearch{
		cfg:        cfg,
		proxyCache: proxyCache,
		http:       &http.Client{Timeout: cfg.Bridge.Timeout},
	}
}

func (l *LiveSearch) Search(ctx context.Context, c *fiber.Ctx, feedCode string, req SearchRequest) (SearchResult, error) {
	feed := mlspoxy.Feed(c)
	endpoint, err := mlspoxy.ResolveLivePropertyEndpoint(l.cfg, feed)
	if err != nil {
		return SearchResult{}, err
	}

	partition := cache.SearchPartition(mlsDomainSlug(c), feedCode)
	fpBase := searchFingerprint(req)
	datasetSlug := mls.DatasetSlugFromFeedCode(feedCode)

	for _, leg := range []string{"reso", "reso-legacy", "reso-bare", "reso-v3", "web"} {
		if body, ok, err := l.proxyCache.Get(ctx, partition, fpBase+":"+leg); err == nil && ok {
			return parseSearchBodyFromUpstream(body, datasetSlug, leg)
		}
	}

	candidates := upstream.BuildPropertySearchCandidates(l.cfg, feed)
	body, leg, status, err := l.fetchSearchCandidates(ctx, feed, req, endpoint.Bearer, candidates, datasetSlug)
	if err != nil {
		return SearchResult{}, err
	}
	if status >= 400 {
		return SearchResult{}, fmt.Errorf("live search upstream status %d", status)
	}
	_ = l.proxyCache.Put(ctx, partition, fpBase+":"+leg, body)
	return parseSearchBodyFromUpstream(body, datasetSlug, leg)
}

func (l *LiveSearch) fetchSearchCandidates(
	ctx context.Context,
	feed dom.FeedDefinition,
	req SearchRequest,
	bearer string,
	candidates []upstream.Candidate,
	datasetSlug string,
) ([]byte, string, int, error) {
	def := odata.ForDataset(datasetSlug)
	for i, cand := range candidates {
		rawURL, err := l.candidateSearchURL(cand, req, datasetSlug, def.OrderByField)
		if err != nil {
			continue
		}
		body, status, err := l.getPage(ctx, rawURL, bearer)
		if err != nil {
			return nil, "", 0, err
		}
		if status == fiber.StatusNotFound && i < len(candidates)-1 {
			continue
		}
		return body, cand.Leg, status, nil
	}
	return nil, "", fiber.StatusNotFound, nil
}

func (l *LiveSearch) candidateSearchURL(cand upstream.Candidate, req SearchRequest, datasetSlug, orderBy string) (string, error) {
	u, err := url.Parse(cand.URL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if cand.Leg == "web" {
		q.Set("limit", fmt.Sprintf("%d", limit))
		if req.Skip > 0 {
			q.Set("offset", fmt.Sprintf("%d", req.Skip))
		}
	} else {
		if filter := buildODataFilter(req, datasetSlug); filter != "" {
			q.Set("$filter", filter)
		}
		q.Set("$top", fmt.Sprintf("%d", limit))
		if req.Skip > 0 {
			q.Set("$skip", fmt.Sprintf("%d", req.Skip))
		}
		q.Set("$orderby", orderBy)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (l *LiveSearch) getPage(ctx context.Context, rawURL, bearer string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Accept", "application/json")
	resp, err := l.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func mlsDomainSlug(c *fiber.Ctx) string {
	s, _ := c.Locals(ctxkeys.MLSDomainSlug).(string)
	return s
}

func searchFingerprint(req SearchRequest) string {
	b, _ := json.Marshal(req)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func parseSearchBody(body []byte, datasetSlug string) (SearchResult, error) {
	var envelope struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return SearchResult{Results: filterSearchResultsForPublic([]json.RawMessage{body}, datasetSlug)}, nil
	}
	return SearchResult{Results: filterSearchResultsForPublic(envelope.Value, datasetSlug), HasMore: false}, nil
}

// buildODataFilter constructs an OData $filter string for live upstream search.
func buildODataFilter(req SearchRequest, datasetSlug string) string {
	var parts []string
	parts = append(parts, "(InternetEntireListingDisplayYN ne false)")
	if extra := odata.ForDataset(datasetSlug).IDXParticipationAnd; extra != "" {
		parts = append(parts, extra)
	}
	if len(req.Statuses) > 0 {
		var quoted []string
		for _, st := range req.Statuses {
			quoted = append(quoted, "'"+strings.ReplaceAll(st, "'", "''")+"'")
		}
		parts = append(parts, "StandardStatus in ("+strings.Join(quoted, ",")+")")
	}
	if req.MinPrice != nil {
		parts = append(parts, fmt.Sprintf("ListPrice ge %g", *req.MinPrice))
	}
	if req.MaxPrice != nil {
		parts = append(parts, fmt.Sprintf("ListPrice le %g", *req.MaxPrice))
	}
	if req.BedsMin != nil {
		parts = append(parts, fmt.Sprintf("BedroomsTotal ge %d", *req.BedsMin))
	}
	if req.BedsMax != nil {
		parts = append(parts, fmt.Sprintf("BedroomsTotal le %d", *req.BedsMax))
	}
	if req.BathsMin != nil {
		parts = append(parts, fmt.Sprintf("BathroomsTotal ge %g", *req.BathsMin))
	}
	if req.LivingAreaMin != nil {
		parts = append(parts, fmt.Sprintf("LivingArea ge %d", *req.LivingAreaMin))
	}
	if req.LivingAreaMax != nil {
		parts = append(parts, fmt.Sprintf("LivingArea le %d", *req.LivingAreaMax))
	}
	if req.LotSizeAcresMin != nil {
		parts = append(parts, fmt.Sprintf("LotSizeAcres ge %g", *req.LotSizeAcresMin))
	}
	if req.YearBuiltMin != nil {
		parts = append(parts, fmt.Sprintf("YearBuilt ge %d", *req.YearBuiltMin))
	}
	if req.PropertyType != nil && strings.TrimSpace(*req.PropertyType) != "" {
		parts = append(parts, fmt.Sprintf("PropertyType eq '%s'", oDataEscape(*req.PropertyType)))
	}
	if req.PropertySubType != nil && strings.TrimSpace(*req.PropertySubType) != "" {
		parts = append(parts, fmt.Sprintf("PropertySubType eq '%s'", oDataEscape(*req.PropertySubType)))
	}
	if req.City != nil && strings.TrimSpace(*req.City) != "" {
		parts = append(parts, fmt.Sprintf("City eq '%s'", oDataEscape(*req.City)))
	}
	if req.CountyOrParish != nil && strings.TrimSpace(*req.CountyOrParish) != "" {
		parts = append(parts, fmt.Sprintf("CountyOrParish eq '%s'", oDataEscape(*req.CountyOrParish)))
	}
	if req.PostalCode != nil && strings.TrimSpace(*req.PostalCode) != "" {
		parts = append(parts, fmt.Sprintf("PostalCode eq '%s'", oDataEscape(*req.PostalCode)))
	}
	if req.PoolPrivate != nil && *req.PoolPrivate {
		parts = append(parts, "PoolPrivateYN eq true")
	}
	if req.Waterfront != nil && *req.Waterfront {
		parts = append(parts, "WaterfrontYN eq true")
	}
	if req.RemarksQuery != nil && strings.TrimSpace(*req.RemarksQuery) != "" {
		parts = append(parts, fmt.Sprintf("contains(PublicRemarks,'%s')", oDataEscape(*req.RemarksQuery)))
	}
	if req.PriceReducedWithinDays != nil && *req.PriceReducedWithinDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -*req.PriceReducedWithinDays)
		parts = append(parts, fmt.Sprintf("PriceChangeTimestamp gt %s", cutoff.Format(time.RFC3339)))
	}
	return strings.Join(parts, " and ")
}

func oDataEscape(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "'", "''")
}
