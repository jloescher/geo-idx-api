# Docker CI CD Reference

## When To Use

Use this reference when the task touches ci cd while working on Docker code in this repository.

## What To Inspect

- Change the narrowest config surface that solves the problem cleanly.
- Keep environment assumptions, secrets, and deployment behavior explicit.
- Preserve the repository's existing workflow naming, build, and release conventions.
- Search for nearby implementations before creating a new structure or helper.

## Recommended Workflow

1. Find two or three nearby examples that already solve a similar problem.
2. Decide whether to extend an existing abstraction or keep the change local.
3. Apply the smallest change that keeps behavior predictable and naming consistent.
4. Re-run the most relevant checks for the surface you touched.
5. Update docs, tests, or supporting config only when the behavior truly changed.

## Quality Bar

- Prefer project-native conventions over generic framework advice.
- Keep instructions concise, actionable, and tied to the repository's current structure.
- Avoid new dependencies or patterns unless repetition clearly justifies them.



## Pitfalls

- Mixing incompatible patterns in the same surface or module.
- Rewriting structure that could be extended safely in place.
- Shipping without checking adjacent states, edge cases, or cleanup work.

## Done Checklist

- [ ] Validate the change against build, deploy, and rollback expectations.
- [ ] Check environment-specific assumptions before merging shared config edits.
- [ ] Confirm monitoring or operational visibility is still adequate after the change.