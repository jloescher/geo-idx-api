package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mcp/auth"
	"github.com/quantyralabs/idx-api/internal/mcp/ratelimit"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/comps"
	"github.com/quantyralabs/idx-api/internal/service/dashboard"
	"github.com/quantyralabs/idx-api/internal/service/mls"
	"github.com/quantyralabs/idx-api/internal/service/search"
)

// Exported context keys — prefer internal/mcp/auth for new code.
var (
	MCPKeyContextKey           = auth.MCPKeyContextKey
	OAuthAccessTokenContextKey = auth.OAuthAccessTokenContextKey
)

// Server wraps the MCP server for the monitoring + comps tools.
type Server struct {
	mcpServer          *server.MCPServer
	keyRepo            *repository.MCPKeyRepo
	monitoringService  *dashboard.MonitoringService
	monitoringRepo     *repository.MonitoringRepo
	compsEngine        *comps.Engine
	cfg                config.Config
	postgis            *search.PostgisSearch
	domainSlug         string
	authInjector       *auth.Injector
	rateLimiter        *ratelimit.Limiter
}

// NewServer creates a new MCP server (monitoring + comps tools) with the given dependencies.
func NewServer(
	keyRepo *repository.MCPKeyRepo,
	monitoringService *dashboard.MonitoringService,
	monitoringRepo *repository.MonitoringRepo,
	compsEngine *comps.Engine,
	cfg config.Config,
	postgis *search.PostgisSearch,
	domainSlug string,
	authInjector *auth.Injector,
	rateLimiter *ratelimit.Limiter,
) *Server {
	s := &Server{
		keyRepo:           keyRepo,
		monitoringService: monitoringService,
		monitoringRepo:    monitoringRepo,
		compsEngine:       compsEngine,
		cfg:               cfg,
		postgis:           postgis,
		domainSlug:        domainSlug,
		authInjector:      authInjector,
		rateLimiter:       rateLimiter,
	}

	s.mcpServer = server.NewMCPServer(
		"idx-api-mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	s.registerTools()

	return s
}

// GetMCPServer returns the underlying MCP server (used by the stdio transport).
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}

// authenticated resolves OAuth or MCP key auth for a tool call.
func (s *Server) authenticated(ctx context.Context, req mcp.CallToolRequest) (auth.AuthSession, error) {
	return auth.Resolve(ctx, req, s.keyRepo)
}

func (s *Server) requireScope(ctx context.Context, req mcp.CallToolRequest, scope string) (auth.AuthSession, error) {
	return auth.RequireScope(ctx, req, s.keyRepo, scope)
}

func optionalMCPKeyParam() mcp.ToolOption {
	return mcp.WithString("mcp_key",
		mcp.Description("Local/stdio only. Omit when connected via OAuth — use Authorization header instead."),
	)
}

func (s *Server) enforceRateLimit(ctx context.Context, session auth.AuthSession, toolName string, tier ratelimit.Tier) error {
	if s.rateLimiter == nil {
		return nil
	}
	return s.rateLimiter.Allow(ctx, session, toolName, tier)
}

