# In-App Guidance Reference

How release notes reference dashboard and in-app surfaces for Quantyra IDX API.

## Dashboard Surfaces in Release Notes

The Quantyra dashboard (`/dashboard`) is an invite-only surface for domain management and API key creation. Release notes that touch dashboard flows should reference specific panels and actions.

### Dashboard Components

| Component | Location | Purpose |
|-----------|----------|---------|
| Setup panel | Dashboard home | Domain TXT verification, auto-issued PAT |
| API Keys panel | `/dashboard/api-tokens` | Custom named tokens (`idx:full`) |
| Staging key | `/dashboard/api-tokens/staging` | One-click staging Bearer |
| Domain management | Dashboard settings | Domain binding, MLS feed allowlists |

### DO: Reference dashboard actions customers take

```markdown
## v2.1.0

### Dashboard
- **API Keys**: New staging token endpoint — generate a staging Bearer from the API Keys panel without CLI access
- **Domain Setup**: TXT verification now checks propagation automatically (no manual refresh needed)
```

### DON'T: Describe internal handler changes

```markdown
<!-- BAD — customers don't know or care about handler internals -->
### Dashboard
- `dashboard.NewHandler` now includes staging token route registration
- Domain verification middleware refactored to separate TXT check from domain binding
```

### In-App Guidance for New Features

When a release adds new API capability that requires dashboard configuration:

1. Note if customers need to visit the dashboard to enable or configure
2. Specify which panel (Setup, API Keys, Settings)
3. State whether existing tokens gain access automatically or need re-issue

Example:

```markdown
### GIS — Parcel Overlay
- New `GET /api/v1/gis` endpoint for public government parcel GeoJSON
- Accessible with existing `idx:full` tokens — no dashboard action needed
- `idx:access` tokens receive teaser-limited responses (coordinate rounding, feature cap)
```

### API Token Scope Changes

Token scope changes (`idx:access` vs `idx:full`) are breaking for some users. Always:

- State which scope level is affected
- Note if teaser limits apply (GIS uses `GIS_TEASER_MAX_FEATURES`, `GIS_TEASER_COORD_DECIMALS`)
- Link to `docs/api.md` auth section for scope reference

### WARNING: Undocumented Dashboard Changes

If a release modifies dashboard behavior but no API endpoint changed, the release note is still needed — dashboard users are a distinct audience from API consumers. Never omit dashboard changes because "no API route changed."

## Related References

- See the **auth-api-token** skill for token scope documentation
- See the **frontend-design** skill for dashboard UI changes
- See the **ux** skill for dashboard UX patterns