# Product Analytics Patterns

## When to use

When tracking user behavior across the GHL OAuth flow, widget embed performance, Bridge API proxy usage, and subscription tier conversions. Essential for understanding which dashboard features drive retention and where users abandon the installation flow.

## Patterns

### Backend Event Tracking

```php
// app/Traits/TracksProductEvents.php
trait TracksProductEvents
{
    public function track(string $event, array $properties = []): void
    {
        ProductEvent::create([
            'user_id' => auth()->id(),
            'event' => $event,
            'properties' => $properties,
            'context' => [
                'url' => request()->url(),
                'referrer' => request()->header('referer'),
                'user_agent' => request()->userAgent(),
            ],
            'created_at' => now(),
        ]);
    }
}

// Usage in Livewire components
class WidgetConfig extends Component
{
    use TracksProductEvents;
    
    public function saveConfig(): void
    {
        $this->track('widget.config.saved', [
            'theme' => $this->theme,
            'gate_enabled' => $this->requireOtp,
            'domain_count' => count($this->domains),
        ]);
    }
}
```

### Frontend Interaction Tracking

```blade
{{-- Alpine-based interaction tracking --}}
<div x-data="{ 
    track(action) {
        fetch('/api/analytics/event', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'X-CSRF-TOKEN': '{{ csrf_token() }}' },
            body: JSON.stringify({ 
                action, 
                page: 'dashboard',
                metadata: { component: 'subscription-card' }
            })
        })
    }
}">
    <button @click="track('subscription.upgrade.clicked')" 
            wire:click="initiateCheckout"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg">
        Upgrade Plan
    </button>
</div>
```

### Funnel Stage Tracking

```php
// app/Services/OnboardingFunnel.php
class OnboardingFunnel
{
    public function recordStage(string $stage, ?User $user = null): void
    {
        $user ??= auth()->user();
        
        // Only track progression, not regression
        $stages = ['oauth_started', 'oauth_completed', 'urls_registered', 'widget_embedded', 'first_lead'];
        $currentIndex = array_search($user->onboarding_stage, $stages);
        $newIndex = array_search($stage, $stages);
        
        if ($newIndex > $currentIndex) {
            $user->update(['onboarding_stage' => $stage]);
            
            // Time between stages
            ProductEvent::create([
                'user_id' => $user->id,
                'event' => 'funnel.progression',
                'properties' => [
                    'from_stage' => $stages[$currentIndex] ?? null,
                    'to_stage' => $stage,
                    'hours_elapsed' => $user->created_at->diffInHours(now()),
                ],
            ]);
        }
    }
}
```

### Dashboard Metrics Aggregation

```php
// app/Livewire/Dashboard/Metrics.php
class Metrics extends Component
{
    public function render()
    {
        $user = auth()->user();
        $location = $user->ghlLocation;
        
        return view('livewire.dashboard.metrics', [
            'activationScore' => $this->calculateActivationScore($user),
            'usageTrend' => $this->get7DayTrend($location),
            'conversionLikelihood' => $this->predictConversion($user),
        ]);
    }
    
    private function calculateActivationScore(User $user): int
    {
        $score = 0;
        if ($user->ghl_oauth_token) $score += 25;
        if ($user->registered_urls()->exists()) $score += 25;
        if ($user->widget_embeds_count > 0) $score += 25;
        if ($user->leads()->exists()) $score += 25;
        return $score;
    }
}
```

## Pitfalls

Never track PII in analytics events—hash GHL location IDs and user emails before logging. The Bridge proxy audit logs (`bridge_proxy_audit_logs`) already capture MLS compliance data; don't duplicate this in product analytics. Respect the `GHL_AUDIT_LOG_ENABLED` flag and user privacy settings. Avoid synchronous analytics writes that could slow down the OAuth callback or widget lead ingestion; queue event persistence or use a fire-and-forget approach.