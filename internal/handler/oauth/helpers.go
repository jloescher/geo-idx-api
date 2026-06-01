package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/quantyralabs/idx-api/internal/repository"
)

// normalizeRedirectURI trims whitespace and removes a trailing slash on the path
// (https://grok.com/api/mcp/auth_callback/ → …/auth_callback).
func normalizeRedirectURI(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}
	if u.Path != "/" && strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	u.Fragment = ""
	return u.String()
}

// redirectURIAllowed returns true when redirectURI matches a registered URI (exact or normalized).
func redirectURIAllowed(redirectURI string, allowedURIs []string) bool {
	candidate := normalizeRedirectURI(redirectURI)
	for _, allowed := range allowedURIs {
		if redirectURI == allowed || candidate == allowed || candidate == normalizeRedirectURI(allowed) {
			return true
		}
	}
	return false
}

func redirectURIRejectedResponse(clientID, redirectURI string) map[string]any {
	return map[string]any{
		"error":                 "invalid_request",
		"error_description":     "redirect_uri is not allowed for this client",
		"client_id":             clientID,
		"received_redirect_uri": redirectURI,
		"normalized_redirect_uri": normalizeRedirectURI(redirectURI),
		"hint":                  "Add received_redirect_uri exactly in Dashboard → MCP Monitoring → Registered Clients, or connect with only the MCP server URL and no manual OAuth fields.",
	}
}

func validatePKCEForAuthorize(codeChallenge, codeChallengeMethod string) error {
	if strings.TrimSpace(codeChallenge) == "" {
		return fmt.Errorf("code_challenge is required")
	}
	method := strings.TrimSpace(codeChallengeMethod)
	if method == "" {
		method = "S256"
	}
	if method != "S256" {
		return fmt.Errorf("code_challenge_method must be S256")
	}
	return nil
}

// buildAuthorizationRedirectURL returns the client redirect with code and state query params.
func buildAuthorizationRedirectURL(redirectURI, code, state string) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func escapeFormValue(s string) string {
	return html.EscapeString(s)
}

// validateTokenExchange checks authorization code binding and PKCE before issuing a token.
func validateTokenExchange(authCode *repository.OAuthAuthorizationCode, clientID, redirectURI, codeVerifier string) error {
	if authCode.ClientID != clientID {
		return fmt.Errorf("client_id mismatch")
	}
	if redirectURI != "" && redirectURI != authCode.RedirectURI {
		return fmt.Errorf("redirect_uri mismatch")
	}
	if authCode.CodeChallenge == nil || strings.TrimSpace(*authCode.CodeChallenge) == "" {
		return fmt.Errorf("PKCE code_challenge missing on authorization code")
	}
	if authCode.CodeChallengeMethod != nil && *authCode.CodeChallengeMethod != "" && *authCode.CodeChallengeMethod != "S256" {
		return fmt.Errorf("unsupported code_challenge_method")
	}
	if strings.TrimSpace(codeVerifier) == "" {
		return fmt.Errorf("code_verifier required")
	}
	hash := sha256.Sum256([]byte(codeVerifier))
	encoded := base64.RawURLEncoding.EncodeToString(hash[:])
	if encoded != *authCode.CodeChallenge {
		return fmt.Errorf("PKCE verification failed")
	}
	return nil
}
