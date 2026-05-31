package dashboard

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/quantyralabs/idx-api/internal/web"
)

// SessionAuthMiddleware requires a dashboard session and returns JSON 401 when missing.
func (h *Handler) SessionAuthMiddleware(c *fiber.Ctx) error {
	uid, _, ok := bindSessionUser(c, h.sessions)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	c.Locals("user_id", uid)
	return c.Next()
}

// MonitoringJSON returns the live monitoring snapshot for dashboard JS and admin API.
func (h *Handler) MonitoringJSON(c *fiber.Ctx) error {
	snap, err := h.monitoring.BuildSnapshot(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(snap)
}

// === MCP Key Management (Admin) ===

func (h *Handler) ListMCPKeys(c *fiber.Ctx) error {
	if h.mcpKeys == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "MCP key service not configured")
	}

	uid := c.Locals("user_id").(int64)
	keys, err := h.mcpKeys.ListKeys(c.Context(), uid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"keys": keys})
}

type createMCPKeyRequest struct {
	Name  string   `json:"name"`
	Scopes []string `json:"scopes"`
	Notes string   `json:"notes"`
}

func (h *Handler) CreateMCPKey(c *fiber.Ctx) error {
	if h.mcpKeys == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "MCP key service not configured")
	}

	var req createMCPKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	uid := c.Locals("user_id").(int64)
	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	plaintext, key, err := h.mcpKeys.CreateKey(c.Context(), req.Name, req.Scopes, uid, notes)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Return the plaintext key only once
	return c.JSON(fiber.Map{
		"key":     key,
		"secret":  plaintext, // shown only this one time
		"message": "Store this key securely. It will not be shown again.",
	})
}

func (h *Handler) RevokeMCPKey(c *fiber.Ctx) error {
	if h.mcpKeys == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "MCP key service not configured")
	}

	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid key id")
	}

	uid := c.Locals("user_id").(int64)
	if err := h.mcpKeys.RevokeKey(c.Context(), int64(id), uid); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

