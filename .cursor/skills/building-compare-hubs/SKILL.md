---
name: building-compare-hubs
description: Creates comparison and alternative pages for discovery
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Building Compare Hubs Skill

This skill assists with implementing comparison and alternative discovery pages in the Quantyra IDX API Laravel application. It supports building Livewire components, Blade templates, and supporting controllers that enable users to compare subscription tiers, explore alternative plans, or discover related features through side-by-side layouts and interactive selectors.

## Quick Start

1. Create a Livewire component in `app/Livewire/Marketing/` for interactive comparison state
2. Add the Blade template in `resources/views/` following the Livewire 3 + Tailwind CSS 4 patterns
3. Register routes in `routes/web.php` with the `marketing.*` naming convention
4. Update `app/Billing/SubscriptionCatalog.php` if comparing billing tiers

```bash
# Example component creation pattern
php artisan make:livewire Marketing/PlanComparison --pest
```

## Key Concepts

- **Comparison State**: Livewire components manage selected tiers, highlighted differences, and toggle between monthly/annual billing (see `SalesLandingPage` pattern)
- **Alternative Discovery**: Cross-link related features or plans using the subscription catalog definitions in `config/billing.php`
- **Teaser Gates**: Non-subscribed views show limited data (3 items max per `BridgeTeaser` service) with clear upgrade CTAs
- **Route Organization**: Marketing pages use named routes (`marketing.sales`, `marketing.compare`) for URL consistency
- **Component Reuse**: Leverage existing `SalesLandingPage` billing toggle patterns for interval switching

## Common Patterns

- **Pricing Comparison Grid**: Use Tailwind CSS grid with `grid-cols-1 md:grid-cols-3` for tier cards, highlighting the recommended plan with border/emphasis utilities
- **Alternative Suggestions**: At decision points (e.g., after teaser limits), suggest the next tier up using `SubscriptionCatalog::getPlans()` data
- **Interactive Feature Matrix**: Livewire-powered checkboxes that filter visible features, persisting selections across requests
- **Discovery Embeds**: Widget-compatible comparison snippets using the `/widget/*` route patterns for external sites
- **CTA Routing**: Compare pages link to `billing.checkout` route with `?plan=` query params for direct subscription initiation