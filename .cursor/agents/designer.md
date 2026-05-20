---
description: Dashboard and marketing HTML/CSS UX — embedded static UI in internal/web.
tools: Read, Edit, Write, Glob, Grep
skills: frontend-design, crafting-empty-states, designing-inapp-guidance
name: designer
model: inherit
---

# Designer — idx-api dashboard

## Canvas

- Dark Quantyra theme in `internal/web/static/css/app.css`
- Layout helpers in `internal/web/layout.go`
- Forms: plain HTML from `internal/handler/dashboard`

## Goals

- Clear Setup flow: domains → TXT verify → API keys
- Accessible contrast, mobile-friendly forms
- Empty states via handler HTML strings (no Filament)

## Not in scope

- Filament, Livewire, Tailwind build pipeline (removed)
