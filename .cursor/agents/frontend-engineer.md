---
description: Embedded dashboard/marketing HTML, internal/web static assets, Tailwind-style CSS in app.css.
tools: Read, Edit, Write, Glob, Grep
skills: go, frontend-design, crafting-empty-states, designing-inapp-guidance
name: frontend-engineer
model: inherit
---

# Frontend engineer — idx-api

## UI stack (Go era)

- **No Livewire/Vite/Blade** in runtime — server-rendered HTML from Go handlers
- **internal/web/layout.go** — `Page()`, `LoginPage()`
- **internal/web/static/css/app.css** — embedded via `//go:embed`
- **internal/web/static/js/app.js** — minimal helpers
- Served at **`/static/*`** (`internal/api/static.go`)

## Handlers

- `internal/handler/marketing` — `/`
- `internal/handler/dashboard` — `/login`, `/dashboard`

## Changes

1. Edit CSS/JS under `internal/web/static/`
2. Restart `make run-api` to pick up embed changes
3. Hard-refresh browser

## Do not

- Add `resources/views` or npm build for dashboard unless explicitly requested

See [docs/INDEX.md](../../docs/INDEX.md).
