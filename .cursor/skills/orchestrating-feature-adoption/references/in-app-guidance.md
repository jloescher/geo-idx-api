# In-App Guidance Reference

## Contents
- Dashboard Guidance Surfaces
- Badge Status System
- Copy-to-Clipboard Pattern
- API Documentation as Guidance
- Anti-Patterns

## Dashboard Guidance Surfaces

The dashboard is the primary guidance surface. It is server-rendered HTML using `web.Page()` and `web.LoginPage()` wrappers from `internal/web/layout.go`.

### Template Structure

```go
// existing pattern — internal/web/layout.go
func Page(title, body string) string {
    // Full page with:
    // - CSS link to /static/css/app.css
    // - JS link to /static/js/app.js
    // - Header with "Quantyra IDX" branding
    // - Navigation (dashboard, login/logout)
    // - HTML-escaped title
    // - Body content passed as parameter
}
```

Guidance lives inline in handler-generated HTML — not in separate template files.

### Where to Add Guidance

| Location | When to use | Example |
|----------|------------|---------|
| Domain card | Explain verification step | "Add a TXT record to your DNS to verify ownership" |
| Token card | Explain staging vs production | "Use staging tokens for development, production for live sites" |
| Empty state card | First action prompt | "Add your first domain to get started" |
| Verification success | Next step nudge | "Domain verified — your production token is ready below" |

## Badge Status System

Badges provide visual status guidance using CSS classes from `app.css`:

```css
/* existing — internal/web/static/css/app.css */
.badge-verified { background: rgba(16,185,129,.15); color: var(--success); }
.badge-pending  { background: rgba(245,158,11,.15); color: var(--warning); }
```

### Badge Usage Pattern

```go
// existing pattern — dashboard HTML generation
fmt.Sprintf(`<span class="badge badge-verified">Verified</span>`)
fmt.Sprintf(`<span class="badge badge-pending">Pending</span>`)
```

When adding new status states, follow the existing color pattern:
- Use `rgba()` with 0.15 opacity background
- Reference CSS variables (`--success`, `--warning`, `--danger`)
- Keep text uppercase and short

## Copy-to-Clipboard Pattern

The `data-copy` attribute pattern from `app.js` provides inline guidance for token handling:

```javascript
// existing — internal/web/static/js/app.js
document.querySelectorAll('[data-copy]').forEach((el) => {
    el.addEventListener('click', () => {
        const target = document.querySelector(el.getAttribute('data-copy'));
        if (target) { navigator.clipboard.writeText(target.textContent.trim()); }
    });
});
```

Usage in dashboard HTML:

```html
<!-- existing pattern -->
<code id="token-value">idx_xxxx...</code>
<button data-copy="#token-value">Copy</button>
```

## API Documentation as Guidance

The OpenAPI spec serves as developer guidance:

- `GET /openapi.json` — machine-readable spec
- `GET /swagger` — interactive API explorer
- Source: `docs/yaak-api-collection.json`

For in-dashboard guidance, link to the Swagger UI rather than embedding API docs.

## Anti-Patterns

### WARNING: Tooltip/Modal Libraries for Dashboard

**The Problem:** Adding a tooltip or modal library (Bootstrap, Popper, etc.) for 2-3 callouts.

**Why This Breaks:** The dashboard has minimal JavaScript (~10 lines in `app.js`). Adding a UI framework for a few text callouts bloats the page and fights the server-rendered architecture.

**The Fix:** Use inline HTML with the existing `.card` and `.badge` system. Add microcopy directly in handler-generated HTML. The `data-copy` pattern shows that simple attribute-driven JS is sufficient.

### WARNING: Guided Tour Overlays

**The Problem:**

```javascript
// BAD — adding a tour library
import Tour from 'shepherd.js'
const tour = new Tour({ steps: [...] })
```

**Why This Breaks:** Server-rendered dashboard. No build pipeline for JS imports. Tour state conflicts with session state.

**The Fix:** Progressive disclosure through card ordering. Show the next actionable card first. Hide completed steps or collapse them.

### WARNING: Marketing Copy in Dashboard

**The Problem:** Using dashboard cards for feature announcements or marketing language.

**Why This Breaks:** Dashboard users are already authenticated customers. They need status and next actions, not feature pitches.

**The Fix:** Dashboard cards should answer: "What is the current state?" and "What do I do next?" Save marketing for the landing page (`/` route handled by `mktH.Home`).

See the **frontend-design** skill for CSS variables, card layout, and badge styling.
See the **ux** skill for dashboard density and state flow patterns.