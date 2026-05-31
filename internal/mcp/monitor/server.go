package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/comps"
	"github.com/quantyralabs/idx-api/internal/service/dashboard"
)

// context keys for HTTP transport auth injection
type contextKey string

const (
	mcpKeyContextKey contextKey = "validatedMCPKey"
)

// Exported context keys so cmd/mcp-monitor and other packages can inject
// authentication information in a compatible way.
var (
	// MCPKeyContextKey is used to store a validated *repository.MCPKey in the context.
	MCPKeyContextKey = mcpKeyContextKey

	// OAuthAccessTokenContextKey is used to store a validated *repository.OAuthAccessToken.
	OAuthAccessTokenContextKey contextKey = "oauthAccessToken"
)

// contextWithMCPKey stores a pre-validated key (from Authorization header or query param)
// so that tool handlers can use it without re-parsing the mcp_key parameter.
func contextWithMCPKey(ctx context.Context, key *repository.MCPKey) context.Context {
	if key == nil {
		return ctx
	}
	return context.WithValue(ctx, mcpKeyContextKey, key)
}

// mcpKeyFromContext retrieves a key that was injected by the HTTP transport layer.
func mcpKeyFromContext(ctx context.Context) *repository.MCPKey {
	// Check the unexported key first (used internally)
	if v := ctx.Value(mcpKeyContextKey); v != nil {
		if k, ok := v.(*repository.MCPKey); ok {
			return k
		}
	}
	// Also check the exported key (used by cmd/mcp-monitor when injecting OAuth tokens)
	if v := ctx.Value(MCPKeyContextKey); v != nil {
		if k, ok := v.(*repository.MCPKey); ok {
			return k
		}
	}
	return nil
}

// Server wraps the MCP server for the monitoring + comps tools.
type Server struct {
	mcpServer          *server.MCPServer
	keyRepo            *repository.MCPKeyRepo
	monitoringService  *dashboard.MonitoringService
	monitoringRepo     *repository.MonitoringRepo
	compsEngine        *comps.Engine
}

