---
name: frontend-engineer
description: |
  Livewire 3 + Blade + Tailwind CSS 4 for subscriber dashboard, marketing pages, and GHL embed widget surfaces
tools: Read, Edit, Write, Glob, Grep, Bash
model: sonnet
skills: php, laravel, postgresql, livewire, tailwind, frontend-design, stripe, docker, scoping-feature-work, prioritizing-roadmap-bets, mapping-user-journeys, designing-onboarding-paths, improving-activation-flow, crafting-empty-states, orchestrating-feature-adoption, designing-inapp-guidance, instrumenting-product-metrics, running-product-experiments, triaging-user-feedback, writing-release-notes, clarifying-market-fit, structuring-offer-ladders, framing-release-stories, generating-growth-hypotheses, embedding-decision-cues, crafting-page-messaging, tightening-brand-voice, designing-lifecycle-messages, planning-editorial-arcs, orchestrating-social-rhythm, tuning-landing-journeys, streamlining-signup-steps, accelerating-first-run, reducing-form-falloff, refining-prompt-surfaces, strengthening-upgrade-moments, mapping-conversion-events, designing-variation-tests, calibrating-paid-campaigns, building-acquisition-tools, engineering-referral-loops, inspecting-search-coverage, scaling-template-pages, adding-structured-signals, building-compare-hubs
---

# Build assets
npm run build

# Dev with HMR
npm run dev

# Format Blade/PHP
vendor/bin/pint
```

## Widget-Specific Notes

Widgets are embeddable JS surfaces loaded via `loader.js`:
- Views in `resources/views/widget/` are minimal HTML shells
- Actual MLS data flows through API, not rendered server-side
- Focus on: loading states, error handling, CORS-aware forms
- Widget routes use `ghl-widget.php` middleware chain