package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/spark"
	"github.com/quantyralabs/idx-api/internal/service/cache"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// LiveSearch queries live upstream RESO for statuses not stored in the mirror (any MLS feed).
type LiveSearch struct {
	cfg        config.Config
	factory    *mlspoxy.Factory
	proxyCache *cache.ProxyCache
}

func NewLiveSearch(cfg config.Config, proxyCache *cache.ProxyCache, sparkLimiter spark.RateLimiter) *LiveSearch {
	return &LiveSearch{
		cfg:        cfg,
		factory:    mlspoxy.NewFactory(cfg, sparkLimiter),
		proxyCache: proxyCache,
	}
}

func (l *LiveSearch) Search(ctx context.Context, c *fiber.Ctx, feedCode string, req SearchRequest) (SearchResult, error) {
	feed := mlspoxy.Feed(c)
	endpoint, err := mlspoxy.LiveResoPropertyEndpoint(l.cfg, feed)
	if err != nil {
		return SearchResult{}, err
	}

	partition := cache.SearchPartition(mlsDomainSlug(c), feedCode)
	fp := searchFingerprint(req)
	datasetSlug := mls.DatasetSlugFromFeedCode(feedCode)
	if body, ok, err := l.proxyCache.Get(ctx, partition, fp); err == nil && ok {
		return parseSearchBody(body, datasetSlug)
	}

	u, err := url.Parse(endpoint.PropertyURL)
	if err != nil {
		return SearchResult{}, err
	}
	q := u.Query()
	if filter := buildODataFilter(req, datasetSlug); filter != "" {
		q.Set("$filter", filter)
	}
	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q.Set("$top", fmt.Sprintf("%d", limit))
	if req.Skip > 0 {
		q.Set("$skip", fmt.Sprintf("%d", req.Skip))
	}
	q.Set("$orderby", "ModificationTimestamp desc")
	u.RawQuery = q.Encode()

	status, body, _, err := l.factory.ForRequest(c).ProxyUpstream(c, u.String())
	if err != nil || status >= 400 {
		return SearchResult{}, err
	}
	_ = l.proxyCache.Put(ctx, partition, fp, body)
	return parseSearchBody(body, datasetSlug)
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

func buildODataFilter(req SearchRequest, datasetSlug string) string {
	var parts []string
	parts = append(parts, "(InternetEntireListingDisplayYN ne false)")
	if strings.EqualFold(datasetSlug, "stellar") {
		parts = append(parts, "(IDXParticipationYN ne false or IDXParticipationYN eq null)")
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
	if req.PriceReducedWithinDays != nil && *req.PriceReducedWithinDays > 0 {
		parts = append(parts, fmt.Sprintf("DaysOnMarket le %d", *req.PriceReducedWithinDays))
	}
	return strings.Join(parts, " and ")
}
