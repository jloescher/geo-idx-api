# Engagement & Adoption Patterns

## When to use

When users have completed initial onboarding but aren't utilizing key features like widget embedding, API token generation, or GIS map overlays. Target users who installed the GHL app but haven't generated leads in 7+ days.

## Patterns

### Feature Discovery Beacons

```blade
{{-- Pulser for new feature --}}
<div class="relative inline-block" x-data="{ seen: localStorage.getItem('feature-gis-maps') }">
    <button @click="seen = '1'; localStorage.setItem('feature-gis-maps', '1')"
            class="relative p-2 text-gray-600 hover:text-gray-900">
        <x-icon-map class="w-6 h-6"/>
        
        <span x-show="!seen" 
              class="absolute -top-1 -right-1 flex h-3 w-3">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
            <span class="relative inline-flex rounded-full h-3 w-3 bg-red-500"></span>
        </span>
    </button>
    
    <div x-show="!seen" x-transition.opacity class="absolute right-0 mt-2 w-64 bg-white rounded-lg shadow-lg border p-4 z-50">
        <p class="font-medium text-gray-900 mb-1">New: GIS Parcel Maps</p>
        <p class="text-sm text-gray-600 mb-3">Add parcel boundaries to your IDX listings.</p>
        <button @click="seen = '1'; localStorage.setItem('feature-gis-maps', '1')"
                class="text-sm text-blue-600 hover:text-blue-800">Got it</button>
    </div>
</div>
```

### Usage-Based Prompts

```php
// app/Livewire/Dashboard/EngagementPrompt.php
class EngagementPrompt extends Component
{
    public function render()
    {
        $user = auth()->user();
        $location = $user->ghlLocation;
        
        // Determine which prompt to show based on usage
        $prompt = match(true) {
            // Installed but no widget embeds
            $location && $location->installed_at && !$location->widget_embeds_count => 'embed-widget',
            // Has widget but no leads in 7 days  
            $location->widget_embeds_count > 0 && $location->leads()->where('created_at', '>', now()->subDays(7))->count() === 0 => 'no-leads',
            // Heavy API usage approaching limit
            $user->api_calls_this_month > $user->plan->api_limit * 0.8 => 'approaching-limit',
            default => null
        };
        
        return view('livewire.dashboard.engagement-prompt', ['prompt' => $prompt]);
    }
}
```

```blade
{{-- Contextual banner based on usage state --}}
@if($prompt === 'embed-widget')
    <div class="bg-amber-50 border border-amber-200 rounded-lg p-4 mb-6 flex items-start gap-4">
        <div class="p-2 bg-amber-100 rounded-lg">
            <x-icon-code class="w-5 h-5 text-amber-600"/>
        </div>
        <div class="flex-1">
            <h4 class="font-medium text-amber-900">Add the widget to your website</h4>
            <p class="text-sm text-amber-800 mt-1">You've connected GHL but haven't embedded the widget. Copy your installation code and start capturing leads.</p>
            <div class="mt-3 flex gap-3">
                <a href="{{ route('widget.install') }}" class="text-sm font-medium text-amber-900 underline">Get embed code</a>
                <button wire:click="dismiss" class="text-sm text-amber-700 hover:text-amber-900">Dismiss</button>
            </div>
        </div>
    </div>
@endif
```

### Milestone Celebrations

```blade
<div x-data="{ show: @entangle('showMilestone') }" x-show="show" x-transition>
    <div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center">
        <div class="bg-white rounded-2xl p-8 max-w-md text-center animate-bounce">
            <div class="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
                <x-icon-trophy class="w-10 h-10 text-green-600"/>
            </div>
            
            <h3 class="text-2xl font-bold text-gray-900 mb-2">First Lead Captured! 🎉</h3>
            <p class="text-gray-600 mb-6">Your widget just captured its first lead. Check your GHL contacts to see the new entry.</p>
            
            <div class="flex gap-3 justify-center">
                <button wire:click="viewLead" class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
                    View Lead
                </button>
                <button wire:click="$set('showMilestone', false)" class="px-4 py-2 text-gray-600 hover:text-gray-900">
                    Keep Going
                </button>
            </div>
            
            <p class="mt-4 text-xs text-gray-500">Tip: Enable SMS notifications in settings</p>
        </div>
    </div>
</div>
```

## Pitfalls

Avoid nagging users who have explicitly dismissed prompts. Track dismissals with `dismissed_at` timestamps and exponentially back off—if they dismiss the "embed widget" prompt twice, wait 7 days before showing a gentler inline suggestion. Never show multiple engagement prompts simultaneously; prioritize the most critical action based on revenue impact (lead generation > feature discovery).