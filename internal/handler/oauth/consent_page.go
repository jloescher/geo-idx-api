package oauth

import (
	"fmt"
	"strings"

	"github.com/quantyralabs/idx-api/internal/web"
)

func renderConsentPage(clientName, clientID, redirectURI, codeChallenge, codeChallengeMethod, scope, state string) string {
	clientName = web.Esc(clientName)
	scopeBullets := scopeDescriptionList(scope)

	body := fmt.Sprintf(`
<div class="center-page">
  <div class="card" style="max-width: 32rem;">
    <h1>Authorize %s</h1>
    <p style="color: var(--muted); margin-bottom: 1.25rem;">%s is requesting access to your Quantyra IDX MCP tools.</p>

    <div class="setup-section">
      <h2 style="font-size: 1.05rem; margin-bottom: 0.5rem;">Requested scopes</h2>
      <ul style="margin: 0 0 1rem 1.25rem; padding: 0; color: var(--text);">
        %s
      </ul>
    </div>

    <form method="POST" action="/oauth/authorize">
      <input type="hidden" name="client_id" value="%s">
      <input type="hidden" name="redirect_uri" value="%s">
      <input type="hidden" name="code_challenge" value="%s">
      <input type="hidden" name="code_challenge_method" value="%s">
      <input type="hidden" name="scope" value="%s">
      <input type="hidden" name="state" value="%s">

      <label class="checkbox-row" style="margin: 1.25rem 0;">
        <input type="checkbox" name="consent" value="granted" required>
        <span>I authorize %s to access MCP tools with the scopes above.</span>
      </label>

      <div style="display: flex; gap: 0.75rem; margin-top: 1rem;">
        <button type="submit" class="btn btn-primary">Authorize</button>
        <a href="%s" class="btn btn-secondary">Cancel</a>
      </div>
    </form>

    <p style="margin-top: 1.5rem; font-size: 0.85rem; color: var(--muted);">
      An MCP access key will be created automatically for this connection when you authorize.
    </p>
  </div>
</div>`,
		clientName,
		clientName,
		scopeBullets,
		escapeFormValue(clientID),
		escapeFormValue(redirectURI),
		escapeFormValue(codeChallenge),
		escapeFormValue(codeChallengeMethod),
		escapeFormValue(scope),
		escapeFormValue(state),
		clientName,
		web.Esc(redirectURI),
	)

	return web.Page(fmt.Sprintf("Authorize %s", clientName), body)
}

func scopeDescriptionList(scope string) string {
	scopes, err := ParseAndValidateScopes(scope)
	if err != nil {
		scopes = strings.Fields(defaultScopeString)
	}
	seen := map[string]struct{}{}
	for _, s := range scopes {
		seen[s] = struct{}{}
	}

	type desc struct {
		name string
		text string
	}
	descriptions := []desc{
		{"monitor", "<strong>monitor</strong> — View system monitoring, queues, GIS health, etc."},
		{"comps", "<strong>comps</strong> — Run comparable sales / BPO analysis"},
		{"content", "<strong>content</strong> — Safe content generation queries over listings and GIS"},
		{"api", "<strong>api</strong> — Listing detail, live MLS search, RESO/GIS proxy (via idx-api-web)"},
	}

	var b strings.Builder
	for _, d := range descriptions {
		if _, ok := seen[d.name]; ok {
			b.WriteString("<li>")
			b.WriteString(d.text)
			b.WriteString("</li>")
		}
	}
	return b.String()
}
