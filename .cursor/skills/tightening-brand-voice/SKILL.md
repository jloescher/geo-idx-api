---
name: tightening-brand-voice
description: Refines copy across documentation, marketing, and UI for clarity, tone, and consistency with Quantyra's professional real estate tech voice.
allowed-tools: [Read, Edit, Write, Glob, Grep, Bash]
---

# Tightening Brand Voice Skill

This skill tightens prose across the Quantyra GeoIDX codebase—documentation, marketing pages, API references, and UI strings—to ensure a consistent, professional voice that balances technical precision with approachable clarity for real estate technology audiences.

## Quick Start

1. **Identify copy targets**: Marketing pages (`resources/views/**/*.blade.php`), documentation (`docs/*.md`), and UI components (`app/Livewire/**/*.php`)
2. **Check for inconsistencies**: Look for mixed terminology (e.g., "leadconnector" vs "LeadConnector", "MLS" vs "mls"), verbose phrasing, or unclear CTAs
3. **Apply constraints**: Keep technical terms exact (OAuth scopes, API endpoints, subscription tier names), remove fluff, ensure active voice in CTAs
4. **Verify in context**: Read surrounding code to ensure tightened copy still fits variable interpolation and conditional logic

## Key Concepts

- **Precision over puffery**: Real estate tech buyers distrust marketing speak. Prefer "Subscribe to unlock full MLS access" over "Supercharge your real estate journey"
- **Terminology lock**: Standard terms—Bridge (not "the Bridge API"), GHL/GoHighLevel (not "HighLevel"), leadconnector (lowercase in URLs, code), Stellar MLS (not "Stellar")
- **Action clarity**: CTAs should state the outcome—"Connect GoHighLevel account" beats "Get started"
- **Teaser vs full**: Copy distinguishes between gated (teaser) and unlocked (full) features; respect this revenue mechanic in UI text
- **Code-adjacent copy**: Blade templates and Livewire components often interpolate variables—ensure tightened strings still work with `{{ }}` and conditional wrappers

## Common Patterns

- **Docs headings**: Sentence case, no trailing periods, front-loaded keywords ("OAuth token refresh" not "How to refresh your OAuth token")
- **Error messages**: Start with what happened, then remediation: "Domain not registered. Add this domain in your dashboard."
- **Marketing tiers**: Preserve exact plan names (Pro, Smart, Ultra, Mega) and price formatting ($39/mo, $374/yr)
- **API descriptions**: Lead with the resource, note the method: "Returns paginated leads for the authenticated location"
- **Comments**: Keep `Revenue impact:` annotations intact; tighten explanatory text around them without removing the marker itself