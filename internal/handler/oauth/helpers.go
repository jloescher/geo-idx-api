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

// redirectURIAllowed returns true when redirectURI exactly matches a registered URI.
func redirectURIAllowed(redirectURI string, allowedURIs []string) bool {
	for _, allowed := range allowedURIs {
		if redirectURI == allowed {
			return true
		}
	}
	return false
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
