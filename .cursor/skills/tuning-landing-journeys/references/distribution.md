# Distribution Reference

## Contents
- Traffic entry points
- Invite distribution mechanism
- API consumer distribution
- SEO considerations
- Anti-patterns

## Traffic Entry Points

The platform has three distinct audiences with different entry paths:

| Audience | Entry point | Goal |
|----------|-------------|------|
| New users (invitees) | `/invite/:token` link shared by admin | Register → setup → first API call |
| Returning users | `/login` or `/dashboard` directly | Manage domains and tokens |
| API consumers | `POST /api/v1/search` with Bearer token | MLS data access (no page views) |

The landing page (`/`) serves returning users and accidental visitors. It is not a primary acquisition channel — the platform is invite-only.

## Invite Distribution Mechanism

Admin creates invitation in dashboard, receives a one-time link:

```go
// existing — dashboard/handler.go:262-271
func (h *Handler) CreateInvitation(c *fiber.Ctx) error {
    plain, err := h.invitations.Create(c.Context(), uid, c.FormValue("email"))
    link := "/invite/" + plain
    body := `<div class="card"><h1>Invitation created</h1>
<p>Share this link (shown once):</p>
<div class="token-box">` + web.Esc(link) + `</div>
<p><a class="btn btn-primary" href="/dashboard">Back</a></p></div>`
    return c.Type("html").SendString(web.Page("Invitation", body))
}
```

Distribution flow:
1. Admin clicks "Send invitation" (misleading label — no email is sent)
2. Link displayed once in `.token-box`
3. Admin copies link manually and sends via their own channel (Slack, email, etc.)

### WARNING: "Send invitation" implies email delivery

The button says "Send invitation" but nothing is sent. The admin must manually copy and share the link. Rename to "Create invitation link" to set correct expectations.

### DO: Show full invite URL with host

The current link is relative (`/invite/abc...`). The admin needs the full URL to share:

```go
// new code to add — build absolute URL
link := h.cfg.IDXPlatformURL() + "/invite/" + plain
```

## API Consumer Distribution

API consumers (real estate websites) authenticate via Bearer token. They never see the dashboard. Distribution happens through:

1. Admin creates domain in dashboard
2. Admin verifies DNS TXT record
3. Production token auto-generated on verification (`VerifyTXT`)
4. Admin embeds token in their website's backend config
5. Website makes API calls with Bearer token

The API docs at `docs/idx-api-bridge-proxy.md` and `docs/INDEX.md` are the primary distribution surface for API consumers. They live in the repo, not in the web UI.

### DO: Link to API docs from dashboard

```html
<!-- new code to add — in dashboard after API keys card -->
<p class="muted"><a href="/docs/api">API documentation</a> — endpoints, auth, and examples.</p>
```

Note: `/docs/api` route does not exist yet. Could link to the external docs or add a static docs page.

## SEO Considerations

The platform is invite-only and not designed for organic acquisition. SEO considerations are minimal:

- `Page()` sets `<title>` with page name + " · Quantyra IDX"
- No `<meta name="description">` tag
- No structured data
- No sitemap

### DON'T: Invest in SEO for an invite-only platform

The landing page will not rank for competitive real estate terms. Focus instead on making the invite → signup → activation flow fast for referred users.

## Multi-DC Distribution Impact

Production uses Cloudflare geo-routing (see **deploy-coolify** skill). Users in NYC and ATL hit local API instances. The landing page and dashboard are served from the same Fiber app on both:

```
Clients → Cloudflare Geo LB
    ├─ Pool NYC → idx-api-nyc → landing + dashboard + API
    └─ Pool ATL → idx-api-atl → landing + dashboard + API
```

Session data is not shared between DCs. A user who logs in via NYC and is geo-routed to ATL on the next request will be logged out. This is a known limitation.

### DO: Pin session affinity or warn users

Either pin sessions to a DC via Cloudflare session affinity, or add a note on the login page about potential session drops.

## Anti-Patterns

### WARNING: No email delivery for invitations

The invite system generates a link but has no email transport. The admin must manually share the link. If the admin closes the page before copying, the link is lost (token is hashed in DB). Consider adding email delivery via a transactional email service.

### WARNING: Relative invite links

The invite link is relative (`/invite/TOKEN`). If the admin is on `localhost:8000` but the invitee uses `idx.quantyralabs.cc`, the link breaks. Always construct absolute URLs using `IDX_PLATFORM_URL` config.