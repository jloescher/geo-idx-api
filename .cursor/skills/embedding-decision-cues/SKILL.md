---
name: embedding-decision-cues
description: Applies behavioral cues that improve conversion decisions across pricing pages, subscription tiers, and gated content flows.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Embedding Decision Cues Skill

This skill guides the application of behavioral economics and conversion-focused design patterns to increase subscription conversions, upgrade rates, and lead capture effectiveness within the tiered SaaS billing system (Pro/Smart/Ultra/Mega) and teaser-gated MLS content flows.

## Quick Start

1. **Identify the conversion goal** — subscription signup, plan upgrade, or lead registration
2. **Locate the relevant surface** — `SalesLandingPage` Livewire component, Blade pricing templates, or `BridgeTeaser` gated content
3. **Apply the appropriate cue pattern** from the library below
4. **Ensure tracking** — verify conversion events log to `bridge_proxy_audit_logs` or GHL audit systems

## Key Concepts

| Cue Type | Application in Codebase | Implementation Target |
|----------|------------------------|----------------------|
| **Price Anchoring** | Establish value reference before showing actual price | `SalesLandingPage`, `SubscriptionCheckoutController` |
| **Decoy Effect** | Insert asymmetric middle tier to drive Smart plan selection | `SubscriptionCatalog` plan definitions |
| **Loss Aversion** | Frame trial expiration as losing access | Trial countdown UI, billing email templates |
| **Social Proof** | Display active user counts or recent signups | Marketing views, widget loader scripts |
| **Scarcity/Urgency** | Limited-time offers or capacity indicators | Promotional pricing (domain limits: Pro=3, Smart=5) |
| **Teaser Gating** | Show partial value to motivate full conversion | `BridgeTeaser` (3-item cap), `DomainOrTokenAuth` middleware |
| **Progressive Disclosure** | Reveal features gradually | Plan comparison tables, onboarding flows |

## Common Patterns

### 1. Recommended Plan Highlighting
In `SubscriptionCatalog` or plan display components, visually distinguish the target conversion tier:

- Apply distinct styling for the "Smart" plan ($79/mo target middle tier)
- Add "Most Popular" or "Recommended" badges via `SubscriptionCatalog::getBadgeForPlan()`
- Use elevation/border changes to draw visual attention

### 2. Teaser Value Demonstration
In `BridgeTeaser` and gated content flows:

- Maintain 3-listing cap for non-`idx:full` access (current behavior)
- Display specific upgrade CTA: "Unlock 247 more listings in this area"
- Inline upgrade prompts within search results

### 3. Trial Framing
In `SubscriptionCheckoutController` and billing flows:

- Lead with "Start your 14-day free trial" not payment collection
- Highlight annual savings prominently (20% off: $374 vs $468)
- Default billing interval toggle to annual with monthly secondary

### 4. Social Proof Integration
In `SalesLandingPage` and `/widget/loader.js`:

- Display agent counts when available via `ghl_installed_locations`
- Show recency indicators ("Just listed in Tampa")
- Include Stellar MLS compliance badges for trust

### 5. Urgency Triggers
In promotional contexts:

- Countdown timers for limited pricing windows
- "Only N spots left at this price" for metered tiers (Ultra: 2M calls/mo)
- Expiration date prominence on subscription status

### 6. Simplified Choice Architecture
In plan selection UI:

- Surface 3 primary plans (hide Mega behind "Enterprise" expansion)
- Consistent CTAs: "Start Free Trial" across tiers
- Pre-select Smart plan in UI state

## Implementation Guidelines

- **Respect teaser boundaries**: Extend `BridgeTeaser` patterns (3-item cap) rather than replace; full access requires `idx:full` Sanctum ability
- **Maintain MLS compliance**: All cues align with Stellar MLS data policies—no false scarcity on listing data itself
- **Preserve audit trails**: Log conversion events to `bridge_proxy_audit_logs` or `ghl_audit_logs` for attribution
- **Widget parity**: Ensure `/widget/*` routes reflect consistent decision cues as main platform
- **Mobile-first**: Test cues in widget embeds and responsive dashboard views

## Anti-Patterns to Avoid

- Dark patterns (forced continuity without clear notice)
- Fake scarcity on unlimited resources
- Disruptive interstitials harming SEO
- Hidden pricing until final checkout step
- Teaser manipulation that violates `BridgeProxyAuditLogger` transparency