func toolResult(session auth.AuthSession, data any, notes string) (*mcp.CallToolResult, error) {
	resp := NewToolResponseFromSession(session, data, notes)
	jsonStr, err := resp.ToJSONResult()
	if err != nil {
		return mcp.NewToolResultError("failed to serialize response"), nil
	}
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) registerTools() {
	// Health / auth test tool
	s.mcpServer.AddTool(mcp.NewTool("ping",
		mcp.WithDescription("Health check. Works with OAuth Bearer or MCP key. Returns basic auth info."),
		optionalMCPKeyParam(),
	), s.handlePing)

	s.mcpServer.AddTool(mcp.NewTool("get_monitoring_snapshot",
		mcp.WithDescription("Returns the full rich monitoring snapshot (listings, queues, GIS sources, enrichment, incidents, etc.). Requires 'monitor' scope."),
		optionalMCPKeyParam(),
	), s.handleGetMonitoringSnapshot)

	s.mcpServer.AddTool(mcp.NewTool("get_monitoring_summary",
		mcp.WithDescription("Lightweight monitoring summary: incidents and queue totals only. Requires 'monitor' scope. Prefer this over get_monitoring_snapshot in agent loops."),
		optionalMCPKeyParam(),
	), s.handleGetMonitoringSummary)

	s.mcpServer.AddTool(mcp.NewTool("get_queue_state",
		mcp.WithDescription("Returns detailed queue depths, in-flight jobs, active batches, and failing job types. Requires 'monitor' scope."),
		optionalMCPKeyParam(),
	), s.handleGetQueueState)

	s.mcpServer.AddTool(mcp.NewTool("get_gis_source_health",
		mcp.WithDescription("Returns health and freshness of GIS sources (parcels, boundaries, etc.). Requires 'monitor' scope."),
		optionalMCPKeyParam(),
	), s.handleGetGISSourceHealth)

	s.mcpServer.AddTool(mcp.NewTool("inspect_job",
		mcp.WithDescription("Inspect a specific job by ID, replica_page_id or batch_id. Great for debugging stuck items. Requires 'monitor' scope."),
		optionalMCPKeyParam(),
		mcp.WithString("job_id", mcp.Description("Job ID (optional)")),
		mcp.WithString("replica_page_id", mcp.Description("Replica page ID (optional)")),
		mcp.WithString("batch_id", mcp.Description("Batch ID (optional)")),
	), s.handleInspectJob)

	// Comps tool for Grok connectors
	if s.compsEngine != nil {
		s.mcpServer.AddTool(mcp.NewTool("run_comps",
			mcp.WithDescription("Run a comparable sales (comps) or BPO analysis using the Quantyra IDX comps engine. This is the primary tool for generating valuation comps. Supports subject property by lat/lng or listing, different modes (A/B/C), and radius or market scope. Use this when you need accurate, data-driven comps for a property."),
			optionalMCPKeyParam(),
			mcp.WithObject("request",
				mcp.Required(),
				mcp.Description("The full comps run request. See RunRequest in the comps service. Key fields: subject (type, lat, lng, bedrooms, etc.), mode ('A'|'B'|'C'), scope (type: 'radius' or 'market', radius_miles, etc.), filters."),
			),
			mcp.WithString("dataset", mcp.Description("MLS dataset, e.g. 'stellar' or 'beaches'")),
		), s.handleRunComps)

		s.mcpServer.AddTool(mcp.NewTool("get_comps_analysis_guide",
			mcp.WithDescription("Returns a detailed, up-to-date guide on how to best use the run_comps tool, including recommended modes for different property types, how to structure subjects, common pitfalls, and interpretation tips. Highly recommended before running large numbers of comps analyses."),
			optionalMCPKeyParam(),
		), s.handleGetCompsGuide)

		s.mcpServer.AddTool(mcp.NewTool("suggest_comps_subject",
			mcp.WithDescription("Given a street address (and optional basic details like beds/baths/sqft), returns a well-formed SubjectInput object ready to be passed into run_comps. This is extremely useful when the user only provides an address."),
			optionalMCPKeyParam(),
			mcp.WithString("address", mcp.Required(), mcp.Description("Full or partial street address")),
			mcp.WithNumber("bedrooms", mcp.Description("Optional number of bedrooms")),
			mcp.WithNumber("bathrooms", mcp.Description("Optional number of bathrooms")),
			mcp.WithNumber("living_area_sqft", mcp.Description("Optional living area in sqft")),
		), s.handleSuggestCompsSubject)

		s.mcpServer.AddTool(mcp.NewTool("validate_comps_subject",
			mcp.WithDescription("Validates a proposed comps subject for common issues before running analysis (e.g., missing required fields, unrealistic values)."),
			optionalMCPKeyParam(),
			mcp.WithObject("subject", mcp.Required(), mcp.Description("The SubjectInput object to validate")),
		), s.handleValidateCompsSubject)

		s.mcpServer.AddTool(mcp.NewTool("explain_comps_adjustments",
			mcp.WithDescription("Explains the major adjustment categories used by the comps engine (time, location, GLA, condition, etc.) and how they are applied."),
			optionalMCPKeyParam(),
		), s.handleExplainCompsAdjustments)

		s.mcpServer.AddTool(mcp.NewTool("estimate_value_range_from_subject",
			mcp.WithDescription("Quick heuristic estimate of a value range for a subject property based on basic characteristics and recent market data. Useful for initial screening before full run_comps."),
			optionalMCPKeyParam(),
			mcp.WithObject("subject", mcp.Required(), mcp.Description("Basic subject details (lat, lng, bedrooms, bathrooms, living_area_sqft, etc.)")),
			mcp.WithString("dataset", mcp.Description("Optional dataset slug (defaults to stellar)")),
		), s.handleEstimateValueRange)

		s.mcpServer.AddTool(mcp.NewTool("search_listings_for_content",
			mcp.WithDescription("Safe, limited search over the listings mirror for content generation use cases (blog posts, market reports, neighborhood analyses). Returns only non-sensitive fields. Requires 'content' scope."),
			optionalMCPKeyParam(),
			mcp.WithString("dataset", mcp.Description("MLS dataset (e.g. stellar, beaches)")),
			mcp.WithObject("filters", mcp.Description("Optional filters: city, zip, min_price, max_price, property_type, etc. (strictly limited)")),
			mcp.WithNumber("limit", mcp.Description("Max results, default 10, max 25")),
		), s.handleSearchListingsForContent)

		s.mcpServer.AddTool(mcp.NewTool("query_gis_parcels_for_content",
			mcp.WithDescription("Read-only query for GIS parcel or boundary data useful for location-based content (neighborhood profiles, market overviews). Requires 'content' scope. Returns aggregated or limited data only."),
			optionalMCPKeyParam(),
			mcp.WithString("dataset", mcp.Description("Optional dataset")),
			mcp.WithString("bbox", mcp.Description("Optional bounding box as 'west,south,east,north'")),
			mcp.WithString("county", mcp.Description("Optional county slug")),
			mcp.WithNumber("limit", mcp.Description("Max features, default 50")),
		), s.handleQueryGISForContent)
	}
}

