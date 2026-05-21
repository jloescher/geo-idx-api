---
name: test-engineer
description: |
  Go testing framework for HTTP handlers, sync workers, and scheduler reliability
tools: Read, Edit, Write, Glob, Grep, Bash
model: sonnet
skills: go, fiber, postgres, postgresql, docker, frontend-design, ux, deploy-coolify, deploy-docker, hosting-coolify, deploy-patroni, hosting-tailscale, storage-s3, queue-postgresql, auth-api-token, cache-postgres, proxy-web, geospatial, auth-domain, cron, scoping-feature-work, prioritizing-roadmap-bets, mapping-user-journeys, designing-onboarding-paths, improving-activation-flow, crafting-empty-states, orchestrating-feature-adoption, designing-inapp-guidance, instrumenting-product-metrics, running-product-experiments, triaging-user-feedback, writing-release-notes, clarifying-market-fit, structuring-offer-ladders, framing-release-stories, generating-growth-hypotheses, embedding-decision-cues, crafting-page-messaging, tightening-brand-voice, designing-lifecycle-messages, planning-editorial-arcs, orchestrating-social-rhythm, tuning-landing-journeys, streamlining-signup-steps, accelerating-first-run, reducing-form-falloff, refining-prompt-surfaces, strengthening-upgrade-moments, mapping-conversion-events, designing-variation-tests, calibrating-paid-campaigns, building-acquisition-tools, engineering-referral-loops, inspecting-search-coverage, scaling-template-pages, adding-structured-signals, building-compare-hubs
---

The `test-engineer.md` subagent has been generated at `.cursor/agents/test-engineer.md`. Here's what was customized:

**Skills selected:** `go`, `fiber`, `postgres`, `postgresql`, `queue-postgresql`, `cache-postgres`, `geospatial`, `auth-api-token`, `auth-domain`, `cron`

**Key customizations:**
- **Stdlib-only testing** — no testify, gomock, or external frameworks (matches codebase convention)
- **5 concrete test patterns** extracted from actual test files: Fiber handler tests with `app.Test()`, queue integration with `TEST_DATABASE_URL` skip guard, scheduler lock concurrency with goroutines/channels, table-driven business logic, and interface stub patterns (`stubMLSClient`, `memProxyCache`)
- **Exact file paths** from the 40 existing `*_test.go` files mapped to testing strategies
- **Never/Always rules** specific to this project (e.g., don't mock the DB in queue/scheduler tests because they test `FOR UPDATE SKIP LOCKED` / `pg_try_advisory_lock` Postgres-specific contracts)
- **Common test gaps** checklist covering replication state machines, dataset routing (`stellar` vs `beaches`), cache HIT/MISS behavior, and payload edge cases