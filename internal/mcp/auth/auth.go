package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/quantyralabs/idx-api/internal/repository"
)

type contextKey string

const (
	oauthTokenContextKey contextKey = "oauthAccessToken"
	mcpKeyContextKey     contextKey = "validatedMCPKey"
)

// OAuthAccessTokenContextKey stores a validated *repository.OAuthAccessToken.
var OAuthAccessTokenContextKey = oauthTokenContextKey

// MCPKeyContextKey stores a validated *repository.MCPKey.
var MCPKeyContextKey = mcpKeyContextKey

// AuthSession is the resolved identity for an MCP tool call.
type AuthSession struct {
	OAuthToken     *repository.OAuthAccessToken
	MCPKey         *repository.MCPKey
	GrantedKeys    []repository.MCPKey
	ClientID       string
	UserID         int64
	oauthScopes    map[string]struct{}
	grantedScopes  map[string]struct{}
}

// Injector validates HTTP credentials and enriches request context.
type Injector struct {
	KeyRepo   *repository.MCPKeyRepo
	OAuthRepo *repository.OAuthRepo
}

// AuthSessionFromContext reads a partially-built session from context values.
func AuthSessionFromContext(ctx context.Context) AuthSession {
	s := AuthSession{}
	if v := ctx.Value(oauthTokenContextKey); v != nil {
		if t, ok := v.(*repository.OAuthAccessToken); ok {
			s.OAuthToken = t
			s.ClientID = t.ClientID
			s.UserID = t.UserID
			s.oauthScopes = parseScopeSet(t.Scope)
		}
	}
	if v := ctx.Value(mcpKeyContextKey); v != nil {
		if k, ok := v.(*repository.MCPKey); ok {
			s.MCPKey = k
		}
	}
	return s
}

// InjectFromHTTP validates Authorization / query mcp_key and injects auth into context.
func (inj *Injector) InjectFromHTTP(ctx context.Context, r *http.Request) context.Context {
	if inj == nil {
		return ctx
	}

	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if strings.HasPrefix(token, "mcp_") {
			if key := inj.loadMCPKey(ctx, token); key != nil {
				return contextWithMCPKey(ctx, key)
			}
			return ctx
		}
		if inj.OAuthRepo != nil {
			if access := inj.loadOAuthToken(ctx, token); access != nil {
				ctx = contextWithOAuthToken(ctx, access)
				if len(access.GrantedMCPKeyIDs) > 0 && inj.KeyRepo != nil {
					if key, err := inj.KeyRepo.FindByID(ctx, access.GrantedMCPKeyIDs[0]); err == nil && key != nil && key.RevokedAt == nil {
						ctx = contextWithMCPKey(ctx, key)
					}
				}
				return ctx
			}
		}
	}

	if q := r.URL.Query().Get("mcp_key"); strings.HasPrefix(q, "mcp_") {
		if key := inj.loadMCPKey(ctx, q); key != nil {
			return contextWithMCPKey(ctx, key)
		}
	}

	return ctx
}

func (inj *Injector) loadMCPKey(ctx context.Context, raw string) *repository.MCPKey {
	if inj.KeyRepo == nil {
		return nil
	}
	hash := repository.HashMCPKey(raw)
	key, err := inj.KeyRepo.FindValidByHash(ctx, hash)
	if err != nil || key == nil {
		return nil
	}
	go inj.KeyRepo.TouchLastUsed(context.Background(), key.ID)
	return key
}

func (inj *Injector) loadOAuthToken(ctx context.Context, raw string) *repository.OAuthAccessToken {
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])
	access, err := inj.OAuthRepo.FindAccessTokenByHash(ctx, hash)
	if err != nil || access == nil || !time.Now().Before(access.ExpiresAt) {
		return nil
	}
	return access
}

