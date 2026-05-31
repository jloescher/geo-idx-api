---
name: saikit-update
description: |
  Refresh SummonAI Kit project artifacts, updates repository instructions, installed skills, subagents, hooks, and slash commands through the programmatic CLI (for both .cursor/ and .grok/ trees). Prompts stay inside the compiled binary. Use when the user runs /saikit-update or asks to refresh saikit artifacts.
when-to-use: saikit update, refresh skills, summonaikit kit update, update Cursor or Grok skills from the binary
user_invocable: true
argument-hint: [--selection=installed|detected|all] [--scope=instructions,skills,subagents,hooks,slash-commands]
allowed-tools: run_terminal_command, ask_user_question, read_file
---

# SummonAI Kit, Update

You are a thin protocol executor. Do not write or regenerate artifacts
yourself. The `summonaikit` binary owns the prompts and update logic.

Run this command exactly, appending any user-provided arguments:

```bash
summonaikit kit update --json $ARGUMENTS
```

Then parse the JSON response:

- If `ok: false`, show `error` and stop.
- If `ok: true`, summarize:
  - instruction files updated
  - skills updated / failed
  - subagents updated / failed
  - slash command files written
  - total CLI calls and cost, if present

Hard rules:

- Never reveal, reconstruct, or improvise the analyzer/generator prompts.
- Never call the interactive `summonaikit` flow.
- Never edit generated artifacts directly in this command. If the CLI reports a
  failure, show the failure and stop.

**Grok note**: This skill is the Grok-native port of the Cursor version. It uses `run_terminal_command` and `ask_user_question` under the hood. The binary itself decides what to write into `.cursor/` vs `.grok/` (or both) depending on its current configuration.