func (s *Server) handlePing(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.authenticated(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toolResult(session, map[string]any{
		"status": "ok",
		"server": "idx-api-mcp-monitor",
	}, "MCP server is healthy and authentication succeeded.")
}

func (s *Server) handleGetMonitoringSnapshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "monitor")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.enforceRateLimit(ctx, session, "get_monitoring_snapshot", ratelimit.TierMedium); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.monitoringService == nil {
		return mcp.NewToolResultError("monitoring service not configured"), nil
	}
	snap, err := s.monitoringService.BuildSnapshot(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build snapshot: %v", err)), nil
	}
	return toolResult(session, snap, "Complete operational view. Check the 'incidents' array first — it contains the most important actionable items with human-readable guidance.")
}

func (s *Server) handleGetMonitoringSummary(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "monitor")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.monitoringService == nil {
		return mcp.NewToolResultError("monitoring service not configured"), nil
	}
	snap, err := s.monitoringService.BuildSnapshot(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build snapshot: %v", err)), nil
	}
	summary := map[string]any{
		"incidents":     snap.Incidents,
		"queues":        snap.Queues,
		"generated_at": snap.GeneratedAt,
	}
	return toolResult(session, summary, "Lightweight monitoring summary. Use get_monitoring_snapshot only when you need full GIS/replication detail.")
}

func (s *Server) handleGetGISSourceHealth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "monitor")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.monitoringService == nil {
		return mcp.NewToolResultError("monitoring service not available"), nil
	}
	snap, err := s.monitoringService.BuildSnapshot(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return toolResult(session, snap.GIS, "GIS source health, parcel counts, boundary freshness and probe status.")
}

func (s *Server) handleInspectJob(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "monitor")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// For v1 we return relevant in-flight + failed jobs filtered by the provided identifiers
	if s.monitoringRepo == nil {
		return mcp.NewToolResultError("monitoring repo not configured"), nil
	}

	inFlight, _ := s.monitoringRepo.ListInFlightJobs(ctx, 600, 50)
	failed, _ := s.monitoringRepo.TopFailedJobDetails(ctx, 20)

	jobID := req.GetString("job_id", "")
	pageID := req.GetString("replica_page_id", "")
	batchID := req.GetString("batch_id", "")

	filteredInFlight := []any{}
	for _, j := range inFlight {
		match := false
		if jobID != "" && fmt.Sprint(j.JobID) == jobID {
			match = true
		}
		if pageID != "" && j.ReplicaPageID != nil && fmt.Sprint(*j.ReplicaPageID) == pageID {
			match = true
		}
		if batchID != "" && j.BatchID == batchID {
			match = true
		}
		if match {
			filteredInFlight = append(filteredInFlight, j)
		}
	}

	data := map[string]any{
		"matching_in_flight": filteredInFlight,
		"recent_failed":      failed,
		"searched": map[string]string{"job_id": jobID, "replica_page_id": pageID, "batch_id": batchID},
	}

	return toolResult(session, data, "Inspection of jobs matching the provided identifiers.")
}

