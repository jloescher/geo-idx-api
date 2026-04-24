---
name: designing-inapp-guidance
description: Builds tooltips, tours, and contextual guidance for Laravel + Livewire + Tailwind applications
allowed-tools: Read, Edit, Write, Glob, Grep, Bash
---

# Designing Inapp Guidance Skill

Designs and implements in-app guidance systems including tooltips, onboarding tours, and contextual help overlays using Blade components, Alpine.js, and Tailwind CSS.

## Quick Start

1. Check existing UI components in `resources/views/components/` or `resources/views/livewire/`
2. Create guidance components: `php artisan make:component Tooltip` or `php artisan make:livewire OnboardingTour`
3. Use Alpine.js for state management (`x-data`, `x-show`, `x-transition`)
4. Style with Tailwind utility classes

## Key Concepts

| Pattern | Implementation | Best For |
|---------|---------------|----------|
| **Tooltips** | Alpine `x-show` + absolute positioning | Hover hints, icon explanations |
| **Step Tours** | Livewire component with step state | Feature introductions, onboarding |
| **Spotlights** | Backdrop overlay + highlight box | Draw attention to new features |
| **Contextual Help** | Collapsible panels, popovers | Form guidance, complex workflows |
| **Beacon Pulsers** | CSS animations + Alpine toggle | Unseen feature notifications |

## Common Patterns

### Tooltip Component
```blade
{{-- resources/views/components/tooltip.blade.php --}}
<div x-data="{ open: false }" class="relative inline-block">
    <span @mouseenter="open = true" @mouseleave="open = false">
        {{ $trigger }}
    </span>
    <div x-show="open" x-transition class="absolute z-50 px-2 py-1 text-sm text-white bg-gray-900 rounded">
        {{ $slot }}
    </div>
</div>
```

### Tour Step State (Livewire)
```php
// Track progress in component
public int $currentStep = 1;
public bool $tourActive = true;

public function nextStep(): void
{
    $this->currentStep++;
}

public function dismissTour(): void
{
    $this->tourActive = false;
    // Persist: auth()->user()->update(['tour_completed' => true]);
}
```

### Spotlight Overlay
```blade
<div x-show="spotlight" class="fixed inset-0 z-40 bg-black/50">
    <div class="absolute border-2 border-blue-500 rounded-lg shadow-lg"
         :style="`top: ${highlightY}px; left: ${highlightX}px; width: ${highlightW}px; height: ${highlightH}px`">
    </div>
</div>
```

### Tailwind Animation Utilities
- `animate-pulse` — Beacon attention grabbers
- `animate-bounce` — Arrows, directional cues
- `transition-opacity duration-300` — Smooth fades
- `scale-100` → `scale-105` — Subtle emphasis

### Persistence Patterns
- Database: `users.onboarding_completed_at`
- LocalStorage: Alpine `x-data` with `localStorage.getItem('tour-seen')`
- Session: Flash messages for one-time guidance

### Accessibility
- Add `role="tooltip"`, `aria-describedby` to triggers
- Trap focus in modals with `x-trap` (Alpine plugin)
- Respect `prefers-reduced-motion` with Tailwind `motion-safe:`