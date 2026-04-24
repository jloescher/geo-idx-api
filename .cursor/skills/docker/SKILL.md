---
name: docker
description: |
  Configures Docker multi-stage builds, FrankenPHP, and Compose workflows.
  Use when: implementing or refactoring Docker work, troubleshooting docker, ci cd, deployment, or aligning new changes with the repository's existing conventions
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Docker Skill

This fallback skill keeps Docker work aligned with the conventions already present in this repository. Prefer extending the closest existing implementation over inventing a new abstraction, and verify neighboring states before finishing.

## Quick Start

### Inspect the current implementation

```sh
rg -n "docker|docker|ci-cd|deployment" .
rg --files | rg "docker|docker|ci-cd"
```

### Make the smallest compatible change

- Change the narrowest config surface that solves the problem cleanly.
- Keep environment assumptions, secrets, and deployment behavior explicit.
- Preserve the repository's existing workflow naming, build, and release conventions.

### Verify before finishing

- Validate the change against build, deploy, and rollback expectations.
- Check environment-specific assumptions before merging shared config edits.
- Confirm monitoring or operational visibility is still adequate after the change.

## Key Concepts

| Concept | Why it matters | What to check |
|---------|----------------|---------------|
| Existing patterns | Keeps the repo coherent | Start from the nearest matching implementation before editing |
| Scope control | Prevents abstraction creep | Keep the change in the same layer as surrounding code |
| Verification | Catches regressions early | Recheck adjacent states, edge cases, and integration points |
| References | Speeds up repeat work | Use the linked topic files when the task needs deeper guidance |

## Common Patterns

### Docker

**When:** The task touches docker in Docker work.

- Inspect the nearest existing implementation before introducing a new pattern.
- Reuse naming, file placement, and helper utilities that are already established in this repo.
- Keep the change easy to review and easy to extend without widening scope unnecessarily.

### CI CD

**When:** The task touches ci cd in Docker work.

- Inspect the nearest existing implementation before introducing a new pattern.
- Reuse naming, file placement, and helper utilities that are already established in this repo.
- Keep the change easy to review and easy to extend without widening scope unnecessarily.

### Deployment

**When:** The task touches deployment in Docker work.

- Inspect the nearest existing implementation before introducing a new pattern.
- Reuse naming, file placement, and helper utilities that are already established in this repo.
- Keep the change easy to review and easy to extend without widening scope unnecessarily.

## See Also

- [Docker](references/docker.md)
- [CI CD](references/ci-cd.md)
- [Deployment](references/deployment.md)
- [Monitoring](references/monitoring.md)