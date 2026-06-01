package monitor

import (
	"github.com/quantyralabs/idx-api/internal/mcp/auth"
	"github.com/quantyralabs/idx-api/internal/mcp/shared"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// ToolResponse is an alias for the shared MCP tool envelope.
type ToolResponse = shared.ToolResponse

// NewToolResponse creates a response from an MCP key (legacy helper).
func NewToolResponse(mcpKey *repository.MCPKey, data any, notes string) *ToolResponse {
	return shared.NewToolResponseFromKey(mcpKey, data, notes)
}

// NewToolResponseFromSession creates a response with OAuth or key metadata.
func NewToolResponseFromSession(session auth.AuthSession, data any, notes string) *ToolResponse {
	return shared.NewToolResponseFromSession(session, data, notes)
}
