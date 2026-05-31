package monitor

import (
	"encoding/json"
	"time"

	"github.com/quantyralabs/idx-api/internal/repository"
)

// ToolResponse is the standard envelope returned by all MCP monitor tools.
// It provides consistent metadata (when it was generated, which key was used,
// and human-friendly notes) around the actual data.
type ToolResponse struct {
	GeneratedAt time.Time `json:"generated_at"`
	KeyName     string    `json:"key_name,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	Data        any       `json:"data"`
}

// NewToolResponse creates a ToolResponse with the current time.
func NewToolResponse(mcpKey *repository.MCPKey, data any, notes string) *ToolResponse {
	resp := &ToolResponse{
		GeneratedAt: time.Now().UTC(),
		Data:        data,
		Notes:       notes,
	}
	if mcpKey != nil {
		resp.KeyName = mcpKey.Name
	}
	return resp
}

// ToJSONResult is a convenience method for MCP tools that want to return
// the response as pretty JSON text.
func (r *ToolResponse) ToJSONResult() (string, error) {
	// We use the standard json package here for simplicity inside the MCP package.
	// In the future we can swap to a faster encoder if needed.
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
