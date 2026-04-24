# In-App Guidance Patterns

## When to use

When users need contextual help completing complex workflows like OAuth URL registration, widget configuration, or understanding Bridge API rate limits. Deploy for form fields with MLS compliance implications or technical integration steps.

## Patterns

### Contextual Field Tooltips

```blade
{{-- Form field with inline help --}}
<div class="space-y-1" x-data="{ showHelp: false }">
    <div class="flex items-center gap-2">
        <label class="text-sm font-medium text-gray-700">Primary Domain URL</label>
        <button @click="showHelp = !showHelp" @mouseenter="showHelp = true" class="text-gray-400 hover:text-gray-600">
            <x-icon-question-mark-circle class="w-4 h-4"/>
        </button>
    </div>
    
    <input type="url" wire:model="primaryUrl" 
           class="w-full rounded-lg border-gray-300 focus:border-blue-500 focus:ring-blue-500"
           placeholder="https://yoursite.com"/>
    
    <div x-show="showHelp" 
         x-transition:enter="transition ease-out duration-200"
         x-transition:enter-start="opacity-0 translate-y-1"
         x-transition:enter-end="opacity-100 translate-y-0"
         class="text-sm text-gray-600 bg-blue-50 p-3 rounded-lg mt-2">
        <p class="font-medium text-blue-900 mb-1">Why this matters for MLS compliance</p>
        <p>Your registered domain must match the site where you embed the widget. Stellar MLS requires accurate domain reporting for IDX display authorization.</p>
        <a href="{{ route('docs.mls-compliance') }}" class="text-blue-600 hover:underline mt-1 inline-block">Learn more →</a>
    </div>
    
    @error('primaryUrl') 
        <p class="text-sm text-red-600">{{ $message }}</p>
    @enderror
</div>
```

### Interactive Tour Steps

```php
// app/Livewire/GuidedTour.php
class GuidedTour extends Component
{
    public array $steps = [
        ['target' => 'nav-domains', 'title' => 'Domain Management', 'content' => 'Register approved MLS domains here.'],
        ['target' => 'nav-widgets', 'title' => 'Widget Configuration', 'content' => 'Customize your embed code and styling.'],
        ['target' => 'nav-analytics', 'title' => 'Usage Analytics', 'content' => 'Monitor API calls and lead volume.'],
    ];
    
    public int $currentStep = 0;
    public bool $active = false;
    
    public function start(): void
    {
        $this->active = true;
        $this->dispatch('tour-step', step: 0);
    }
    
    public function next(): void
    {
        if ($this->currentStep < count($this->steps) - 1) {
            $this->currentStep++;
            $this->dispatch('tour-step', step: $this->currentStep);
        } else {
            $this->complete();
        }
    }
    
    public function complete(): void
    {
        auth()->user()->update(['tour_completed_at' => now()]);
        $this->active = false;
    }
}
```

```blade
{{-- Tour overlay with spotlight --}}
<div x-data="{ active: @entangle('active'), step: @entangle('currentStep') }" x-show="active">
    <div class="fixed inset-0 bg-black/50 z-40"></div>
    
    @foreach($steps as $index => $step)
        <div x-show="step === {{ $index }}" 
             x-transition
             class="fixed z-50 max-w-sm"
             :style="$refs['target-{{ $step['target'] }}'] && getPosition($refs['target-{{ $step['target'] }}'])">
            
            <div class="bg-white rounded-lg shadow-xl p-4 border-2 border-blue-500">
                <p class="font-semibold text-gray-900 mb-1">Step {{ $index + 1 }}: {{ $step['title'] }}</p>
                <p class="text-sm text-gray-600 mb-4">{{ $step['content'] }}</p>
                
                <div class="flex justify-between items-center">
                    <span class="text-xs text-gray-400">{{ $index + 1 }} of {{ count($steps) }}</span>
                    <div class="space-x-2">
                        <button @click="active = false" class="text-sm text-gray-600">Skip</button>
                        <button wire:click="next" class="px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700">
                            {{ $index === count($steps) - 1 ? 'Finish' : 'Next' }}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    @endforeach
</div>
```

### Form Validation Guidance

```blade
{{-- Inline validation with actionable fixes --}}
<div wire:loading.class="opacity-50" class="space-y-4">
    <div>
        <label class="block text-sm font-medium text-gray-700">GHL Location ID</label>
        <input wire:model.live="locationId" type="text" 
               class="mt-1 block w-full rounded-lg border {{ $errors->has('locationId') ? 'border-red-300 focus:border-red-500 focus:ring-red-500' : 'border-gray-300 focus:border-blue-500' }}"/>
        
        @if($errors->has('locationId'))
            <div class="mt-2 flex items-start gap-2 text-sm text-red-600">
                <x-icon-exclamation-circle class="w-4 h-4 mt-0.5 flex-shrink-0"/>
                <div>
                    <p class="font-medium">Location not found</p>
                    <p class="text-red-500">Make sure you've installed the app from the GHL Marketplace and granted permissions.</p>
                    <a href="{{ route('ghl.install') }}" class="text-red-700 underline mt-1 inline-block">Re-run OAuth flow →</a>
                </div>
            </div>
        @endif
    </div>
</div>
```

## Pitfalls

Don't block user input with persistent overlays. Tours must be dismissible with an explicit "Skip Tour" option that sets a preference. Avoid auto-starting tours on every page load—check `tour_completed_at` or `tour_dismissed_at` timestamps first. The OAuth callback flow is especially sensitive to interference; never inject guidance UI that could intercept the critical token exchange POST request.