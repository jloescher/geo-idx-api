package auth

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestAuthSession_HasScope_OAuthTokenScope(t *testing.T) {
	s := AuthSession{
		OAuthToken: &repository.OAuthAccessToken{
			Scope: "monitor comps content",
		},
		oauthScopes: parseScopeSet("monitor comps content"),
	}
	if !s.HasScope("comps") {
		t.Fatal("expected comps scope from oauth token")
	}
	if s.HasScope("api") {
		t.Fatal("did not expect api scope")
	}
}

func TestAuthSession_HasScope_GrantedKeyUnion(t *testing.T) {
	s := AuthSession{
		OAuthToken: &repository.OAuthAccessToken{Scope: "monitor"},
		oauthScopes: parseScopeSet("monitor"),
		grantedScopes: map[string]struct{}{
			"comps": {},
			"api":   {},
		},
	}
	if !s.HasScope("comps") {
		t.Fatal("expected comps from granted key union")
	}
	if !s.HasScope("api") {
		t.Fatal("expected api from granted key union")
	}
}

func TestAuthSession_HasScope_MCPKey(t *testing.T) {
	s := AuthSession{
		MCPKey: &repository.MCPKey{Scopes: []string{"monitor"}},
	}
	if !s.HasScope("monitor") {
		t.Fatal("expected monitor from mcp key")
	}
}

func TestAuthSession_EffectiveScopesList(t *testing.T) {
	s := AuthSession{
		OAuthToken: &repository.OAuthAccessToken{Scope: "monitor"},
		oauthScopes: parseScopeSet("monitor"),
		grantedScopes: map[string]struct{}{
			"api": {},
		},
	}
	scopes := s.EffectiveScopesList()
	if len(scopes) != 2 {
		t.Fatalf("got %v", scopes)
	}
}

func TestParseScopeSet_SkipsGrantedKeys(t *testing.T) {
	set := parseScopeSet("monitor comps granted_keys:1,2")
	if _, ok := set["granted_keys:1,2"]; ok {
		t.Fatal("granted_keys fragment should be skipped")
	}
	if _, ok := set["monitor"]; !ok {
		t.Fatal("expected monitor scope")
	}
}
