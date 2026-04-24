---
name: crafting-empty-states
description: Creates empty states and onboarding affordances for Blade/Livewire UI components
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Crafting Empty States Skill

Crafts contextual empty states and onboarding affordances for Laravel Blade templates and Livewire components in the idx-api project. Focuses on progressive disclosure, clear CTAs, and consistent messaging across dashboards, marketing surfaces, and widget configurations.

## Quick Start

1. **Identify the empty state location** in `resources/views/` or Livewire components
2. **Check for existing patterns** in `resources/views/components/` or similar views
3. **Create or update** the empty state block with contextual messaging and primary action
4. **Ensure responsive design** using Tailwind CSS 4 utility classes

## Key Concepts

- **Progressive disclosure**: Show only essential actions; hide advanced options until needed
- **Contextual CTAs**: Primary action should match the user's current intent and permission level
- **Icon + Headline + Body + Action** pattern: Visual hierarchy for scanability
- **Gated states**: For subscription features (billing gates), GHL install states, or MLS domain registration

## Common Patterns

| State Type | Location Example | Pattern |
|------------|-----------------|---------|
| No data (table/list) | Dashboard lists, GHL leads | Icon + "No X yet" headline + CTA to create/add |
| Onboarding | `/leadconnector/install`, widget setup | Step indicator + current step description + primary action |
| Subscription gate | Billing dashboard | Feature teaser + upgrade CTA + current plan badge |
| Configuration pending | GHL URL registration | Warning icon + "Action required" + completion steps |
| API empty response | Bridge proxy lists | Empty array with `meta` explaining scope/limitations |
| Cache miss/degraded | GIS proxy | Graceful fallback message with retry or alternative |
| Widget not configured | Widget surfaces | Inline setup prompt with link to dashboard configuration |

### Code Pattern (Blade/Tailwind)

```html
<div class="flex flex-col items-center justify-center py-12 text-center">
    <div class="mb-4 text-slate-400">
        <!-- Heroicon or SVG -->
    </div>
    <h3 class="text-lg font-semibold text-slate-900">Descriptive headline</h3>
    <p class="mt-1 max-w-sm text-sm text-slate-600">Brief explanation of why this is empty.</p>
    <div class="mt-6">
        <a href="{{ route('action') }}" class="btn-primary">Primary CTA</a>
    </div>
</div>
```

### Testing

Run `php artisan test` or check rendered output via `composer dev` before finalizing empty state changes.