// OAuthClientsPage renders the admin page for managing OAuth clients (for Custom MCP connectors).
func (h *Handler) OAuthClientsPage(c *fiber.Ctx) error {
	uid := c.Locals("user_id").(int64)
	clients, _ := h.oauthClients.ListByCreator(c.Context(), uid)

	var b strings.Builder

	b.WriteString(`<section class="card">`)
	b.WriteString(`<h1>OAuth Clients</h1>`)
	b.WriteString(`<p class="text-muted">Registered clients that can start OAuth flows against this MCP server (e.g. Grok Web as a Custom Connector).</p>`)

	// Create form
	b.WriteString(`<div class="setup-section">`)
	b.WriteString(`<h2>Register New Client</h2>`)

	b.WriteString(`<form id="oauth-client-form">`)
	b.WriteString(`<div class="form-grid">`)
	b.WriteString(`<div><label for="client-name">Name</label><input id="client-name" name="name" required placeholder="Grok Web" class="form-control"></div>`)
	b.WriteString(`<div><label for="client-id">Client ID</label><input id="client-id" name="client_id" required placeholder="grok-web" class="form-control"></div>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div style="margin-top: 0.5rem;">`)
	b.WriteString(`<label>Redirect URIs (one per line)</label>`)
	b.WriteString(`<textarea name="redirect_uris" required placeholder="https://grok.x.ai" class="form-control" rows="3"></textarea>`)
	b.WriteString(`<div class="form-hint">Enter full allowed redirect URI prefixes.</div>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div style="margin-top: 0.75rem;">`)
	b.WriteString(`<button type="submit" class="btn btn-primary">Register Client</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`</form>`)
	b.WriteString(`</div>`)

	// List
	b.WriteString(`<div class="setup-section" style="margin-top:1.5rem;">`)
	b.WriteString(`<h2>Registered Clients</h2>`)

	if len(clients) == 0 {
		b.WriteString(`<div class="empty-state"><p>No OAuth clients registered yet.</p><p class="empty-hint">Create one above so Grok Web (or other clients) can use the OAuth flow.</p></div>`)
	} else {
		b.WriteString(`<table class="table" style="width:100%;">`)
		b.WriteString(`<thead><tr><th>Name</th><th>Client ID</th><th>Redirect URIs</th><th>Trusted</th><th>Active Tokens</th><th></th></tr></thead>`)
		b.WriteString(`<tbody>`)
		for _, c := range clients {
			uris := strings.Join(c.RedirectURIs, "<br>")
			trusted := "No"
			if c.IsTrusted {
				trusted = `<span style="color:#166534;">Yes</span>`
			}
			b.WriteString(fmt.Sprintf(
				`<tr data-client-id="%d" data-client-name="%s">
					<td>%s</td>
					<td><code>%s</code></td>
					<td>%s</td>
					<td>%s</td>
					<td>
						<button class="btn btn-sm btn-secondary load-tokens-btn">View Tokens</button>
						<div class="active-tokens" style="margin-top:0.5rem; display:none;"></div>
					</td>
					<td style="text-align:right;">
						<button class="btn btn-sm btn-secondary revoke-client-btn">Revoke Client</button>
					</td>
				</tr>`,
				c.ID, web.Esc(c.Name), web.Esc(c.Name), web.Esc(c.ClientID), uris, trusted,
			))
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</div></section>`)

	// Inline JS (modeled after Admin Tokens / MCP Keys)
	script := `
<script>
(function() {
  const form = document.getElementById('oauth-client-form');

  if (form) {
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const fd = new FormData(form);
      const payload = {
        name: fd.get('name'),
        client_id: fd.get('client_id'),
        redirect_uris: fd.get('redirect_uris').split('\n').map(s => s.trim()).filter(Boolean)
      };

      const res = await fetch('/dashboard/api/oauth/clients', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      if (res.ok) {
        location.reload();
      } else {
        const txt = await res.text();
        alert('Failed to create client: ' + txt);
      }
    });
  }

  document.querySelectorAll('.revoke-btn').forEach(btn => {
    btn.addEventListener('click', async (e) => {
      const row = e.target.closest('tr');
      const id = row.dataset.clientId;
      if (!confirm('Revoke this OAuth client? Existing tokens will stop working.')) return;

      const res = await fetch('/dashboard/api/oauth/clients/' + id, { method: 'DELETE' });
      if (res.ok) {
        row.style.transition = 'opacity .2s';
        row.style.opacity = '0';
        setTimeout(() => row.remove(), 220);
      } else {
        alert('Failed to revoke client.');
      }
    });
  });

  // === Revocation UI for issued access tokens ===
  document.querySelectorAll('.load-tokens-btn').forEach(btn => {
    btn.addEventListener('click', async (e) => {
      const row = e.target.closest('tr');
      const clientId = row.dataset.clientId;
      const container = row.querySelector('.active-tokens');

      if (container.style.display === 'block') {
        container.style.display = 'none';
        btn.textContent = 'View Tokens';
        return;
      }

      btn.textContent = 'Loading...';

      const res = await fetch('/dashboard/api/oauth/access-tokens?client_id=' + encodeURIComponent(clientId));
      const tokens = await res.json();

      let html = '<div style="font-size:0.8rem; margin-top:0.25rem;">';

      if (tokens.length > 0) {
        html += '<div style="margin-bottom:8px;"><button class="btn btn-sm btn-danger revoke-all-tokens-btn" data-client-id="' + clientId + '">Revoke All Tokens for this Client</button></div>';
      }

      if (tokens.length === 0) {
        html += '<em>No active tokens.</em>';
      } else {
        tokens.forEach(function(t) {
          const expires = new Date(t.expires_at).toLocaleString();
          let keyDisplay = (t.granted_key_names && t.granted_key_names.length) ? t.granted_key_names.join(', ') : (t.granted_mcp_key_ids ? t.granted_mcp_key_ids.join(', ') : 'scope-based');

          html += '<div style="margin-bottom:6px; padding:6px 8px; background:#f8fafc; border-radius:4px;">' +
                  '<div><strong>Expires:</strong> ' + expires + '</div>' +
                  '<div><strong>Keys:</strong> <small>' + keyDisplay + '</small></div>' +
                  '<button class="btn btn-sm btn-danger revoke-token-btn" data-token-hash="' + t.token_hash + '" style="margin-top:2px; padding:1px 6px; font-size:0.7rem;">Revoke Token</button>' +
                  '</div>';
        });
      }
      html += '</div>';

      container.innerHTML = html;
      container.style.display = 'block';
      btn.textContent = 'Hide Tokens';

      // Wire revoke buttons for tokens
      container.querySelectorAll('.revoke-token-btn').forEach(revokeBtn => {
        revokeBtn.addEventListener('click', async (ev) => {
          const hash = ev.target.dataset.tokenHash;
          if (!confirm('Revoke this access token?')) return;

          const res = await fetch('/dashboard/api/oauth/access-tokens/' + hash, { method: 'DELETE' });
          if (res.ok) {
            ev.target.parentElement.remove();
          } else {
            alert('Failed to revoke token.');
          }
        });
      });

      // Wire "Revoke All Tokens for this Client" bulk action
      container.querySelectorAll('.revoke-all-tokens-btn').forEach(allBtn => {
        allBtn.addEventListener('click', async (ev) => {
          const cid = ev.target.dataset.clientId;
          if (!confirm('Revoke ALL access tokens for this client? This cannot be undone.')) return;

          const res = await fetch('/dashboard/api/oauth/clients/' + cid + '/tokens', { method: 'DELETE' });
          if (res.ok) {
            container.innerHTML = '<em>All tokens revoked.</em>';
          } else {
            alert('Failed to revoke tokens for client.');
          }
        });
      });
    });
  });
})();
</script>`

	body := b.String() + script
	return c.Type("html").SendString(renderDashboardPage("OAuth Clients", NavOAuthClients, true, body, false))
}

