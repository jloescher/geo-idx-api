# Programmatic SEO Patterns

When to use: Generating comparison pages at scale, creating alternative suggestion flows, or building dynamic discovery hubs based on subscription catalog data.

## Patterns

### Dynamic Comparison Routes

Generate comparison pairs programmatically:

```php
// routes/web.php - dynamic plan comparisons
Route::get('compare/{planA}-vs-{planB}', function ($planA, $planB) {
    $catalog = app(SubscriptionCatalog::class);
    
    return view('marketing.compare.show', [
        'left' => $catalog->getPlan($planA),
        'right' => $catalog->getPlan($planB),
        'alternatives' => $catalog->getAdjacentPlans($planA),
    ]);
})->whereIn(['planA', 'planB'], ['pro', 'smart', 'ultra', 'mega']);
```

### Canonical URL Generation

Prevent duplicate content when multiple paths compare the same plans:

```php
// In controller
$canonical = route('marketing.compare.show', [
    'planA' => min($planA, $planB), // Alphabetical ordering
    'planB' => max($planA, $planB),
]);
```

### Widget-Embed Comparison Snippets

Reuse comparison tables for external sites via widget routes:

```php
// routes/ghl-widget.php
Route::get('widget/compare-snippet/{apiKey}', [WidgetController::class, 'compareSnippet'])
    ->middleware(['widget.key', 'widget.origin']);
```

## Pitfalls

Avoid creating comparison pages for every possible feature combination—this generates thin content. Focus on high-intent pairs (Pro vs Smart, Ultra vs Mega) and ensure each page has unique copy beyond the feature matrix.

-----