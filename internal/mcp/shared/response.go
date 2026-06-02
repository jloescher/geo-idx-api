package shared

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/mcp/auth"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// ToolResponse is the standard envelope returned by all MCP tools.
type ToolResponse struct {
	GeneratedAt     time.Time `json:"generated_at"`
	KeyName         string    `json:"key_name,omitempty"`
	OAuthClientID   string    `json:"oauth_client_id,omitempty"`
	GrantedScopes    []string  `json:"granted_scopes,omitempty"`
	EffectiveScopes  []string  `json:"effective_scopes,omitempty"`
	GrantedKeyNames  []string  `json:"granted_key_names,omitempty"`
	Notes           string    `json:"notes,omitempty"`
	Data            any       `json:"data"`
}

// NewToolResponseFromKey creates a response for direct MCP key auth.
func NewToolResponseFromKey(mcpKey *repository.MCPKey, data any, notes string) *ToolResponse {
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

// NewToolResponseFromSession creates a response with OAuth or key identity metadata.
func NewToolResponseFromSession(session auth.AuthSession, data any, notes string) *ToolResponse {
	resp := &ToolResponse{
		GeneratedAt: time.Now().UTC(),
		Data:        data,
		Notes:       notes,
	}
	if session.MCPKey != nil {
		resp.KeyName = session.MCPKey.Name
	}
	if session.OAuthToken != nil {
		resp.OAuthClientID = session.OAuthToken.ClientID
		for _, scope := range oauthScopesList(session.OAuthToken.Scope) {
			resp.GrantedScopes = append(resp.GrantedScopes, scope)
		}
		for _, k := range session.GrantedKeys {
			resp.GrantedKeyNames = append(resp.GrantedKeyNames, k.Name)
		}
	}
	resp.EffectiveScopes = session.EffectiveScopesList()
	return resp
}

func oauthScopesList(scope string) []string {
	var out []string
	for _, part := range strings.Fields(scope) {
		if part == "" || strings.HasPrefix(part, "granted_keys:") {
			continue
		}
		out = append(out, part)
	}
	return out
}

// ToJSONResult returns compact JSON for MCP tool results.
func (r *ToolResponse) ToJSONResult() (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