// ActiveOAuthTokensPage shows all currently active OAuth access tokens across all clients (global admin view).
func (h *Handler) ActiveOAuthTokensPage(c *fiber.Ctx) error {
	// Note: For simplicity in this page we'll fetch via API in JS, so the server render can stay light.
	var b strings.Builder

	b.WriteString(`<section class="card">`)
	b.WriteString(`<h1>Active OAuth Access Tokens</h1>`)
	b.WriteString(`<p class="text-muted">All currently valid access tokens issued via the OAuth flow. Use this to audit or revoke access granted to external clients (e.g. Grok Web).</p>`)

	b.WriteString(`<div id="tokens-container" class="setup-section">
		<p>Loading active tokens...</p>
	</div>`)

	script := `
<script>
(async function() {
  const container = document.getElementById('tokens-container');
  const res = await fetch('/dashboard/api/oauth/access-tokens/all');
  const tokens = await res.json();

  if (tokens.length === 0) {
    container.innerHTML = '<div class="empty-state"><p>No active OAuth access tokens.</p></div>';
    return;
  }

  let html = '<table class="table"><thead><tr><th>Client</th><th>User ID</th><th>Granted Keys</th><th>Expires</th><th>Issued</th><th></th></tr></thead><tbody>';
  
  tokens.forEach(t => {
    const expires = new Date(t.expires_at);
    const now = new Date();
    const hoursLeft = Math.round((expires - now) / (1000 * 60 * 60));
    
    let expiryClass = '';
    let expiryText = expires.toLocaleString();
    
    if (hoursLeft < 0) {
      expiryClass = 'style="color:#b91c1c"';
      expiryText += ' (expired)';
    } else if (hoursLeft < 24) {
      expiryClass = 'style="color:#b45309; font-weight:600"';
      expiryText += ' (soon)';
    } else if (hoursLeft < 72) {
      expiryClass = 'style="color:#b45309"';
    }

    let keyDisplay = (t.granted_key_names && t.granted_key_names.length) ? t.granted_key_names.join(', ') : (t.granted_mcp_key_ids ? t.granted_mcp_key_ids.join(', ') : '<em>scope-based</em>');

    let row = '<tr>' +
      '<td><code>' + t.client_id + '</code></td>' +
      '<td>' + t.user_id + '</td>' +
      '<td><small>' + keyDisplay + '</small></td>' +
      '<td ' + expiryClass + '>' + expiryText + '</td>' +
      '<td>' + new Date(t.created_at).toLocaleString() + '</td>' +
      '<td style="text-align:right">' +
        '<button class="btn btn-sm btn-danger revoke-token-btn" data-token-hash="' + t.token_hash + '">Revoke</button>' +
      '</td>' +
    '</tr>';
    html += row;
  });

  html += '</tbody></table>';
  container.innerHTML = html;

  // Wire revoke buttons
  container.querySelectorAll('.revoke-token-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const hash = btn.dataset.tokenHash;
      if (!confirm('Revoke this access token?')) return;
      
      const res = await fetch('/dashboard/api/oauth/access-tokens/' + hash, { method: 'DELETE' });
      if (res.ok) {
        btn.closest('tr').remove();
      } else {
        alert('Failed to revoke token.');
      }
    });
  });
})();
</script>`;

	body := b.String() + script;
	return c.Type("html").SendString(renderDashboardPage("Active OAuth Tokens", NavOAuthTokens, true, body, false));
}
func (h *Handler) AdminTokensPage(c *fiber.Ctx) error {
	uid := c.Locals("user_id").(int64)
	tokens, _ := h.tokens.ListAdminTokensForUser(c.Context(), uid)

	var b strings.Builder

	b.WriteString(`<section class="card">`)
	b.WriteString(`<h1>Admin Tokens</h1>`)
	b.WriteString(`<p class="text-muted">Global admin tokens that are not scoped to any specific MLS domain. Use these for cross-domain operations, the MCP server, or internal tooling.</p>`)

	// === Create Form ===
	b.WriteString(`<div class="setup-section">`)
	b.WriteString(`<h2>Create New Admin Token</h2>`)

	b.WriteString(`<form id="admin-token-form">`)
	b.WriteString(`<div class="form-grid">`)
	b.WriteString(`<div><label for="admin-token-name">Name</label><input id="admin-token-name" name="name" required placeholder="Grok Production MCP" class="form-control"></div>`)
	b.WriteString(`</div>`)

	b.WriteString(`<div class="form-section" style="margin-top: 0.75rem;">`)
	b.WriteString(`<div><strong>Abilities</strong></div>`)
	b.WriteString(`<div class="form-row" style="margin-top: 0.25rem;">`)
	b.WriteString(`<label class="checkbox-label"><input type="checkbox" name="abilities" value="admin" checked> <span>admin</span></label>`)
	b.WriteString(`<label class="checkbox-label"><input type="checkbox" name="abilities" value="idx:full" checked> <span>idx:full</span></label>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="form-hint">These tokens bypass normal domain scoping. Grant with care.</div>`)
	b.WriteString(`</div>`)

	b.WriteString(`<div style="margin-top: 0.75rem;">`)
	b.WriteString(`<button type="submit" class="btn btn-primary">Create Admin Token</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`</form>`)

	// One-time secret reveal (exact same UX pattern as MCP Keys)
	b.WriteString(`<div id="admin-token-secret" class="alert-success" hidden style="margin-top: 1rem;">`)
	b.WriteString(`<div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:0.5rem;">`)
	b.WriteString(`<strong style="color:#166534;">Admin token created — copy now (shown only once)</strong>`)
	b.WriteString(`<button type="button" class="btn btn-sm btn-secondary" id="admin-dismiss-secret">Dismiss</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="token-box" style="background:#f0fdf4; border-color:#86efac;">`)
	b.WriteString(`<code id="admin-secret-value" style="font-size:0.9rem; word-break:break-all;"></code>`)
	b.WriteString(`<button type="button" class="btn btn-secondary btn-sm" id="admin-copy-btn" style="margin-left:0.5rem;">Copy</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)

	// === Active Admin Tokens Table ===
	b.WriteString(`<div class="setup-section" style="margin-top:1.5rem;">`)
	b.WriteString(`<h2>Active Admin Tokens</h2>`)

	if len(tokens) == 0 {
		b.WriteString(`<div class="empty-state"><p>No admin tokens yet.</p><p class="empty-hint">Create one above for global admin access (e.g. for the MCP server or cross-domain tools).</p></div>`)
	} else {
		b.WriteString(`<table class="table" style="width:100%;">`)
		b.WriteString(`<thead><tr><th>Name</th><th>Abilities</th><th>Created</th><th>Last Used</th><th style="width:80px;"></th></tr></thead>`)
		b.WriteString(`<tbody>`)
		for _, t := range tokens {
			// We don't have abilities here in the current List query — we'll improve later.
			// For now show a placeholder.
			abilities := `<code>admin</code>`
			created := "—" // We don't return created_at in the current List query
			lastUsed := "never"
			if !t.NeverUsed {
				lastUsed = t.LastUsed
			}
			b.WriteString(fmt.Sprintf(
				`<tr data-token-id="%d"><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td style="text-align:right;"><button class="btn btn-danger btn-sm revoke-btn">Revoke</button></td></tr>`,
				t.ID, web.Esc(t.Name), abilities, created, lastUsed,
			))
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</div></section>`)

	// Inline JS — almost identical structure to MCP Keys for muscle memory
	script := `
<script>
(function() {
  const form = document.getElementById('admin-token-form');
  const secretBox = document.getElementById('admin-token-secret');
  const secretValueEl = document.getElementById('admin-secret-value');
  const copyBtn = document.getElementById('admin-copy-btn');
  const dismissBtn = document.getElementById('admin-dismiss-secret');

  if (form) {
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const fd = new FormData(form);
      const abilities = fd.getAll('abilities');
      const payload = {
        name: fd.get('name') || 'Admin Token',
        abilities: abilities.length ? abilities : ['admin']
      };

      const res = await fetch('/dashboard/api/admin-tokens', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      if (!res.ok) {
        const txt = await res.text();
        alert('Failed to create admin token: ' + txt);
        return;
      }
      const data = await res.json();

      secretValueEl.textContent = data.token || '(error)';
      secretBox.hidden = false;
      form.reset();

      secretBox.scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
  }

  if (copyBtn && secretValueEl) {
    copyBtn.addEventListener('click', async () => {
      try {
        await navigator.clipboard.writeText(secretValueEl.textContent);
        const original = copyBtn.textContent;
        copyBtn.textContent = 'Copied!';
        setTimeout(() => { copyBtn.textContent = original; }, 1600);
      } catch (e) {
        alert('Copy failed. Please copy manually.');
      }
    });
  }

  if (dismissBtn && secretBox) {
    dismissBtn.addEventListener('click', () => {
      secretBox.hidden = true;
    });
  }

  // Revoke
  document.querySelectorAll('.revoke-btn').forEach(btn => {
    btn.addEventListener('click', async (e) => {
      const row = e.target.closest('tr');
      const id = row.dataset.tokenId;
      if (!confirm('Revoke this admin token? This cannot be undone.')) return;

      const res = await fetch('/dashboard/api/admin-tokens/' + id, { method: 'DELETE' });
      if (res.ok) {
        row.style.transition = 'opacity .2s';
        row.style.opacity = '0';
        setTimeout(() => row.remove(), 220);
      } else {
        alert('Failed to revoke admin token.');
      }
    });
  });
})();
</script>`

	body := b.String() + script
	return c.Type("html").SendString(renderDashboardPage("Admin Tokens", NavAdminTokens, true, body, false))
}

// MCPKeysPage renders the MCP Keys management page for admins (polished UX).
func (h *Handler) MCPKeysPage(c *fiber.Ctx) error {
	uid := c.Locals("user_id").(int64)
	keys, _ := h.mcpKeys.ListKeys(c.Context(), uid)

	var b strings.Builder

	b.WriteString(`<section class="card">`)
	b.WriteString(`<h1>MCP Access Keys</h1>`)
	b.WriteString(`<p class="text-muted">Long-lived keys for AI agents, monitoring tools, and automated systems. The secret value is only shown once upon creation.</p>`)

	// === Create Form ===
	b.WriteString(`<div class="setup-section">`)
	b.WriteString(`<h2>Create New Key</h2>`)

	b.WriteString(`<form id="mcp-key-form">`)
	b.WriteString(`<div class="form-grid">`)
	b.WriteString(`<div><label for="mcp-name">Name</label><input id="mcp-name" name="name" required placeholder="Claude Content Agent" class="form-control"></div>`)
	b.WriteString(`<div><label for="mcp-notes">Notes (optional)</label><input id="mcp-notes" name="notes" placeholder="Used for blog generation + monitoring" class="form-control"></div>`)
	b.WriteString(`</div>`)

	b.WriteString(`<div class="form-section" style="margin-top: 0.75rem;">`)
	b.WriteString(`<div><strong>Scopes</strong></div>`)
	b.WriteString(`<div class="form-row" style="margin-top: 0.25rem;">`)
	b.WriteString(`<label class="checkbox-label"><input type="checkbox" name="scopes" value="monitor" checked> <span>monitor</span></label>`)
	b.WriteString(`<label class="checkbox-label"><input type="checkbox" name="scopes" value="comps" checked> <span>comps</span></label>`)
	b.WriteString(`<label class="checkbox-label"><input type="checkbox" name="scopes" value="content"> <span>content</span></label>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="form-hint">Choose the minimum scopes needed. "content" is for AI content generation tools.</div>`)
	b.WriteString(`</div>`)

	b.WriteString(`<div style="margin-top: 0.75rem;">`)
	b.WriteString(`<button type="submit" class="btn btn-primary">Create Key</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`</form>`)

	// Secret reveal area (shown after successful creation via JS)
	b.WriteString(`<div id="mcp-key-secret" class="alert-success" hidden style="margin-top: 1rem;">`)
	b.WriteString(`<div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:0.5rem;">`)
	b.WriteString(`<strong style="color:#166534;">Key created — copy now (shown only once)</strong>`)
	b.WriteString(`<button type="button" class="btn btn-sm btn-secondary" id="mcp-dismiss-secret">Dismiss</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`<div class="token-box" style="background:#f0fdf4; border-color:#86efac;">`)
	b.WriteString(`<code id="mcp-secret-value" style="font-size:0.9rem; word-break:break-all;"></code>`)
	b.WriteString(`<button type="button" class="btn btn-secondary btn-sm" id="mcp-copy-btn" style="margin-left:0.5rem;">Copy</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)

	// === Existing Keys Table ===
	b.WriteString(`<div class="setup-section" style="margin-top:1.5rem;">`)
	b.WriteString(`<h2>Active Keys</h2>`)

	if len(keys) == 0 {
		b.WriteString(`<div class="empty-state"><p>No MCP keys yet.</p><p class="empty-hint">Create one above to give AI agents and tools access.</p></div>`)
	} else {
		b.WriteString(`<table class="table" style="width:100%;">`)
		b.WriteString(`<thead><tr><th>Name</th><th>Scopes</th><th>Created</th><th>Last Used</th><th style="width:80px;"></th></tr></thead>`)
		b.WriteString(`<tbody>`)
		for _, k := range keys {
			scopes := strings.Join(k.Scopes, ", ")
			created := k.CreatedAt.Format("2006-01-02")
			lastUsed := "never"
			if k.LastUsedAt != nil {
				lastUsed = k.LastUsedAt.Format("2006-01-02 15:04")
			}
			b.WriteString(fmt.Sprintf(
				`<tr data-key-id="%d"><td>%s</td><td><code>%s</code></td><td>%s</td><td>%s</td><td style="text-align:right;"><button class="btn btn-danger btn-sm revoke-btn">Revoke</button></td></tr>`,
				k.ID, web.Esc(k.Name), scopes, created, lastUsed,
			))
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</div></section>`)

	// Self-contained script for excellent UX
	script := `
<script>
(function() {
  const form = document.getElementById('mcp-key-form');
  const secretBox = document.getElementById('mcp-key-secret');
  const secretValueEl = document.getElementById('mcp-secret-value');
  const copyBtn = document.getElementById('mcp-copy-btn');
  const dismissBtn = document.getElementById('mcp-dismiss-secret');

  if (form) {
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const fd = new FormData(form);
      const scopes = fd.getAll('scopes');
      const payload = {
        name: fd.get('name') || '',
        notes: fd.get('notes') || '',
        scopes: scopes.length ? scopes : ['monitor']
      };

      const res = await fetch('/dashboard/api/mcp-keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      if (!res.ok) {
        const txt = await res.text();
        alert('Failed to create key: ' + txt);
        return;
      }
      const data = await res.json();

      secretValueEl.textContent = data.secret || '(error)';
      secretBox.hidden = false;
      form.reset();

      // Scroll to secret
      secretBox.scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
  }

  if (copyBtn && secretValueEl) {
    copyBtn.addEventListener('click', async () => {
      try {
        await navigator.clipboard.writeText(secretValueEl.textContent);
        const original = copyBtn.textContent;
        copyBtn.textContent = 'Copied!';
        setTimeout(() => { copyBtn.textContent = original; }, 1600);
      } catch (e) {
        alert('Copy failed. Please copy manually.');
      }
    });
  }

  if (dismissBtn && secretBox) {
    dismissBtn.addEventListener('click', () => {
      secretBox.hidden = true;
    });
  }

  // Revoke with confirmation
  document.querySelectorAll('.revoke-btn').forEach(btn => {
    btn.addEventListener('click', async (e) => {
      const row = e.target.closest('tr');
      const id = row.dataset.keyId;
      if (!confirm('Revoke this MCP key permanently?')) return;

      const res = await fetch('/dashboard/api/mcp-keys/' + id, { method: 'DELETE' });
      if (res.ok) {
        row.style.transition = 'opacity .2s';
        row.style.opacity = '0';
        setTimeout(() => row.remove(), 220);
      } else {
        alert('Failed to revoke key.');
      }
    });
  });
})();
</script>`

	body := b.String() + script
	return c.Type("html").SendString(renderDashboardPage("MCP Keys", NavMCPKeys, true, body, false))
}

// Add minimal client-side behavior for the MCP Keys page (create + revoke)
func init() {
	// This runs when the package is loaded. Real interactivity is in dashboard.js or inline below.
	// For the rendered page we include a small script via the body.
}

