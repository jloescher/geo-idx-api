# Technical Implementation Reference

When to use: Building comparison features, alternative pages, or discovery hubs that require Livewire components, Blade templates, and Laravel routing patterns.

## Patterns

### Livewire 3 Component with Billing Toggle

```php
// app/Livewire/Marketing/PlanComparison.php
class PlanComparison extends Component
{
    public string $billingInterval = 'monthly';
    
    public function toggleInterval(): void
    {
        $this->billingInterval = $this->billingInterval === 'monthly' ? 'yearly' : 'monthly';
    }
    
    public function render()
    {
        return view('livewire.marketing.plan-comparison', [
            'plans' => SubscriptionCatalog::getPlans(),
        ]);
    }
}
```

Use the `SalesLandingPage` pattern: store toggle state in the component, pass computed pricing to the view, and persist selection via `#[Url]` for shareable comparison URLs.

### Comparison Grid with Tailwind CSS 4

```blade
<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
    @foreach($plans as $plan)
        <div @class([
            'border-2 rounded-lg p-6',
            'border-indigo-600 ring-4 ring-indigo-100' => $plan['recommended'],
            'border-gray-200' => !$plan['recommended'],
        ])>
            <h3>{{ $plan['name'] }}</h3>
            <p class="text-3xl font-bold">${{ $plan[$billingInterval] }}</p>
        </div>
    @endforeach
</div>
```

### Route Registration with Named Groups

```php
// routes/web.php
Route::prefix('compare')->name('marketing.compare.')->group(function () {
    Route::get('plans', [CompareController::class, 'plans'])->name('plans');
    Route::get('alternatives/{plan}', [CompareController::class, 'alternatives'])->name('alternatives');
});
```

## Pitfalls

Avoid hardcoding plan prices in Blade templates—always source from `SubscriptionCatalog::getPlans()` or `config/billing.php` to ensure Stripe price IDs and display amounts stay synchronized when tiers change.

-----