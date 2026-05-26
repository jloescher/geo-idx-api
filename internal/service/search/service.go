package search

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/images"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/audit"
	"github.com/quantyralabs/idx-api/internal/service/cache"
)

// Service implements hybrid POST /api/v1/search.
// Revenue impact: PostGIS mirror path cuts live upstream OData cost for Active/Pending queries.
type Service struct {
	cfg      config.Config
	db       *repository.DB
	postgis  *PostgisSearch
	upstream *LiveSearch
}

func NewService(cfg config.Config, db *repository.DB, proxyCache *cache.ProxyCache) *Service {
	return &Service{
		cfg:      cfg,
		db:       db,
		postgis:  NewPostgisSearch(db),
		upstream: NewLiveSearch(cfg, proxyCache),
	}
}

func (s *Service) Handle(c *fiber.Ctx) error {
	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid search body")
	}
	feed, _ := c.Locals(ctxkeys.MLSFeedCode).(string)
	mode := DecideRoute(req)
	var result SearchResult
	var err error
	switch mode {
	case RoutePostgresOnly:
		result, err = s.postgis.Search(c.Context(), feed, req, s.cfg.MLS.LocalMirrorRollingMonths)
	case RouteUpstreamOnly:
		result, err = s.upstream.Search(c.Context(), c, feed, req)
	case RouteSplit:
		result, err = s.searchSplit(c, feed, req)
	default:
		result, err = s.postgis.Search(c.Context(), feed, req, s.cfg.MLS.LocalMirrorRollingMonths)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	rewriter := images.NewRewriter(s.cfg)
	feedDef := mlspoxy.Feed(c)
	for i, r := range result.Results {
		result.Results[i] = json.RawMessage(images.RewriteBytes(rewriter, []byte(r), feedDef.Dataset, ""))
	}
	audit.NewLogger(s.db).Log(c, "search.listings", intPtr(len(result.Results)), nil)
	return c.JSON(result)
}

func (s *Service) searchSplit(c *fiber.Ctx, feed string, req SearchRequest) (SearchResult, error) {
	ap, err := s.postgis.Search(c.Context(), feed, req, s.cfg.MLS.LocalMirrorRollingMonths)
	if err != nil {
		return s.upstream.Search(c.Context(), c, feed, req)
	}
	closed := req
	closed.Statuses = []string{"Closed"}
	cl, err := s.upstream.Search(c.Context(), c, feed, closed)
	if err != nil {
		return ap, nil
	}
	return MergeResults(ap, cl), nil
}

func intPtr(n int) *int { return &n }

// SearchRequest mirrors Laravel SearchRequest JSON.
type SearchRequest struct {
	Statuses               []string `json:"statuses"`
	ActiveOnly             *bool    `json:"active_only"`
	MinPrice               *float64 `json:"min_price"`
	MaxPrice               *float64 `json:"max_price"`
	BedsMin                *int     `json:"beds_min"`
	BathsMin               *float64 `json:"baths_min"`
	LivingAreaMin          *int     `json:"living_area_min"`
	LivingAreaMax          *int     `json:"living_area_max"`
	LotSizeAcresMin        *float64 `json:"lot_size_acres_min"`
	YearBuiltMin           *int     `json:"year_built_min"`
	PropertyType           *string  `json:"property_type"`
	PropertySubType        *string  `json:"property_sub_type"`
	City                   *string  `json:"city"`
	PostalCode             *string  `json:"postal_code"`
	PoolPrivate            *bool    `json:"pool_private"`
	Waterfront             *bool    `json:"waterfront"`
	Lat                    *float64 `json:"lat"`
	Lng                    *float64 `json:"lng"`
	RadiusMiles            *float64 `json:"radius_miles"`
	PriceReducedWithinDays *int     `json:"price_reduced_within_days"`
	LowRiskFloodzone       *bool    `json:"low_risk_floodzone"`
	MinMonthlyFees         *float64 `json:"min_monthly_fees"`
	MaxMonthlyFees         *float64 `json:"max_monthly_fees"`
	Skip                   int      `json:"skip"`
	Limit                  int      `json:"limit"`
}

type SearchResult struct {
	Results  []json.RawMessage `json:"results"`
	HasMore  bool              `json:"hasMore"`
	NextSkip int               `json:"nextSkip"`
}