// Resolve completes auth for a tool call (context + optional stdio mcp_key param).
func Resolve(ctx context.Context, req mcp.CallToolRequest, keyRepo *repository.MCPKeyRepo) (AuthSession, error) {
	s := AuthSessionFromContext(ctx)

	if s.OAuthToken != nil && keyRepo != nil && len(s.OAuthToken.GrantedMCPKeyIDs) > 0 {
		for _, id := range s.OAuthToken.GrantedMCPKeyIDs {
			key, err := keyRepo.FindByID(ctx, id)
			if err != nil || key == nil || key.RevokedAt != nil {
				continue
			}
			s.GrantedKeys = append(s.GrantedKeys, *key)
		}
		s.grantedScopes = unionKeyScopes(s.GrantedKeys)
	}

	if s.MCPKey == nil {
		keyStr := req.GetString("mcp_key", "")
		if keyStr != "" {
			if keyRepo == nil {
				return AuthSession{}, fmt.Errorf("key validation unavailable")
			}
			hash := repository.HashMCPKey(keyStr)
			key, err := keyRepo.FindValidByHash(ctx, hash)
			if err != nil {
				return AuthSession{}, fmt.Errorf("key validation error: %w", err)
			}
			if key == nil {
				return AuthSession{}, fmt.Errorf("invalid or revoked MCP key")
			}
			s.MCPKey = key
			go keyRepo.TouchLastUsed(context.Background(), key.ID)
		}
	} else if keyRepo != nil {
		go keyRepo.TouchLastUsed(context.Background(), s.MCPKey.ID)
	}

	if s.OAuthToken == nil && s.MCPKey == nil {
		return AuthSession{}, fmt.Errorf("authentication required: connect via OAuth (Authorization header) or provide mcp_key")
	}

	return s, nil
}

// HasScope checks OAuth token scopes and/or MCP key scopes.
func (s AuthSession) HasScope(scope string) bool {
	if scope == "" {
		return false
	}
	if s.MCPKey != nil && s.MCPKey.HasScope(scope) {
		return true
	}
	if _, ok := s.oauthScopes[scope]; ok {
		return true
	}
	if _, ok := s.grantedScopes[scope]; ok {
		return true
	}
	return false
}

// RequireAnyScope resolves auth and verifies at least one of the requested scopes.
func RequireAnyScope(ctx context.Context, req mcp.CallToolRequest, keyRepo *repository.MCPKeyRepo, scopes ...string) (AuthSession, error) {
	s, err := Resolve(ctx, req, keyRepo)
	if err != nil {
		return AuthSession{}, err
	}
	for _, scope := range scopes {
		if s.HasScope(scope) {
			return s, nil
		}
	}
	if len(scopes) == 1 {
		return AuthSession{}, fmt.Errorf("insufficient permissions: '%s' scope is required", scopes[0])
	}
	return AuthSession{}, fmt.Errorf("insufficient permissions: one of %v scopes is required", scopes)
}

// EffectiveScopesList returns the union of OAuth token scopes and granted MCP key scopes.
func (s AuthSession) EffectiveScopesList() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(scope string) {
		if scope == "" {
			return
		}
		if _, ok := seen[scope]; ok {
			return
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	for scope := range s.oauthScopes {
		add(scope)
	}
	for scope := range s.grantedScopes {
		add(scope)
	}
	if s.MCPKey != nil {
		for _, scope := range s.MCPKey.Scopes {
			add(scope)
		}
	}
	return out
}

// RequireScope resolves auth and verifies the requested scope.
func RequireScope(ctx context.Context, req mcp.CallToolRequest, keyRepo *repository.MCPKeyRepo, scope string) (AuthSession, error) {
	s, err := Resolve(ctx, req, keyRepo)
	if err != nil {
		return AuthSession{}, err
	}
	if !s.HasScope(scope) {
		return AuthSession{}, fmt.Errorf("insufficient permissions: '%s' scope is required", scope)
	}
	return s, nil
}

func parseScopeSet(scope string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Fields(scope) {
		if part == "" || strings.HasPrefix(part, "granted_keys:") {
			continue
		}
		out[part] = struct{}{}
	}
	return out
}

func unionKeyScopes(keys []repository.MCPKey) map[string]struct{} {
	out := map[string]struct{}{}
	for _, k := range keys {
		for _, s := range k.Scopes {
			out[s] = struct{}{}
		}
	}
	return out
}

func contextWithMCPKey(ctx context.Context, key *repository.MCPKey) context.Context {
	if key == nil {
		return ctx
	}
	return context.WithValue(ctx, mcpKeyContextKey, key)
}

func contextWithOAuthToken(ctx context.Context, token *repository.OAuthAccessToken) context.Context {
	if token == nil {
		return ctx
	}
	return context.WithValue(ctx, oauthTokenContextKey, token)
}
