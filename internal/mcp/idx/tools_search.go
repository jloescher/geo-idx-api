package idx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/quantyralabs/idx-api/internal/mcp/auth"
	"github.com/quantyralabs/idx-api/internal/mcp/ratelimit"
	"github.com/quantyralabs/idx-api/internal/mcp/shared"
	"github.com/quantyralabs/idx-api/internal/service/mls"
	"github.com/quantyralabs/idx-api/internal/service/search"
)

const (
	maxSearchLimit     = 25
	defaultSearchLimit = 10
)

func optionalMCPKey() mcp.ToolOption {
	return mcp.WithString("mcp_key",
		mcp.Description("Local/stdio only. Omit when connected via OAuth — use Authorization header instead."),
	)
}

func (s *Server) registerSearchTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("search_listings",
		mcp.WithDescription("Search mirrored Active/Pending listings via PostGIS. Closed or split queries delegate to idx-api-web when configured. Requires 'api' or 'content' scope."),
		optionalMCPKey(),
		mcp.WithString("dataset", mcp.Description("MLS dataset slug, e.g. stellar or beaches (default stellar)")),
		mcp.WithObject("filters", mcp.Description("Search filters matching POST /api/v1/search body (statuses, min_price, city, etc.)")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 10, max 25)")),
		mcp.WithString("response_profile", mcp.Description("summary or standard (default standard)")),
	), s.handleSearchListings)

	mcpServer.AddTool(mcp.NewTool("search_listings_live",
		mcp.WithDescription("Force live upstream search via idx-api-web (Closed, split, price_reduced_within_days). Requires 'api' scope and MCP_API_* env."),
		optionalMCPKey(),
		mcp.WithString("dataset", mcp.Description("MLS dataset slug")),
		mcp.WithObject("filters", mcp.Required(), mcp.Description("Full search request body")),
	), s.handleSearchListingsLive)
}

func (s *Server) handleSearchListings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := auth.RequireAnyScope(ctx, req, s.keyRepo, "api", "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Allow(ctx, session, "search_listings", ratelimit.TierExpensive); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	dataset := req.GetString("dataset", "stellar")
	searchReq, err := parseSearchRequest(req, defaultSearchLimit, maxSearchLimit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	feedCode := mls.FeedDefinitionFromCode(s.cfg, dataset).Code
	mode := search.DecideRoute(searchReq)

	var result search.SearchResult
	switch mode {
	case search.RouteUpstreamOnly, search.RouteSplit:
		if s.apiClient == nil || !s.apiClient.Enabled() {
			return mcp.NewToolResultError("this search requires live MLS data; configure MCP_API_INTERNAL_URL and MCP_API_SERVICE_TOKEN"), nil
		}
		body, _, err := s.apiClient.Post(ctx, "/api/v1/search", url.Values{"dataset": {dataset}}, searchReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("live search failed: %v", err)), nil
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return mcp.NewToolResultError("failed to decode live search response"), nil
		}
	default:
		result, err = s.postgis.Search(ctx, feedCode, searchReq, s.cfg.MLS.LocalMirrorRollingMonths)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}
	}

	datasetSlug := mls.DatasetSlugFromFeedCode(feedCode)
	result.Results = filterSearchResultsForPublic(result.Results, datasetSlug)

	data := map[string]any{
		"results":  result.Results,
		"hasMore":  result.HasMore,
		"nextSkip": result.NextSkip,
		"dataset":  datasetSlug,
		"route":    routeName(mode),
	}
	resp := shared.NewToolResponseFromSession(session, data, "Listing search results with public IDX visibility applied.")
	jsonStr, err := resp.ToJSONResult()
	if err != nil {
		return mcp.NewToolResultError("failed to serialize response"), nil
	}
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleSearchListingsLive(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := auth.RequireScope(ctx, req, s.keyRepo, "api")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.apiClient == nil || !s.apiClient.Enabled() {
		return mcp.NewToolResultError("MCP API client not configured"), nil
	}

	dataset := req.GetString("dataset", "stellar")
	searchReq, err := parseSearchRequest(req, defaultSearchLimit, maxSearchLimit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	body, _, err := s.apiClient.Post(ctx, "/api/v1/search", url.Values{"dataset": {dataset}}, searchReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("live search failed: %v", err)), nil
	}

	var result search.SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return mcp.NewToolResultError("failed to decode live search response"), nil
	}

	resp := shared.NewToolResponseFromSession(session, result, "Live MLS search via idx-api-web.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func parseSearchRequest(req mcp.CallToolRequest, defaultLimit, maxLimit int) (search.SearchRequest, error) {
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
	if lim := int(req.GetFloat("limit", 0)); lim > 0 {
		searchReq.Limit = lim
	}
	if searchReq.Limit <= 0 {
		searchReq.Limit = defaultLimit
	}
	if searchReq.Limit > maxLimit {
		searchReq.Limit = maxLimit
	}
	return searchReq, nil
}

func routeName(mode search.HybridSearchRouteMode) string {
	switch mode {
	case search.RoutePostgresOnly:
		return "postgis"
	case search.RouteUpstreamOnly:
		return "upstream"
	case search.RouteSplit:
		return "split"
	default:
		return "unknown"
	}
}

func filterSearchResultsForPublic(results []json.RawMessage, datasetSlug string) []json.RawMessage {
	return searchFilterPublic(results, datasetSlug)
}

// searchFilterPublic wraps the unexported filter in search package via duplicate thin call.
func searchFilterPublic(results []json.RawMessage, datasetSlug string) []json.RawMessage {
	// Re-implement minimal path: search package filter is unexported; use same logic inline.
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