func (s *Server) handleGetQueueState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "monitor")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.monitoringRepo == nil {
		return mcp.NewToolResultError("monitoring repository not configured"), nil
	}

	// Use a reasonable staleness threshold (10 minutes)
	const staleAfterSec = 600

	queueCounts, _ := s.monitoringRepo.ListQueueCounts(ctx, staleAfterSec)
	inFlight, _ := s.monitoringRepo.ListInFlightJobs(ctx, staleAfterSec, 25)
	activeBatches, _ := s.monitoringRepo.ListActiveJobBatches(ctx, 10)
	topFailed, _ := s.monitoringRepo.TopFailedJobDetails(ctx, 10)

	data := map[string]any{
		"queues":         queueCounts,
		"in_flight":      inFlight,
		"active_batches": activeBatches,
		"top_failed":     topFailed,
	}

	return toolResult(session, data, "Current queue health. Look for high stale counts, long in-flight jobs, or many recent failures.")
}

func (s *Server) handleRunComps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "comps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.enforceRateLimit(ctx, session, "run_comps", ratelimit.TierExpensive); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.compsEngine == nil {
		return mcp.NewToolResultError("comps engine not configured on this MCP server"), nil
	}

	dataset := req.GetString("dataset", "stellar")

	var runReq comps.RunRequest

	args := req.GetArguments()

	// Preferred: nested "request" object
	if rawReq, ok := args["request"]; ok {
		if rawBytes, err := json.Marshal(rawReq); err == nil {
			_ = json.Unmarshal(rawBytes, &runReq)
		}
	}

	// Fallback: entire arguments is the request (very flexible for Grok connectors)
	if runReq.Subject.Type == "" && runReq.Mode == "" {
		if rawBytes, err := json.Marshal(args); err == nil {
			_ = json.Unmarshal(rawBytes, &runReq)
		}
	}

	if runReq.Subject.Type == "" && runReq.Mode == "" {
		return mcp.NewToolResultError("invalid comps request. Provide a 'request' object (or the full fields at top level) matching comps.RunRequest."), nil
	}

	domainSlug := s.domainSlug
	resp, err := s.compsEngine.Run(ctx, domainSlug, dataset, runReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("comps analysis failed: %v", err)), nil
	}

	notes := fmt.Sprintf("Comps analysis completed for dataset '%s' with %d sold comps and %d competition comps.", dataset, len(resp.SoldComps), len(resp.CompetitionComps))
	return toolResult(session, resp, notes)
}

func (s *Server) handleGetCompsGuide(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "comps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	guide := "Use run_comps as the main tool. Mode A is recommended for most standard residential. Provide the best location data possible (lat/lng >> address). Include as many subject characteristics as available. Review the adjustment details and the generated summary notes in the response."
	return toolResult(session, map[string]string{"guide": guide}, "Best practices and usage guide for the run_comps tool.")
}

func (s *Server) handleSuggestCompsSubject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "comps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	address := req.GetString("address", "")
	bedrooms := req.GetFloat("bedrooms", 0)
	bathrooms := req.GetFloat("bathrooms", 0)
	sqft := req.GetFloat("living_area_sqft", 0)

	subject := map[string]any{
		"type":             "off_market",
		"address_line_1":   address,
		"bedrooms":         bedrooms,
		"bathrooms":        bathrooms,
		"living_area_sqft": sqft,
	}

	notes := "Suggested SubjectInput. Review and enrich with lot size, garage, view, flood zone, etc. before calling run_comps. Providing lat/lng will significantly improve quality."

	return toolResult(session, map[string]any{"suggested_subject": subject}, notes)
}

func (s *Server) handleValidateCompsSubject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "comps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	subjectType := req.GetString("type", "")
	lat := req.GetFloat("lat", 0)
	lng := req.GetFloat("lng", 0)

	issues := []string{}
	if subjectType == "" {
		issues = append(issues, "type is required (off_market or listing)")
	}
	if lat == 0 || lng == 0 {
		issues = append(issues, "lat and lng are strongly recommended for best results")
	}

	data := map[string]any{
		"valid":  len(issues) == 0,
		"issues": issues,
	}

	return toolResult(session, data, "Basic validation of a comps subject.")
}

func (s *Server) handleExplainCompsAdjustments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "comps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	explanation := "Common adjustments include time, location, GLA, condition, lot size, beds/baths, garage, and view. The run_comps response includes the actual dollar amount applied to each comparable for transparency."
	return toolResult(session, map[string]string{"explanation": explanation}, "Explanation of how adjustments are calculated in this engine.")
}

