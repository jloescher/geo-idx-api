# On-Page UX Patterns

When to use: Designing comparison layouts, alternative suggestion flows, and decision-support interfaces that guide users toward subscription upgrades.

## Patterns

### Teaser-Gated Comparison

Limit non-subscribed comparison views to 3 items using the `BridgeTeaser` service pattern:

```php
// In controller or Livewire
if (!$user?->subscribed()) {
    $alternatives = collect($allPlans)->take(3);
    $upgradeCta = true;
}
```

Show blurred/faded rows beyond the limit with a clear CTA: "Upgrade to compare all plans."

### Alternative Discovery Cards

At decision friction points, surface relevant alternatives:

```blade
<div class="bg-gray-50 rounded-lg p-4 mt-6">
    <h4>Not sure? Consider these alternatives:</h4>
    <div class="flex gap-4 mt-2">
        @foreach($alternatives as $alt)
            <a href="{{ route('marketing.compare.show', $alt) }}" class="text-indigo-600 hover:underline">
                {{ $alt['name'] }} — ${{ $alt['monthly'] }}/mo
            </a>
        @endforeach
    </div>
</div>
```

### Highlighted Differences Table

Use a feature matrix with checkmarks/dashes. Highlight rows where plans differ:

```blade
<tr @class(['bg-yellow-50' => $feature['differs']])>
    <td>{{ $feature['name'] }}</td>
    @foreach($plans as $plan)
        <td>@if($plan->hasFeature($feature))✓@else—@endif</td>
    @endforeach
</tr>
```

## Pitfalls

Do not show unlimited MLS listings in comparison previews—always apply the 3-item teaser cap (`BridgeTeaser::TEASER_LIMIT`) to maintain compliance with Stellar MLS data usage agreements on marketing pages.

-----