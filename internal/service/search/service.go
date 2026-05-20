package search

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/audit"
)

// Service implements hybrid POST /api/v1/search.
// Revenue impact: PostGIS mirror path cuts Bridge OData cost for Active/Pending queries.
type Service struct {
	cfg     config.Config
	db      *repository.DB
	postgis *PostgisSearch
	bridge  *BridgeLiveSearch
}

func NewService(cfg config.Config, db *repository.DB) *Service {
	return &Service{
		cfg:     cfg,
		db:      db,
		postgis: NewPostgisSearch(db),
		bridge:  NewBridgeLiveSearch(cfg, db),
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
		result, err = s.postgis.Search(c.Context(), feed, req)
	case RouteBridgeOnly:
		result, err = s.bridge.Search(c.Context(), c, feed, req)
	case RouteSplit:
		result, err = s.searchSplit(c, feed, req)
	default:
		result, err = s.postgis.Search(c.Context(), feed, req)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	audit.NewLogger(s.db).Log(c, "search.listings", intPtr(len(result.Results)))
	return c.JSON(result)
}

func (s *Service) searchSplit(c *fiber.Ctx, feed string, req SearchRequest) (SearchResult, error) {
	ap, err := s.postgis.Search(c.Context(), feed, req)
	if err != nil {
		return s.bridge.Search(c.Context(), c, feed, req)
	}
	closed := req
	closed.Statuses = []string{"Closed"}
	cl, err := s.bridge.Search(c.Context(), c, feed, closed)
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
