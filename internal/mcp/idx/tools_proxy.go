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

func (s *Server) registerProxyTools(mcpServer *server.MCPServer) {
	type spec struct {
		name, desc, path string
	}
	specs := []spec{
		{"list_properties", "GET /api/v1/properties", "/api/v1/properties"},
		{"list_listings", "GET /api/v1/listings", "/api/v1/listings"},
		{"lookup", "GET /api/v1/lookup", "/api/v1/lookup"},
		{"list_agents", "GET /api/v1/agents", "/api/v1/agents"},
		{"get_agent", "GET /api/v1/agents/:agentId", "/api/v1/agents"},
		{"list_offices", "GET /api/v1/offices", "/api/v1/offices"},
		{"list_openhouses", "GET /api/v1/openhouses", "/api/v1/openhouses"},
		{"list_members", "GET /api/v1/members", "/api/v1/members"},
		{"list_reso_offices", "GET /api/v1/reso-offices", "/api/v1/reso-offices"},
		{"list_reso_openhouses", "GET /api/v1/reso-openhouses", "/api/v1/reso-openhouses"},
		{"get_bridge_stats", "GET /api/v1/bridge/stats", "/api/v1/bridge/stats"},
		{"list_pub_parcels", "GET /api/v1/pub/parcels", "/api/v1/pub/parcels"},
		{"list_pub_assessments", "GET /api/v1/pub/assessments", "/api/v1/pub/assessments"},
		{"list_pub_transactions", "GET /api/v1/pub/transactions", "/api/v1/pub/transactions"},
	}
	for _, sp := range specs {
		sp := sp
		mcpServer.AddTool(mcp.NewTool(sp.name,
			mcp.WithDescription(sp.desc+" via idx-api-web. Requires 'api' scope."),
			optionalMCPKey(),
			mcp.WithString("dataset", mcp.Description("MLS dataset slug")),
			mcp.WithString("path_suffix", mcp.Description("Optional path suffix (e.g. id/key)")),
			mcp.WithObject("query", mcp.Description("Optional query parameters")),
		), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return s.handleProxyGET(ctx, req, sp.path)
		})
	}
}

func (s *Server) handleProxyGET(ctx context.Context, req mcp.CallToolRequest, basePath string) (*mcp.CallToolResult, error) {
	session, err := auth.RequireScope(ctx, req, s.keyRepo, "api")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.apiClient == nil || !s.apiClient.Enabled() {
		return mcp.NewToolResultError("MCP API client not configured"), nil
	}

	path := basePath
	if suffix := req.GetString("path_suffix", ""); suffix != "" {
		path += "/" + url.PathEscape(suffix)
	}

	q := url.Values{}
	if ds := req.GetString("dataset", ""); ds != "" {
		q.Set("dataset", ds)
	}
	if raw, ok := req.GetArguments()["query"].(map[string]any); ok {
		for k, v := range raw {
			q.Set(k, fmt.Sprint(v))
		}
	}

	body, _, err := s.apiClient.Get(ctx, path, q)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("proxy GET failed: %v", err)), nil
	}

	resp := shared.NewToolResponseFromSession(session, json.RawMessage(body), "Proxied idx-api-web response.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}