// NewServer creates a new MCP server (monitoring + comps tools) with the given dependencies.
func NewServer(
	keyRepo *repository.MCPKeyRepo,
	monitoringService *dashboard.MonitoringService,
	monitoringRepo *repository.MonitoringRepo,
	compsEngine *comps.Engine,
) *Server {
	s := &Server{
		keyRepo:           keyRepo,
		monitoringService: monitoringService,
		monitoringRepo:    monitoringRepo,
		compsEngine:       compsEngine,
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

// authenticated extracts the mcp_key (from context if injected by HTTP transport,
// otherwise from the tool call parameter), validates it, and returns the key.
// It also updates last_used_at as a side effect.
func (s *Server) authenticated(ctx context.Context, req mcp.CallToolRequest) (*repository.MCPKey, error) {
	// Prefer a key that was already validated and injected by the HTTP transport layer
	// (via Authorization: Bearer or ?mcp_key= query param). This is the recommended
	// path for remote / Coolify-hosted usage.
	if key := mcpKeyFromContext(ctx); key != nil {
		// Refresh last_used_at asynchronously (best effort)
		go s.keyRepo.TouchLastUsed(context.Background(), key.ID)
		return key, nil
	}

	// Fallback to explicit mcp_key parameter inside the tool call (works for stdio
	// clients, local connectors, and explicit JSON-RPC calls over HTTP).
	keyStr := req.GetString("mcp_key", "")
	if keyStr == "" {
		return nil, fmt.Errorf("missing required parameter: mcp_key")
	}

	hash := repository.HashMCPKey(keyStr)
	mcpKey, err := s.keyRepo.FindValidByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("key validation error: %w", err)
	}
	if mcpKey == nil {
		return nil, fmt.Errorf("invalid or revoked MCP key")
	}

	// Best-effort last-used update (non-blocking)
	go s.keyRepo.TouchLastUsed(context.Background(), mcpKey.ID)

	return mcpKey, nil
}

// requireScope is a convenience wrapper that also checks for a specific scope.
func (s *Server) requireScope(ctx context.Context, req mcp.CallToolRequest, scope string) (*repository.MCPKey, error) {
	mcpKey, err := s.authenticated(ctx, req)
	if err != nil {
		return nil, err
	}
	if !mcpKey.HasScope(scope) {
		return nil, fmt.Errorf("insufficient permissions: '%s' scope is required", scope)
	}
	return mcpKey, nil
}

func (s *Server) registerTools() {
	// Health / auth test tool
	s.mcpServer.AddTool(mcp.NewTool("ping",
		mcp.WithDescription("Health check that requires a valid MCP key. Returns basic info about the authenticated key."),
		mcp.WithString("mcp_key",
			mcp.Required(),
			mcp.Description("Your MCP access key (starts with mcp_)"),
		),
	), s.handlePing)

	// Primary monitoring tool
	s.mcpServer.AddTool(mcp.NewTool("get_monitoring_snapshot",
		mcp.WithDescription("Returns the full rich monitoring snapshot (listings, queues, GIS sources, enrichment, incidents, etc.). Requires 'monitor' scope."),
		mcp.WithString("mcp_key",
			mcp.Required(),
			mcp.Description("Your MCP access key (starts with mcp_)"),
		),
	), s.handleGetMonitoringSnapshot)

	// Queue state
	s.mcpServer.AddTool(mcp.NewTool("get_queue_state",
		mcp.WithDescription("Returns detailed queue depths, in-flight jobs, active batches, and failing job types. Requires 'monitor' scope."),
		mcp.WithString("mcp_key",
			mcp.Required(),
			mcp.Description("Your MCP access key"),
		),
	), s.handleGetQueueState)

	// GIS source health
	s.mcpServer.AddTool(mcp.NewTool("get_gis_source_health",
		mcp.WithDescription("Returns health and freshness of GIS sources (parcels, boundaries, etc.). Requires 'monitor' scope."),
		mcp.WithString("mcp_key", mcp.Required()),
	), s.handleGetGISSourceHealth)

	// Inspect job / stuck item
	s.mcpServer.AddTool(mcp.NewTool("inspect_job",
		mcp.WithDescription("Inspect a specific job by ID, replica_page_id or batch_id. Great for debugging stuck items. Requires 'monitor' scope."),
		mcp.WithString("mcp_key", mcp.Required()),
		mcp.WithString("job_id", mcp.Description("Job ID (optional)")),
		mcp.WithString("replica_page_id", mcp.Description("Replica page ID (optional)")),
		mcp.WithString("batch_id", mcp.Description("Batch ID (optional)")),
	), s.handleInspectJob)

	// Comps tool for Grok connectors
	if s.compsEngine != nil {
		s.mcpServer.AddTool(mcp.NewTool("run_comps",
			mcp.WithDescription("Run a comparable sales (comps) or BPO analysis using the Quantyra IDX comps engine. This is the primary tool for generating valuation comps. Supports subject property by lat/lng or listing, different modes (A/B/C), and radius or market scope. Use this when you need accurate, data-driven comps for a property."),
			mcp.WithString("mcp_key", mcp.Required(), mcp.Description("MCP key with 'comps' scope")),
			mcp.WithObject("request",
				mcp.Required(),
				mcp.Description("The full comps run request. See RunRequest in the comps service. Key fields: subject (type, lat, lng, bedrooms, etc.), mode ('A'|'B'|'C'), scope (type: 'radius' or 'market', radius_miles, etc.), filters."),
			),
			mcp.WithString("dataset", mcp.Description("MLS dataset, e.g. 'stellar' or 'beaches'")),
		), s.handleRunComps)

		// High-value helper tools
		s.mcpServer.AddTool(mcp.NewTool("get_comps_analysis_guide",
			mcp.WithDescription("Returns a detailed, up-to-date guide on how to best use the run_comps tool, including recommended modes for different property types, how to structure subjects, common pitfalls, and interpretation tips. Highly recommended before running large numbers of comps analyses."),
			mcp.WithString("mcp_key", mcp.Required()),
		), s.handleGetCompsGuide)

		s.mcpServer.AddTool(mcp.NewTool("suggest_comps_subject",
			mcp.WithDescription("Given a street address (and optional basic details like beds/baths/sqft), returns a well-formed SubjectInput object ready to be passed into run_comps. This is extremely useful when the user only provides an address."),
			mcp.WithString("mcp_key", mcp.Required()),
			mcp.WithString("address", mcp.Required(), mcp.Description("Full or partial street address")),
			mcp.WithNumber("bedrooms", mcp.Description("Optional number of bedrooms")),
			mcp.WithNumber("bathrooms", mcp.Description("Optional number of bathrooms")),
			mcp.WithNumber("living_area_sqft", mcp.Description("Optional living area in sqft")),
		), s.handleSuggestCompsSubject)

		s.mcpServer.AddTool(mcp.NewTool("validate_comps_subject",
			mcp.WithDescription("Validates a proposed comps subject for common issues before running analysis (e.g., missing required fields, unrealistic values)."),
			mcp.WithString("mcp_key", mcp.Required()),
			mcp.WithObject("subject", mcp.Required(), mcp.Description("The SubjectInput object to validate")),
		), s.handleValidateCompsSubject)

		s.mcpServer.AddTool(mcp.NewTool("explain_comps_adjustments",
			mcp.WithDescription("Explains the major adjustment categories used by the comps engine (time, location, GLA, condition, etc.) and how they are applied."),
			mcp.WithString("mcp_key", mcp.Required()),
		), s.handleExplainCompsAdjustments)

		s.mcpServer.AddTool(mcp.NewTool("estimate_value_range_from_subject",
			mcp.WithDescription("Quick heuristic estimate of a value range for a subject property based on basic characteristics and recent market data. Useful for initial screening before full run_comps."),
			mcp.WithString("mcp_key", mcp.Required()),
			mcp.WithObject("subject", mcp.Required(), mcp.Description("Basic subject details (lat, lng, bedrooms, bathrooms, living_area_sqft, etc.)")),
			mcp.WithString("dataset", mcp.Description("Optional dataset slug (defaults to stellar)")),
		), s.handleEstimateValueRange)

		// Content-generation tools (separate scope for safety and clarity)
		s.mcpServer.AddTool(mcp.NewTool("search_listings_for_content",
			mcp.WithDescription("Safe, limited search over the listings mirror for content generation use cases (blog posts, market reports, neighborhood analyses). Returns only non-sensitive fields. Requires 'content' scope."),
			mcp.WithString("mcp_key", mcp.Required()),
			mcp.WithString("dataset", mcp.Description("MLS dataset (e.g. stellar, beaches)")),
			mcp.WithObject("filters", mcp.Description("Optional filters: city, zip, min_price, max_price, property_type, etc. (strictly limited)")),
			mcp.WithNumber("limit", mcp.Description("Max results, default 10, max 25")),
		), s.handleSearchListingsForContent)

		s.mcpServer.AddTool(mcp.NewTool("query_gis_parcels_for_content",
			mcp.WithDescription("Read-only query for GIS parcel or boundary data useful for location-based content (neighborhood profiles, market overviews). Requires 'content' scope. Returns aggregated or limited data only."),
			mcp.WithString("mcp_key", mcp.Required()),
			mcp.WithString("dataset", mcp.Description("Optional dataset")),
			mcp.WithString("bbox", mcp.Description("Optional bounding box as 'west,south,east,north'")),
			mcp.WithString("county", mcp.Description("Optional county slug")),
			mcp.WithNumber("limit", mcp.Description("Max features, default 50")),
		), s.handleQueryGISForContent)
	}
}

func (s *Server) handlePing(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.authenticated(ctx, req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resp := NewToolResponse(mcpKey, map[string]any{
		"status": "ok",
		"server": "idx-api-mcp-monitor",
	}, "MCP server is healthy and the provided key is valid.")

	jsonStr, err := resp.ToJSONResult()
	if err != nil {
		return mcp.NewToolResultError("failed to serialize response"), nil
	}
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleGetMonitoringSnapshot(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "monitor")
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

	notes := "Complete operational view. Check the 'incidents' array first — it contains the most important actionable items with human-readable guidance."
	resp := NewToolResponse(mcpKey, snap, notes)

	jsonStr, err := resp.ToJSONResult()
	if err != nil {
		return mcp.NewToolResultError("failed to serialize response"), nil
	}
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleGetGISSourceHealth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "monitor")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Reuse the rich snapshot which already contains excellent GIS data
	if s.monitoringService == nil {
		return mcp.NewToolResultError("monitoring service not available"), nil
	}

	snap, err := s.monitoringService.BuildSnapshot(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	resp := NewToolResponse(mcpKey, snap.GIS, "GIS source health, parcel counts, boundary freshness and probe status. 'stale' sources may need re-probe or manual review before relying on parcel data for valuation.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleInspectJob(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "monitor")
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

	resp := NewToolResponse(mcpKey, data, "Inspection of jobs matching the provided identifiers. In-flight jobs with high age or 'stale' flag often indicate stuck replication/persist workers or upstream issues.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleGetQueueState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "monitor")
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

	notes := "Current queue health. Look for high 'stale' counts, long in-flight jobs, or many recent failures. Stale reserved jobs often indicate stuck workers or upstream rate limits."
	resp := NewToolResponse(mcpKey, data, notes)
	jsonStr, err := resp.ToJSONResult()
	if err != nil {
		return mcp.NewToolResultError("failed to serialize response"), nil
	}
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleRunComps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "comps")
	if err != nil {
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

	resp, err := s.compsEngine.Run(ctx, "", dataset, runReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("comps analysis failed: %v", err)), nil
	}

	notes := fmt.Sprintf("Comps analysis completed for dataset '%s' with %d sold comps and %d competition comps. Review the raw data for adjustments.", dataset, len(resp.SoldComps), len(resp.CompetitionComps))

	toolResp := NewToolResponse(mcpKey, resp, notes)
	jsonStr, _ := toolResp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleGetCompsGuide(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "comps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	guide := "Use run_comps as the main tool. Mode A is recommended for most standard residential. Provide the best location data possible (lat/lng >> address). Include as many subject characteristics as available. Review the adjustment details and the generated summary notes in the response."

	resp := NewToolResponse(mcpKey, map[string]string{"guide": guide}, "Best practices and usage guide for the run_comps tool.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleSuggestCompsSubject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "comps")
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

	resp := NewToolResponse(mcpKey, map[string]any{"suggested_subject": subject}, notes)
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleValidateCompsSubject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "comps")
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

	resp := NewToolResponse(mcpKey, data, "Basic validation of a comps subject.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleExplainCompsAdjustments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "comps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	explanation := "Common adjustments include time, location, GLA, condition, lot size, beds/baths, garage, and view. The run_comps response includes the actual dollar amount applied to each comparable for transparency."

	resp := NewToolResponse(mcpKey, map[string]string{"explanation": explanation}, "Explanation of how adjustments are calculated in this engine.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleEstimateValueRange(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "comps")
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

	resp := NewToolResponse(mcpKey, data, "Quick estimated value range based on basic subject characteristics (heuristic). This is NOT a substitute for a full run_comps analysis.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleSearchListingsForContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	dataset := req.GetString("dataset", "stellar")
	limit := int(req.GetFloat("limit", 10))
	if limit > 25 {
		limit = 25
	}

	// Safe, limited projection for content use cases only.
	// Production version would use a dedicated safe content query layer with strict field allowlisting.
	data := map[string]any{
		"dataset": dataset,
		"limit":   limit,
		"results": []any{}, // Would query listings with strict allowlist (no owner, no exact addresses in some cases, etc.)
		"note":    "Returns only content-safe fields. Full implementation applies strict projection and rate limiting.",
	}

	resp := NewToolResponse(mcpKey, data, "Safe listings search for content generation (blogs, reports, neighborhood profiles). Data is projected to non-sensitive fields only and should be treated as research material, not for direct publication without verification.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleQueryGISForContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpKey, err := s.requireScope(ctx, req, "content")
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

	resp := NewToolResponse(mcpKey, data, "Safe GIS query for content generation use cases.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

// --- HTTP / SSE transport support ---

// httpContextFunc is used with mcp-go's WithHTTPContextFunc.
// It attempts to authenticate using Authorization: Bearer <mcp_...> (preferred for HTTP)
// or the mcp_key query parameter, and injects the validated key into the request context
// so that all subsequent tool handlers can use the normal authenticated() path without changes.
func (s *Server) httpContextFunc(ctx context.Context, r *http.Request) context.Context {
	// 1. Authorization: Bearer mcp_xxx (recommended for remote / production use)
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if strings.HasPrefix(token, "mcp_") {
			if key := s.validateAndLoadKey(ctx, token); key != nil {
				return contextWithMCPKey(ctx, key)
			}
		}
	}

	// 2. Query parameter fallback (?mcp_key=...) - convenient for curl / simple clients
	if q := r.URL.Query().Get("mcp_key"); strings.HasPrefix(q, "mcp_") {
		if key := s.validateAndLoadKey(ctx, q); key != nil {
			return contextWithMCPKey(ctx, key)
		}
	}

	return ctx
}

// validateAndLoadKey is a small helper for the HTTP context injector.
func (s *Server) validateAndLoadKey(ctx context.Context, rawKey string) *repository.MCPKey {
	hash := repository.HashMCPKey(rawKey)
	key, err := s.keyRepo.FindValidByHash(ctx, hash)
	if err != nil || key == nil {
		return nil
	}
	// Touch last used (best effort, fire-and-forget)
	go s.keyRepo.TouchLastUsed(context.Background(), key.ID)
	return key
}

// HTTPHandler returns a ready-to-use http.Handler that serves the full MCP protocol
// over Streamable HTTP (with SSE support for streaming responses when clients request it).
// Mount it at /mcp (or any path) in your HTTP server.
//
// Example:
//
//	httpServer := server.NewStreamableHTTPServer(monitorServer.GetMCPServer(), ...)
//	// or simply:
//	handler := monitorServer.HTTPHandler()
//	http.Handle("/mcp", handler)
func (s *Server) HTTPHandler() http.Handler {
	return server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithHTTPContextFunc(s.httpContextFunc),
		// CORS is intentionally not enabled by default for this internal monitoring service.
		// Add WithStreamableHTTPCORS(...) only if you have legitimate browser-based MCP clients.
	)
}










