package idx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/quantyralabs/idx-api/internal/mcp/auth"
	"github.com/quantyralabs/idx-api/internal/mcp/shared"
)

func (s *Server) registerGISTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("query_gis",
		mcp.WithDescription("Query GIS parcel data via idx-api-web GET /api/v1/gis. Requires 'api' scope."),
		optionalMCPKey(),
		mcp.WithString("dataset", mcp.Description("MLS dataset slug")),
		mcp.WithString("bbox", mcp.Description("Bounding box west,south,east,north")),
		mcp.WithString("county", mcp.Description("County slug")),
	), s.handleQueryGIS)

	mcpServer.AddTool(mcp.NewTool("autocomplete_cities",
		mcp.WithDescription("Autocomplete city|county names from GIS catalog. Requires 'api' scope."),
		optionalMCPKey(),
		mcp.WithString("q", mcp.Required(), mcp.Description("Search prefix")),
		mcp.WithNumber("limit", mcp.Description("Max suggestions (default 10)")),
	), s.handleAutocompleteCities)

	mcpServer.AddTool(mcp.NewTool("autocomplete_counties",
		mcp.WithDescription("Autocomplete county names from GIS catalog. Requires 'api' scope."),
		optionalMCPKey(),
		mcp.WithString("q", mcp.Required(), mcp.Description("Search prefix")),
		mcp.WithNumber("limit", mcp.Description("Max suggestions (default 10)")),
	), s.handleAutocompleteCounties)
}

func (s *Server) handleQueryGIS(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := auth.RequireScope(ctx, req, s.keyRepo, "api")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.apiClient == nil || !s.apiClient.Enabled() {
		return mcp.NewToolResultError("MCP API client not configured"), nil
	}

	q := url.Values{}
	if ds := req.GetString("dataset", ""); ds != "" {
		q.Set("dataset", ds)
	}
	if bbox := req.GetString("bbox", ""); bbox != "" {
		q.Set("bbox", bbox)
	}
	if county := req.GetString("county", ""); county != "" {
		q.Set("county", county)
	}

	body, _, err := s.apiClient.Get(ctx, "/api/v1/gis", q)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("query_gis failed: %v", err)), nil
	}

	resp := shared.NewToolResponseFromSession(session, json.RawMessage(body), "GIS parcel query via idx-api-web.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleAutocompleteCities(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := auth.RequireScope(ctx, req, s.keyRepo, "api")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	q := req.GetString("q", "")
	limit := int(req.GetFloat("limit", 10))
	if limit <= 0 || limit > 25 {
		limit = 10
	}

	results, err := s.autocomplete.Cities(ctx, q, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("autocomplete failed: %v", err)), nil
	}

	resp := shared.NewToolResponseFromSession(session, results, "City autocomplete from GIS catalog.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleAutocompleteCounties(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := auth.RequireScope(ctx, req, s.keyRepo, "api")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	q := req.GetString("q", "")
	limit := int(req.GetFloat("limit", 10))
	if limit <= 0 || limit > 25 {
		limit = 10
	}

	results, err := s.autocomplete.Counties(ctx, q, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("autocomplete failed: %v", err)), nil
	}

	resp := shared.NewToolResponseFromSession(session, results, "County autocomplete from GIS catalog.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}
