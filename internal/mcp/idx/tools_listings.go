package idx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/quantyralabs/idx-api/internal/mcp/auth"
	"github.com/quantyralabs/idx-api/internal/mcp/shared"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func (s *Server) registerListingTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("get_listing",
		mcp.WithDescription("Fetch a single mirrored listing by mls_listing_id or listing_key. Requires 'api' scope."),
		optionalMCPKey(),
		mcp.WithString("dataset", mcp.Required(), mcp.Description("Dataset slug, e.g. stellar")),
		mcp.WithString("listing_id", mcp.Description("MLS listing id (e.g. TB8459085)")),
		mcp.WithString("listing_key", mcp.Description("RESO listing key (alternative to listing_id)")),
		mcp.WithBoolean("include_expanded", mcp.Description("Include expanded JSONB fields (default false)")),
	), s.handleGetListing)

	mcpServer.AddTool(mcp.NewTool("get_property",
		mcp.WithDescription("Fetch a live RESO property via idx-api-web GET /api/v1/properties/:listingKey. Requires 'api' scope."),
		optionalMCPKey(),
		mcp.WithString("dataset", mcp.Description("MLS dataset slug")),
		mcp.WithString("listing_key", mcp.Required(), mcp.Description("RESO listing key")),
	), s.handleGetProperty)
}

func (s *Server) handleGetListing(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := auth.RequireScope(ctx, req, s.keyRepo, "api")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	dataset := req.GetString("dataset", "")
	if dataset == "" {
		return mcp.NewToolResultError("dataset is required"), nil
	}
	listingID := req.GetString("listing_id", "")
	listingKey := req.GetString("listing_key", "")
	if listingID == "" && listingKey == "" {
		return mcp.NewToolResultError("listing_id or listing_key is required"), nil
	}

	pool, err := s.db.ReadPool(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	q := `SELECT ` + mls.MirrorListingColumns + ` FROM listings WHERE dataset_slug = $1 AND `
	args := []any{dataset}
	if listingID != "" {
		q += `mls_listing_id = $2 LIMIT 1`
		args = append(args, listingID)
	} else {
		q += `listing_key = $2 LIMIT 1`
		args = append(args, listingKey)
	}

	row := pool.QueryRow(ctx, q, args...)
	mirrorRow, err := mls.ScanMirrorListingRow(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return mcp.NewToolResultError("listing not found"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("fetch failed: %v", err)), nil
	}

	includeExpanded := req.GetBool("include_expanded", false)
	var body json.RawMessage
	if includeExpanded {
		body = mls.BuildPublicListingJSON(mirrorRow)
	} else {
		var ok bool
		body, ok = mls.BuildPublicListingJSONForSearch(mirrorRow)
		if !ok {
			return mcp.NewToolResultError("listing is not publicly compliant"), nil
		}
	}

	resp := shared.NewToolResponseFromSession(session, body, "Single listing from mirror with public visibility applied.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}

func (s *Server) handleGetProperty(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	session, err := auth.RequireScope(ctx, req, s.keyRepo, "api")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if s.apiClient == nil || !s.apiClient.Enabled() {
		return mcp.NewToolResultError("MCP API client not configured"), nil
	}

	listingKey := req.GetString("listing_key", "")
	if listingKey == "" {
		return mcp.NewToolResultError("listing_key is required"), nil
	}
	dataset := req.GetString("dataset", "stellar")

	body, _, err := s.apiClient.Get(ctx, "/api/v1/properties/"+url.PathEscape(listingKey), url.Values{"dataset": {dataset}})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get_property failed: %v", err)), nil
	}

	resp := shared.NewToolResponseFromSession(session, json.RawMessage(body), "Live RESO property from idx-api-web.")
	jsonStr, _ := resp.ToJSONResult()
	return mcp.NewToolResultText(jsonStr), nil
}
