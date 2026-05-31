# .grok/hooks — Project Lifecycle Hooks for Grok

JSON hook definitions that run at key moments (SessionStart, PreToolUse, PostToolUse, etc.).

See `~/.grok/docs/user-guide/10-hooks.md` for the full format, events, and trust model.

## Trust

Project hooks require explicit trust the first time:

```
/hooks-trust
```

(or use the `/hooks` modal, `l` to reload after changes).

## Current Hooks

(Examples below — extend as needed for safety, auto-formatting, or surfacing native Grok skills like xlsx/pptx on relevant file events.)

## Example: SessionStart Greeting + Native Skill Reminder

`session-start.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "echo 'Grok session started in idx-api workspace. Project skills from .grok/skills/ (go, fiber, postgresql, deploy-coolify, ux, saikit-*) are highest priority. Native skills (xlsx, pptx, help, best-of-n, review, implement) also available.'"
          }
        ]
      }
    ]
  }
}
```

## Example: PostToolUse Auto-fmt Hint (non-blocking)

`post-edit-go.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "search_replace|write",
        "hooks": [
          {
            "type": "command",
            "command": "if echo \"$GROK_HOOK_EVENT\" | grep -q 'search_replace|write' && git diff --name-only --diff-filter=ACMRT -- '*.go' | head -1 >/dev/null; then echo 'Tip: run `make fmt` or `gofmt -l -w .' after Go edits (from .grok/hooks/).'; fi",
            "timeout": 3
          }
        ]
      }
    ]
  }
}
```

Hooks are fail-open on error. Keep them fast (background with `&` if needed).

For stronger production contract enforcement (the old Cursor skill-instructions-hook.sh), rely on the embedded text in every `.grok/skills/*/SKILL.md` + plan mode + reviewer subagents (`/review`, `implement`, `best-of-n`).

## Adding More

1. Create `*.json` here.
2. `/hooks` (or `Ctrl+L`) → reload (`l`).
3. Test the event.
4. Commit (but never secrets).

Native Grok skills (xlsx etc.) can be surfaced or reacted to by hooks that watch for `.xlsx`/`.pptx` file events.
