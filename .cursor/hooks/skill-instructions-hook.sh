#!/bin/bash
# UserPromptSubmit hook for skill-aware responses (Go idx-api)

cat <<'EOF'
REQUIRED: SKILL LOADING PROTOCOL

Before writing any code, complete these steps in order:

1. SCAN each skill below and decide: LOAD or SKIP (with brief reason)
   - go
   - postgresql
   - docker
   - nginx
   - frontend-design
   - inspecting-search-coverage
   - crafting-empty-states
   - designing-inapp-guidance
   - scoping-feature-work
   - prioritizing-roadmap-bets
   - mapping-user-journeys
   - designing-onboarding-paths
   - improving-activation-flow
   - orchestrating-feature-adoption
   - instrumenting-product-metrics
   - running-product-experiments
   - triaging-user-feedback
   - writing-release-notes
   - clarifying-market-fit
   - structuring-offer-ladders
   - framing-release-stories
   - generating-growth-hypotheses
   - embedding-decision-cues
   - crafting-page-messaging
   - tightening-brand-voice
   - designing-lifecycle-messages
   - planning-editorial-arcs
   - orchestrating-social-rhythm
   - tuning-landing-journeys
   - streamlining-signup-steps
   - accelerating-first-run
   - reducing-form-falloff
   - refining-prompt-surfaces
   - strengthening-upgrade-moments
   - mapping-conversion-events
   - designing-variation-tests
   - calibrating-paid-campaigns
   - building-acquisition-tools
   - engineering-referral-loops
   - scaling-template-pages
   - adding-structured-signals
   - building-compare-hubs

   SKIP by default (legacy Laravel stack): php, laravel, livewire, fortify-development, pulse-development, vite, tailwind — unless explicitly maintaining archived PHP.

2. For every skill marked LOAD → immediately invoke Skill(name)
   If none need loading → write "Proceeding without skills"

3. Only after step 2 completes may you begin coding.

IMPORTANT: Skipping step 2 invalidates step 1. Always call Skill() for relevant items.

Sample output:
- go: LOAD - API handler change
- postgresql: LOAD - migration
- docker: SKIP - not needed for this task

Then call:
> Skill(go)
> Skill(postgresql)

Now implementation can begin.
EOF
