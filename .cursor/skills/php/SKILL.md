---
name: php
description: |
  Enforces PHP 8.3+ patterns, strict typing, and constructor promotion.
  Use when: implementing or refactoring PHP work, troubleshooting patterns, workflows, or aligning new changes with the repository's existing conventions
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# PHP Skill

This fallback skill keeps PHP work aligned with the conventions already present in this repository. Prefer extending the closest existing implementation over inventing a new abstraction, and verify neighboring states before finishing.

## Quick Start

### Inspect the current implementation

```sh
rg -n "php|patterns|workflows" .
rg --files | rg "php|patterns|workflows"
```

### Make the smallest compatible change

- Inspect nearby implementations before writing anything new.
- Prefer the smallest change that fits the repository's current structure and naming.
- Keep the output easy to review, easy to extend, and easy to verify.

### Verify before finishing

- Verify the changed path and the most likely adjacent edge cases.
- Check that naming, layering, and file placement still match nearby code.
- Confirm there is a clear reason for any new abstraction, dependency, or workflow.

## Key Concepts

| Concept | Why it matters | What to check |
|---------|----------------|---------------|
| Existing patterns | Keeps the repo coherent | Start from the nearest matching implementation before editing |
| Scope control | Prevents abstraction creep | Keep the change in the same layer as surrounding code |
| Verification | Catches regressions early | Recheck adjacent states, edge cases, and integration points |
| References | Speeds up repeat work | Use the linked topic files when the task needs deeper guidance |

## Common Patterns

### Patterns

**When:** The task touches patterns in PHP work.

- Inspect the nearest existing implementation before introducing a new pattern.
- Reuse naming, file placement, and helper utilities that are already established in this repo.
- Keep the change easy to review and easy to extend without widening scope unnecessarily.

### Workflows

**When:** The task touches workflows in PHP work.

- Inspect the nearest existing implementation before introducing a new pattern.
- Reuse naming, file placement, and helper utilities that are already established in this repo.
- Keep the change easy to review and easy to extend without widening scope unnecessarily.

## See Also

- [Patterns](references/patterns.md)
- [Workflows](references/workflows.md)