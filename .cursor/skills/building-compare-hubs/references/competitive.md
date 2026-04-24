# Competitive Analysis Patterns

When to use: Creating "vs competitor" pages, positioning against alternative IDX providers, or building alternative suggestion flows for users comparing options.

## Patterns

### Competitor Alternative Pages

Create pages targeting "alternative to {competitor}" search queries:

```php
// routes/web.php
Route::get('alternatives-to-{competitor}', [AlternativeController::class, 'show'])
    ->whereIn('competitor', ['idxbroker', 'diverse-solutions', 'showcase-idx'])
    ->name('marketing.alternatives.show');
```

Structure the content:
1. "Why agents switch from {competitor}" (pain points)
2. Side-by-side comparison table
3. Migration guide/CTA

### Differentiation Matrix

| Feature | GeoIDX Pro | IDX Broker | Advantage |
|---------|------------|------------|-----------|
| GHL Integration | Native | Zapier only | ✅ Built-in |
| GIS Parcels | Included | +$49/mo | ✅ Bundled |
| Setup Time | 5 minutes | 2-3 days | ✅ Faster |

### Switching Cost Calculator

```blade
<div x-data="{ currentPrice: 99, geoPrice: 79 }">
    <p>Save $<span x-text="(currentPrice - geoPrice) * 12"></span>/year</p>
    <input type="range" x-model="currentPrice" min="50" max="500">
</div>
```

## Pitfalls

Never make false claims about competitor pricing or features—verify publicly listed rates before publishing. Avoid naming competitors in URL slugs without legal review; use generic terms like "alternatives-to-legacy-idx" instead of trademarked names when uncertain.