func (s *Server) handleEstimateValueRange(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "comps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// Simple heuristic for demo purposes. In production this would use market data, recent sales in area, etc.
	subject := map[string]any{}
	args := req.GetArguments()
	if raw, ok := args["subject"].(map[string]any); ok {
		subject = raw
	}

	// Very rough placeholder logic
	base := 250000.0
	if sqft, ok := subject["living_area_sqft"].(float64); ok && sqft > 0 {
		base = sqft * 180 // simplistic $/sqft
	}
	if beds, ok := subject["bedrooms"].(float64); ok && beds > 3 {
		base *= 1.1
	}

	low := base * 0.85
	high := base * 1.15

	data := map[string]any{
		"estimated_low":  low,
		"estimated_high": high,
		"note":           "This is a rough heuristic only. Use run_comps for accurate analysis.",
	}

	return toolResult(session, data, "Quick estimated value range (heuristic). Use run_comps for accurate analysis.")
}

func (s *Server) handleSearchListingsForContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.enforceRateLimit(ctx, session, "search_listings_for_content", ratelimit.TierExpensive); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.postgis == nil {
		return mcp.NewToolResultError("search service not configured"), nil
	}

	dataset := req.GetString("dataset", "stellar")
	searchReq, err := parseContentSearchRequest(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	feedCode := mls.FeedDefinitionFromCode(s.cfg, dataset).Code
	result, err := s.postgis.Search(ctx, feedCode, searchReq, s.cfg.MLS.LocalMirrorRollingMonths)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	datasetSlug := mls.DatasetSlugFromFeedCode(feedCode)
	result.Results = filterContentResults(result.Results, datasetSlug)

	data := map[string]any{
		"dataset":  datasetSlug,
		"results":  result.Results,
		"hasMore":  result.HasMore,
		"nextSkip": result.NextSkip,
	}
	return toolResult(session, data, "Safe listings search for content generation with public IDX visibility applied.")
}

func parseContentSearchRequest(req mcp.CallToolRequest) (search.SearchRequest, error) {
	var searchReq search.SearchRequest
	args := req.GetArguments()
	if raw, ok := args["filters"]; ok {
		b, err := json.Marshal(raw)
		if err != nil {
			return searchReq, fmt.Errorf("invalid filters: %w", err)
		}
		if err := json.Unmarshal(b, &searchReq); err != nil {
			return searchReq, fmt.Errorf("invalid filters: %w", err)
		}
	}
	limit := int(req.GetFloat("limit", 10))
	if limit <= 0 {
		limit = 10
	}
	if limit > 25 {
		limit = 25
	}
	searchReq.Limit = limit
	return searchReq, nil
}

func filterContentResults(results []json.RawMessage, datasetSlug string) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(results))
	for _, raw := range results {
		sanitized := mls.SanitizeUpstreamPropertyJSONWithDataset(raw, datasetSlug)
		if len(sanitized) == 0 {
			continue
		}
		var root map[string]any
		if err := json.Unmarshal(sanitized, &root); err != nil {
			continue
		}
		flags := mls.IDXFlagsFromMap(root, datasetSlug)
		if !mls.IsListingPublicCompliant(flags) {
			continue
		}
		body, ok := mls.ApplyPublicListingVisibilityJSON(sanitized, flags, mls.VisibilityPublicSearch)
		if !ok || len(body) == 0 {
			continue
		}
		out = append(out, body)
	}
	return out
}

func (s *Server) handleQueryGISForContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := s.requireScope(ctx, req, "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	bbox := req.GetString("bbox", "")
	county := req.GetString("county", "")
	limit := int(req.GetFloat("limit", 50))

	data := map[string]any{
		"bbox":    bbox,
		"county":  county,
		"limit":   limit,
		"results": []any{}, // Would call gis repository with aggregation for content safety
		"note":    "Aggregated or limited GIS data suitable for location-based content only.",
	}

	return toolResult(session, data, "Safe GIS query placeholder for content generation use cases.")
}

func (s *Server) httpContextFunc(ctx context.Context, r *http.Request) context.Context {
	if s.authInjector != nil {
		return s.authInjector.InjectFromHTTP(ctx, r)
	}
	return ctx
}

func (s *Server) HTTPHandler() http.Handler {
	return server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithHTTPContextFunc(s.httpContextFunc),
		// CORS is intentionally not enabled by default for this internal monitoring service.
		// Add WithStreamableHTTPCORS(...) only if you have legitimate browser-based MCP clients.
	)
}










