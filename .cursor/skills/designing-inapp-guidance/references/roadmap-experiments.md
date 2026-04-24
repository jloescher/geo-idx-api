# Roadmap & Experiments Patterns

## When to use

When A/B testing pricing page layouts, validating new widget themes, or rolling out GIS parcel map features gradually. Use for experiments that require backend persistence and user cohort assignment.

## Patterns

### Feature Flag Component

```php
// app/Services/FeatureFlags.php
class FeatureFlags
{
    public function isEnabled(string $flag, ?User $user = null): bool
    {
        $user ??= auth()->user();
        
        // Global kill switch
        if (config("features.{$flag}.enabled") === false) {
            return false;
        }
        
        // Percentage-based rollout
        $percentage = config("features.{$flag}.rollout_percentage", 0);
        if ($user && $this->userInPercentage($user->id, $percentage)) {
            return true;
        }
        
        // Specific user allowlist
        return in_array($user?->email, config("features.{$flag}.allowed_users", []));
    }
    
    private function userInPercentage(string $userId, int $percentage): bool
    {
        return (crc32($userId) % 100) < $percentage;
    }
}
```

```blade
{{-- Conditional feature rendering --}}
@if(app(FeatureFlags::class)->isEnabled('gis-parcel-overlay'))
    <livewire:gis.parcel-map />
@else
    <div class="bg-gray-100 rounded-lg p-8 text-center">
        <p class="text-gray-600">Map enhancements coming soon</p>
        <button wire:click="joinWaitlist" class="mt-2 text-blue-600 hover:underline">Join waitlist</button>
    </div>
@endif
```

### Experiment Assignment

```php
// app/Models/ExperimentAssignment.php
class ExperimentAssignment extends Model
{
    protected $fillable = ['user_id', 'experiment', 'variant', 'assigned_at'];
    
    public static function assign(User $user, string $experiment): string
    {
        $existing = static::where('user_id', $user->id)
            ->where('experiment', $experiment)
            ->first();
            
        if ($existing) {
            return $existing->variant;
        }
        
        // Consistent hashing for sticky assignment
        $hash = crc32("{$user->id}:{$experiment}");
        $variant = $hash % 2 === 0 ? 'control' : 'treatment';
        
        static::create([
            'user_id' => $user->id,
            'experiment' => $experiment,
            'variant' => $variant,
            'assigned_at' => now(),
        ]);
        
        return $variant;
    }
}
```

### Dashboard Experiment Component

```php
// app/Livewire/Dashboard/PricingExperiment.php
class PricingExperiment extends Component
{
    public string $variant;
    
    public function mount()
    {
        $this->variant = ExperimentAssignment::assign(auth()->user(), 'pricing_page_2026_04');
        $this->track('experiment.viewed', ['variant' => $this->variant]);
    }
    
    public function render()
    {
        return view("livewire.dashboard.pricing.{$this->variant}");
    }
    
    public function selectPlan(string $plan): void
    {
        $this->track('experiment.conversion', [
            'variant' => $this->variant,
            'plan' => $plan,
        ]);
        
        $this->dispatch('plan-selected', plan: $plan);
    }
}
```

```blade
{{-- Control variant: pricing table --}}
@if($variant === 'control')
    <div class="grid grid-cols-3 gap-6">
        @foreach($plans as $plan)
            <div class="border rounded-lg p-6 {{ $plan->featured ? 'border-blue-500 ring-2 ring-blue-500' : '' }}">
                <h3 class="font-semibold text-lg">{{ $plan->name }}</h3>
                <p class="text-3xl font-bold mt-2">${{ $plan->price }}<span class="text-sm text-gray-500">/mo</span></p>
                <button wire:click="selectPlan('{{ $plan->id }}')" class="mt-4 w-full py-2 bg-blue-600 text-white rounded-lg">
                    Select {{ $plan->name }}
                </button>
            </div>
        @endforeach
    </div>

{{-- Treatment variant: feature comparison --}}
@else
    <div class="overflow-x-auto">
        <table class="w-full">
            <thead>
                <tr>
                    <th class="text-left p-4">Plan</th>
                    @foreach($plans as $plan)
                        <th class="p-4 {{ $plan->featured ? 'bg-blue-50' : '' }}">
                            {{ $plan->name }}
                            @if($plan->featured)
                                <span class="block text-xs text-blue-600 font-normal">Most Popular</span>
                            @endif
                        </th>
                    @endforeach
                </tr>
            </thead>
            <tbody>
                @foreach($features as $feature)
                    <tr class="border-t">
                        <td class="p-4 font-medium">{{ $feature->name }}</td>
                        @foreach($plans as $plan)
                            <td class="p-4 text-center {{ $plan->featured ? 'bg-blue-50' : '' }}">
                                @if(in_array($feature->id, $plan->features))
                                    <x-icon-check class="w-5 h-5 text-green-500 mx-auto"/>
                                @else
                                    <span class="text-gray-300">—</span>
                                @endif
                            </td>
                        @endforeach
                    </tr>
                @endforeach
            </tbody>
        </table>
    </div>
@endif
```

## Pitfalls

Never change experiment assignments after they've been persisted—this invalidates your metrics and creates user confusion when UI suddenly changes. Always include `experiment.assigned_at` timestamps in analysis to account for seasonality. Avoid running experiments during the GHL Marketplace review process; stick to the stable control variant to prevent rejection. Document all active experiments in `config/features.php` with planned end dates and success criteria.