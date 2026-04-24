---
name: refining-prompt-surfaces
description: Optimizes banners, modals, and in-app prompts for the Livewire + Blade + Tailwind CSS frontend surfaces of the Quantyra IDX API project.
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Refining Prompt Surfaces Skill

Optimizes banners, modals, and in-app prompts in the Livewire + Blade + Tailwind CSS frontend. Covers sales landing pages, subscriber dashboards, GHL OAuth flows, and widget embeds with consistent styling patterns.

## Quick Start

1. **Locate existing surfaces**: Marketing pages in `resources/views/marketing/`, Livewire components in `app/Livewire/Marketing/`, auth views in `resources/views/auth/`
2. **Check current patterns**: The sales landing uses `SalesLandingPage` Livewire component with Tailwind 4 utility classes
3. **Modal patterns**: Login modal and similar surfaces use Blade components with Alpine/Livewire integration
4. **Run dev server**: `composer dev` (starts Vite HMR + queue + logs in parallel)

## Key Concepts

- **Livewire 3 full-page components**: Marketing surfaces use `app/Livewire/Marketing/` with `#[Layout('layouts.marketing')]` attributes
- **Blade component slots**: Reusable UI patterns in `resources/views/components/` (likely `ui/`, `marketing/` subdirectories)
- **Tailwind CSS 4**: Uses `@tailwindcss/vite` plugin; styles in `resources/css/app.css`
- **Alpine.js interop**: Livewire 3 pairs with Alpine for modal transitions and prompt animations
- **GHL widget contexts**: Embedded widgets at `/widget/*` routes require lightweight, self-contained prompt surfaces

## Common Patterns

**Modal trigger from Livewire**:
```php
// In a Livewire component
public bool $showUpgradeModal = false;

public function openUpgrade(): void
{
    $this->showUpgradeModal = true;
}
```

**Blade modal markup** (Tailwind 4 pattern):
```blade
<div x-data="{ open: @entangle('showUpgradeModal') }" x-show="open" class="relative z-50">
    <div class="fixed inset-0 bg-gray-500/75 transition-opacity"></div>
    <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 sm:items-center sm:p-0">
            {{-- Modal content --}}
        </div>
    </div>
</div>
```

**Banner/prompt positioning**:
- Marketing: Full-width above fold, gradient backgrounds (`bg-gradient-to-br from-indigo-600 to-violet-700`)
- Dashboard: Inline cards with border accents (`border-l-4 border-indigo-500`)
- Widget embeds: Compact, minimal dependencies, `p-4` padding standard

**Conditional display logic** (subscription gating):
```php
// In controllers or Livewire
if ($user?->subscription()?->ended()) {
    return redirect()->route('billing.checkout');
}
```

**Toast/flash patterns**: Session-based with Livewire's `$this->dispatch('notify', ...)` events
```php
$this->dispatch('notify', [
    'type' => 'success',
    'message' => 'Subscription updated.'
]);
```