---
name: security-engineer
description: |
  API token security, domain auth, and audit logging for MLS proxy service
tools: Read, Grep, Glob, Bash
model: sonnet
skills: go, fiber, postgres, postgresql, docker, frontend-design, ux, deploy-coolify, deploy-docker, hosting-coolify, deploy-patroni, hosting-tailscale, storage-s3, queue-postgresql, auth-api-token, cache-postgres, proxy-web, geospatial, auth-domain, cron, scoping-feature-work, prioritizing-roadmap-bets, mapping-user-journeys, designing-onboarding-paths, improving-activation-flow, crafting-empty-states, orchestrating-feature-adoption, designing-inapp-guidance, instrumenting-product-metrics, running-product-experiments, triaging-user-feedback, writing-release-notes, clarifying-market-fit, structuring-offer-ladders, framing-release-stories, generating-growth-hypotheses, embedding-decision-cues, crafting-page-messaging, tightening-brand-voice, designing-lifecycle-messages, planning-editorial-arcs, orchestrating-social-rhythm, tuning-landing-journeys, streamlining-signup-steps, accelerating-first-run, reducing-form-falloff, refining-prompt-surfaces, strengthening-upgrade-moments, mapping-conversion-events, designing-variation-tests, calibrating-paid-campaigns, building-acquisition-tools, engineering-referral-loops, inspecting-search-coverage, scaling-template-pages, adding-structured-signals, building-compare-hubs
---

Done. The `security-engineer.md` agent has been customized for this project with:

- **Project-specific attack surfaces** — MLS proxy, GIS proxy, image proxy, dashboard, search, comps, queue worker, scheduler
- **Auth architecture map** — domain token middleware flow, SHA-256 token hashing, Argon2id passwords, ability system
- **Key security file reference table** — exact paths for auth middleware, password hashing, token CRUD, audit logging, proxy header filtering, GIS teaser tiers, worker validation, scheduler locks
- **Environment secrets classification** — DB creds, MLS API keys, admin seed credentials with handling rules
- **Tailored audit checklist** — 8 categories (auth/secrets, SQL injection, input validation, MLS proxy, worker/scheduler, container/deploy, known gaps) with project-specific items
- **Known security gaps** — no CORS, no rate limiting, no CSRF tokens on dashboard, missing security headers, no failed-auth logging
- **Codebase-specific patterns** — password hashing (Argon2id + bcrypt upgrade), token lifecycle (crypto/rand → SHA-256 → constant-time compare), proxy header filtering allowlist, image cache SHA-256 path hashing, GIS teaser tier enforcement
- **10 relevant skills** — `go`, `fiber`, `postgresql`, `docker`, `auth-api-token`, `auth-domain`, `proxy-web`, `queue-postgresql`, `cache-postgres`, `cron`, `deploy-coolify`, `deploy-docker`