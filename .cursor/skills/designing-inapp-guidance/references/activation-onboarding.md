# Activation & Onboarding Patterns

## When to use

When designing first-run experiences for GHL Marketplace installations, subscriber dashboard onboarding, or introducing new features to existing users. Focus on getting users to their first "aha moment"—registering their first MLS domain or completing OAuth.

## Patterns

### Installation Wizard Stepper

```php
// app/Livewire/Onboarding/InstallationWizard.php
class InstallationWizard extends Component
{
    public int $step = 1;
    public array $completed = [];
    
    public function mount()
    {
        // Resume from last completed step
        $this->step = auth()->user()->onboarding_step ?? 1;
    }
    
    public function markComplete(): void
    {
        $this->completed[] = $this->step;
        auth()->user()->increment('onboarding_step');
        
        if ($this->step === 4) {
            auth()->user()->update(['onboarded_at' => now()]);
        }
    }
}
```

```blade
{{-- resources/views/livewire/onboarding/installation-wizard.blade.php --}}
<div x-data="{ step: @entangle('step') }" class="max-w-2xl mx-auto">
    <div class="flex items-center justify-between mb-8">
        @foreach([1 => 'OAuth', 2 => 'URLs', 3 => 'Verify', 4 => 'Launch'] as $num => $label)
            <div class="flex flex-col items-center" 
                 :class="step >= {{ $num }} ? 'text-blue-600' : 'text-gray-400'">
                <div class="w-10 h-10 rounded-full flex items-center justify-center border-2"
                     :class="step > {{ $num }} ? 'bg-green-500 border-green-500 text-white' : 
                             step === {{ $num }} ? 'border-blue-600 bg-blue-50' : 'border-gray-300'">
                    <span x-show="step > {{ $num }}">✓</span>
                    <span x-show="step <= {{ $num }}">{{ $num }}</span>
                </div>
                <span class="mt-2 text-sm font-medium">{{ $label }}</span>
            </div>
        @endforeach
    </div>
    
    <div x-show="step === 1" x-transition class="bg-white rounded-lg shadow p-6">
        {{-- OAuth connect content --}}
    </div>
</div>
```

### Progressive Disclosure Empty States

```blade
{{-- Dashboard before first widget installation --}}
<div class="text-center py-16 px-4">
    <div class="animate-pulse bg-blue-100 w-24 h-24 rounded-full mx-auto mb-6 flex items-center justify-center">
        <x-icon-rocket class="w-12 h-12 text-blue-600"/>
    </div>
    
    <h2 class="text-2xl font-bold text-gray-900 mb-2">Activate Your IDX Integration</h2>
    <p class="text-gray-600 max-w-md mx-auto mb-8">
        Complete 3 steps to embed MLS listings on your website and start capturing leads.
    </p>
    
    <div class="space-y-4 max-w-sm mx-auto">
        <a href="{{ route('oauth.start') }}" class="flex items-center p-4 bg-white border rounded-lg hover:border-blue-500 group">
            <span class="w-8 h-8 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center font-bold mr-4 group-hover:bg-blue-600 group-hover:text-white transition">1</span>
            <div class="flex-1">
                <p class="font-medium text-gray-900">Connect GoHighLevel</p>
                <p class="text-sm text-gray-500">Authorize your location</p>
            </div>
            <x-icon-chevron-right class="w-5 h-5 text-gray-400 group-hover:text-blue-600"/>
        </a>
        
        <div class="flex items-center p-4 bg-gray-50 border border-gray-200 rounded-lg opacity-60">
            <span class="w-8 h-8 rounded-full bg-gray-200 text-gray-500 flex items-center justify-center font-bold mr-4">2</span>
            <p class="text-gray-500">Register your domain URLs</p>
        </div>
    </div>
</div>
```

### Contextual Checklist Sidebar

```blade
<div class="fixed right-0 top-20 w-80 bg-white shadow-lg border-l p-4"
     x-data="{ open: true }" 
     x-show="open"
     x-transition:enter="transform transition ease-in-out duration-300"
     x-transition:enter-start="translate-x-full"
     x-transition:enter-end="translate-x-0">
    
    <div class="flex items-center justify-between mb-4">
        <h3 class="font-semibold text-gray-900">Setup Checklist</h3>
        <button @click="open = false" class="text-gray-400 hover:text-gray-600">×</button>
    </div>
    
    <div class="space-y-3">
        @foreach($checklist as $item)
            <div class="flex items-start" wire:key="check-{{ $item->id }}">
                <input type="checkbox" 
                       wire:click="toggle({{ $item->id }})"
                       @checked($item->completed)
                       class="mt-1 rounded border-gray-300 text-blue-600 focus:ring-blue-500">
                <span class="ml-3 text-sm {{ $item->completed ? 'text-gray-400 line-through' : 'text-gray-700' }}">
                    {{ $item->label }}
                </span>
            </div>
        @endforeach
        
        <div class="pt-3 border-t">
            <div class="flex items-center justify-between text-sm">
                <span class="text-gray-600">Progress</span>
                <span class="font-medium text-blue-600">{{ $percent }}%</span>
            </div>
            <div class="mt-2 h-2 bg-gray-200 rounded-full overflow-hidden">
                <div class="h-full bg-blue-600 transition-all duration-500" style="width: {{ $percent }}%"></div>
            </div>
        </div>
    </div>
</div>
```

## Pitfalls

Avoid blocking users from the core product while forcing onboarding completion. The GHL widget installation should be skippable—store `onboarding_skipped_at` and allow users to resume later via a persistent "Complete Setup" button in the dashboard header. Never require domain URL registration before OAuth callback; the session can temporarily hold the pending state.