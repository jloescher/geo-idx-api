package comps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// Engine runs comparables analysis: Active/Pending from listings mirror; Closed from live upstream RESO.
// Revenue impact: comps/BPO/home value drives agent conversion on valuation tools.
type Engine struct {
	cfg  config.Config
	db   *repository.DB
	http *http.Client
}

func NewEngine(cfg config.Config, db *repository.DB) *Engine {
	timeout := cfg.Bridge.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if cfg.Spark.Timeout > timeout {
		timeout = cfg.Spark.Timeout
	}
	return &Engine{
		cfg:  cfg,
		db:   db,
		http: &http.Client{Timeout: timeout},
	}
}

func (e *Engine) Run(ctx context.Context, feedCode string, req RunRequest) (RunResponse, error) {
	if err := validateRequest(req); err != nil {
		return RunResponse{}, err
	}
	feed := mls.FeedDefinitionFromCode(e.cfg, feedCode)
	dataset := feed.Dataset
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	var subject SubjectProfile
	var err error
	if mode == "home_value" {
		subject, err = e.resolveHomeValueSubject(ctx, dataset, req.Subject)
	} else {
		subject, err = e.resolveSubject(ctx, dataset, req.Subject)
	}
	if err != nil {
		return RunResponse{}, err
	}
	resp := RunResponse{
		Success: true,
		Subject: subject,
		Metadata: map[string]any{
			"mode":     req.Mode,
			"dataset":  dataset,
			"feed":     feed.Code,
			"provider": feed.Provider,
		},
		Warnings: []string{},
	}

	switch mode {
	case "a", "b", "c", "d", "e",
		"A", "B", "C", "D", "E":
		return e.runSalesModes(ctx, feed, req, subject, resp)
	case "rent_hold_cashflow":
		return e.runRentHold(ctx, feed, req, subject, resp)
	case "flip_vs_hold":
		return e.runFlipVsHold(ctx, feed, req, subject, resp)
	case "appraiser_simulation":
		return e.runAppraiserSim(ctx, feed, req, subject, resp)
	case "bpo":
		return e.runBPOMode(ctx, feed, req, subject, resp)
	case "home_value":
		return e.runHomeValueMode(ctx, feed, req, subject, resp)
	default:
		return RunResponse{}, fmt.Errorf("unsupported mode %q", req.Mode)
	}
}

func (e *Engine) runSalesModes(ctx context.Context, feed mls.FeedDefinition, req RunRequest, subject SubjectProfile, resp RunResponse) (RunResponse, error) {
	f := req.Filters
	maxSold := 12
	if f.MaxSoldComps != nil {
		maxSold = *f.MaxSoldComps
	}
	sold, err := e.fetchSoldComps(ctx, feed, subject, req.Scope, f, maxSold)
	if err != nil {
		resp.Warnings = append(resp.Warnings, "sold comps partial: "+err.Error())
	}
	sold = applyAdjustments(subject, sold, f)
	resp.SoldComps = sold

	includeComp := strings.EqualFold(req.Mode, "C") || (f.IncludeActivePending != nil && *f.IncludeActivePending)
	if includeComp {
		maxComp := 20
		if f.MaxCompetitionComps != nil {
			maxComp = *f.MaxCompetitionComps
		}
		comp, err := e.findMirrorComps(ctx, feed.Dataset, subject, []string{"active", "pending"}, req.Scope, f, maxComp, false)
		if err != nil {
			resp.Warnings = append(resp.Warnings, err.Error())
		} else {
			resp.CompetitionComps = comp
		}
	}
	months := 12
	if f.SoldMonthsBack != nil {
		months = *f.SoldMonthsBack
	}
	resp.MarketConditions = marketConditions(sold, months)
	if f.IncludeOverpriced != nil && *f.IncludeOverpriced {
		resp.OverpricedSignals = overpricedSignals(resp.CompetitionComps, sold)
	}
	if strings.EqualFold(req.Mode, "D") || strings.EqualFold(req.Mode, "E") {
		resp.SoldComps = scoreKeywords(resp.SoldComps, req.Keywords)
	}
	return resp, nil
}

func scoreKeywords(comps []CompRecord, keywords []string) []CompRecord {
	if len(keywords) == 0 {
		return comps
	}
	out := make([]CompRecord, len(comps))
	for i, c := range comps {
		text := strings.ToLower(string(c.Property))
		var score float64
		for _, kw := range keywords {
			kw = strings.ToLower(strings.TrimSpace(kw))
			if kw != "" && strings.Contains(text, kw) {
				score++
			}
		}
		c.KeywordScore = score
		out[i] = c
	}
	return out
}

func validateRequest(req RunRequest) error {
	if strings.TrimSpace(req.Mode) == "" {
		return fmt.Errorf("mode is required")
	}
	if strings.TrimSpace(req.Scope.Type) == "" {
		return fmt.Errorf("scope.type is required")
	}
	if req.Scope.Type == "radius" && (req.Scope.RadiusMiles == nil || *req.Scope.RadiusMiles <= 0) {
		return fmt.Errorf("scope.radius_miles is required for radius scope")
	}
	if req.Scope.Type == "zip" && len(req.Scope.PostalCodes) == 0 {
		return fmt.Errorf("scope.postal_codes is required for zip scope")
	}
	return nil
}

// ParseRentalParams decodes rental_params JSON.
func ParseRentalParams(raw json.RawMessage) map[string]float64 {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	out := make(map[string]float64)
	for k, v := range m {
		out[k] = num(v)
	}
	return out
}
