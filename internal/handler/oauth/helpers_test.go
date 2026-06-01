package oauth

import (
	"strings"
	"testing"

	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestRedirectURIAllowedExactMatch(t *testing.T) {
	allowed := []string{"https://grok.x.ai", "https://grok.x.ai/callback"}
	if !redirectURIAllowed("https://grok.x.ai", allowed) {
		t.Fatal("expected exact match")
	}
	if redirectURIAllowed("https://grok.x.ai/evil", allowed) {
		t.Fatal("prefix trick must not match")
	}
	if redirectURIAllowed("https://evil.grok.x.ai", allowed) {
		t.Fatal("must not match subdomain prefix")
	}
}

func TestValidatePKCEForAuthorize(t *testing.T) {
	if err := validatePKCEForAuthorize("", ""); err == nil {
		t.Fatal("expected error for missing challenge")
	}
	if err := validatePKCEForAuthorize("challenge", "plain"); err == nil {
		t.Fatal("expected error for plain method")
	}
	if err := validatePKCEForAuthorize("challenge", "S256"); err != nil {
		t.Fatalf("S256 should be allowed: %v", err)
	}
}

func TestBuildAuthorizationRedirectURLPreservesState(t *testing.T) {
	u, err := buildAuthorizationRedirectURL("https://grok.x.ai/callback", "auth-code-123", "grok-state-abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "code=auth-code-123") {
		t.Fatalf("missing code in %q", u)
	}
	if !strings.Contains(u, "state=grok-state-abc") {
		t.Fatalf("missing state in %q", u)
	}
}

func TestConsentFormStateHiddenInputEscaped(t *testing.T) {
	state := `"><script>`
	snippet := `<input type="hidden" name="state" value="` + escapeFormValue(state) + `">`
	if strings.Contains(snippet, "<script>") {
		t.Fatalf("state must be escaped: %s", snippet)
	}
}

func TestValidateTokenExchangeMismatches(t *testing.T) {
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	authCode := &repository.OAuthAuthorizationCode{
		ClientID:            "grok-web",
		RedirectURI:         "https://grok.x.ai",
		CodeChallenge:       &challenge,
		CodeChallengeMethod: strPtr("S256"),
	}

	if err := validateTokenExchange(authCode, "other-client", "", "verifier"); err == nil || !strings.Contains(err.Error(), "client_id") {
		t.Fatalf("expected client_id mismatch, got %v", err)
	}
	if err := validateTokenExchange(authCode, "grok-web", "https://evil.example", "verifier"); err == nil || !strings.Contains(err.Error(), "redirect_uri") {
		t.Fatalf("expected redirect_uri mismatch, got %v", err)
	}
}

func strPtr(s string) *string { return &s }
