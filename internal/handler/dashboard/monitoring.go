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

