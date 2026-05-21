# Fiber Auth Reference

## When To Use

Use this reference when the task touches auth while working on Fiber code in this repository.

## What To Inspect

- Keep transport concerns thin and push reusable business logic into the layers this repo already uses.
- Match the current validation, auth, and error-shaping patterns before introducing new helpers.
- Preserve the existing request and response contract unless the task explicitly requires a change.
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

- [ ] Recheck validation, auth, and error paths alongside the happy path.
- [ ] Confirm downstream callers still receive the shape and status semantics they expect.
- [ ] Audit logging, retries, or persistence side effects if the change touches them.