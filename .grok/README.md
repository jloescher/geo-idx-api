# .grok/ — Grok Skills, Agents & Hooks for Quantyra IDX API

This directory provides **project-scoped** customizations for the Grok CLI/TUI (xAI), mirroring the rich `.cursor/` setup for Cursor users.

## What Lives Here

- **`.grok/skills/`** — 50+ domain-specific skills (Go/Fiber/Postgres/PostGIS, MLS/RESO proxy, GIS, multi-DC Coolify/Patroni/Tailscale, auth, queue, dashboard, product growth/onboarding flows, etc.). Each skill is a dir with `SKILL.md` + `references/`.
  - Highest priority for this repo (overrides `~/.grok/skills/`).
  - Appear as `/<skill-name>` slash commands.
  - `description` fields drive automatic invocation.

- **`.grok/agents/`** — Custom agents (e.g. `devops-engineer` for Coolify + Patroni + Tailscale ops).

- **`.grok/hooks/`** — Project JSON lifecycle hooks (PreToolUse safety, PostToolUse automation, SessionStart context). Requires `/hooks-trust` the first time.

- Grok also loads the root `AGENTS.md` (shared with Cursor) and any subdirectory `AGENTS.md`.

## Coexistence with Cursor

| Tool   | Skills                  | Agents                | Hooks (lifecycle)      | Slash commands     | Root rules |
|--------|-------------------------|-----------------------|------------------------|--------------------|------------|
| Grok   | `.grok/skills/`        | `.grok/agents/`      | `.grok/hooks/*.json`  | Skills + builtins | `AGENTS.md` |
| Cursor | `.cursor/skills/`      | `.cursor/agents/`    | `.cursor/hooks/` + settings.json | `.cursor/commands/` + skills | `AGENTS.md` |

Both trees are committed. Run `grok inspect` (Grok) or the equivalent Cursor command to see loaded artifacts. The shared `AGENTS.md` (Prompt-Aware Production Contract, UI/UX Quality Contract, Skill Usage Guide) applies to both.

## Native Grok Skills (also available)

Grok brings powerful first-party skills from `~/.grok/skills/` and plugins (lower priority than project ones):
- `xlsx` / `pptx` — full office document creation/editing with bundled Python + LibreOffice scripts (see `~/.grok/skills/xlsx/scripts/`)
- `help`, `create-skill` (aka `/skillify`), `best-of-n`, `review`, `implement`, `canvas`, `statusline`, etc.
- Many more via plugins.

Project hooks in `.grok/hooks/` can surface or react to these native skills.

## Quick Start (Grok)

```bash
grok inspect                 # See all discovered skills (project > user > plugin)
grok inspect --json | jq '.skills[] | {name, path, scope}'

# In a session:
/skills                      # List + inject
/go "implement a new listing search helper"
# or just describe the task — the right skill auto-loads via its description

/plan                        # Enter plan mode (uses the plan agent + project skills)
```

See `~/.grok/docs/user-guide/08-skills.md`, `12-project-rules.md`, `10-hooks.md`, and `16-subagents.md` for full details.

## Maintenance Note

- Cursor changes (via saikit or manual) only touch `.cursor/`.
- Grok changes stay in `.grok/`.
- The two are intentionally kept in sync manually for now (or via future saikit enhancements).
- `AGENTS.md` is the single source of truth for cross-tool contracts and the skill invocation table.

**This setup gives Grok users the same (or better) "at home" experience in the Quantyra IDX API codebase as Cursor users enjoy today.**
