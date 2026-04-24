# Feedback & Insights Patterns

## When to use

When collecting qualitative input from GHL Marketplace users, gathering NPS after key milestones, or capturing error context when widget embeds fail. Deploy for UX research recruitment and feature prioritization voting.

## Patterns

### Micro-Survey Modal

```php
// app/Livewire/Feedback/MicroSurvey.php
class MicroSurvey extends Component
{
    public bool $show = false;
    public int $step = 1;
    public array $responses = [];
    
    protected $listeners = ['trigger-survey' => 'open'];
    
    public function open(string $trigger): void
    {
        $this->trigger = $trigger;
        $this->show = true;
        
        // Don't survey the same user within 30 days
        if (cache()->has("survey:{$this->user->id}")) {
            $this->show = false;
            return;
        }
        
        $this->questions = match($trigger) {
            'post-oauth' => [
                ['id' => 'difficulty', 'text' => 'How easy was the GHL connection?', 'type' => 'scale'],
                ['id' => 'blocked', 'text' => 'What almost stopped you?', 'type' => 'text'],
            ],
            'first-lead' => [
                ['id' => 'source', 'text' => 'Where did this lead come from?', 'type' => 'choice', 'options' => ['Website', 'Social Media', 'Referral']],
            ],
            default => [],
        };
    }
    
    public function submit(): void
    {
        FeedbackResponse::create([
            'user_id' => auth()->id(),
            'trigger' => $this->trigger,
            'responses' => $this->responses,
            'nps_score' => $this->responses['nps'] ?? null,
        ]);
        
        cache()->put("survey:{$this->user->id}", true, now()->addDays(30));
        $this->show = false;
        $this->dispatch('survey-completed');
    }
}
```

```blade
<div x-data="{ show: @entangle('show'), step: @entangle('step') }" x-show="show" class="fixed inset-0 z-50">
    <div class="absolute inset-0 bg-black/50" @click="show = false"></div>
    
    <div class="absolute bottom-4 right-4 w-96 bg-white rounded-xl shadow-2xl p-6">
        <div class="flex items-center justify-between mb-4">
            <h3 class="font-semibold text-gray-900">Quick Question</h3>
            <button @click="show = false" class="text-gray-400 hover:text-gray-600">×</button>
        </div>
        
        @foreach($questions as $index => $question)
            <div x-show="step === {{ $index + 1 }}" x-transition>
                <p class="text-gray-700 mb-4">{{ $question['text'] }}</p>
                
                @if($question['type'] === 'scale')
                    <div class="flex gap-2 justify-center">
                        @foreach(range(1, 5) as $n)
                            <button wire:click="$set('responses.{{ $question['id'] }}', {{ $n }})" 
                                    class="w-10 h-10 rounded-lg border-2 font-medium
                                    {{ $responses[$question['id']] === $n ? 'border-blue-500 bg-blue-50 text-blue-600' : 'border-gray-200 hover:border-gray-300' }}">
                                {{ $n }}
                            </button>
                        @endforeach
                    </div>
                    <div class="flex justify-between text-xs text-gray-500 mt-2">
                        <span>Very difficult</span>
                        <span>Very easy</span>
                    </div>
                @endif
                
                @if($question['type'] === 'text')
                    <textarea wire:model="responses.{{ $question['id'] }}" 
                              class="w-full rounded-lg border-gray-300" rows="3"
                              placeholder="Tell us what happened..."></textarea>
                @endif
            </div>
        @endforeach
        
        <div class="flex justify-between mt-6">
            <button @click="step--" x-show="step > 1" class="text-gray-600">Back</button>
            <button @click="step++" x-show="step < count($questions)" class="px-4 py-2 bg-blue-600 text-white rounded-lg">Next</button>
            <button wire:click="submit" x-show="step === count($questions)" class="px-4 py-2 bg-blue-600 text-white rounded-lg">Submit</button>
        </div>
        
        <div class="mt-4 h-1 bg-gray-100 rounded-full overflow-hidden">
            <div class="h-full bg-blue-600 transition-all duration-300" style="width: {{ ($step / count($questions)) * 100 }}%"></div>
        </div>
    </div>
</div>
```

### Inline Error Feedback

```blade
{{-- Error boundary with feedback capture --}}
<div x-data="{ error: null, feedback: '' }" 
     @error-captured.window="error = $event.detail">
    
    <template x-if="error">
        <div class="bg-red-50 border border-red-200 rounded-lg p-4">
            <div class="flex items-start gap-3">
                <x-icon-x-circle class="w-5 h-5 text-red-600 mt-0.5"/>
                <div class="flex-1">
                    <h4 class="font-medium text-red-900">Something went wrong</h4>
                    <p class="text-sm text-red-700 mt-1" x-text="error.message"></p>
                    
                    <div class="mt-4">
                        <p class="text-sm font-medium text-red-900 mb-2">Help us fix this:</p>
                        <textarea x-model="feedback" 
                                  class="w-full rounded-lg border-red-300 text-sm"
                                  placeholder="What were you trying to do?"></textarea>
                        <button @click="$wire.submitErrorFeedback({error: error.code, feedback: feedback})"
                                class="mt-2 px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700">
                            Send Feedback
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </template>
</div>
```

### Feature Voting

```blade
<div class="bg-white rounded-lg shadow p-6">
    <h3 class="font-semibold text-gray-900 mb-2">What's next?</h3>
    <p class="text-sm text-gray-600 mb-4">Vote on features you'd like to see.</p>
    
    <div class="space-y-3">
        @foreach($features as $feature)
            <div wire:key="vote-{{ $feature->id }}" 
                 class="flex items-center justify-between p-3 bg-gray-50 rounded-lg group hover:bg-gray-100">
                <div>
                    <p class="font-medium text-gray-900">{{ $feature->title }}</p>
                    <p class="text-xs text-gray-500">{{ $feature->votes_count }} votes</p>
                </div>
                
                <button wire:click="vote({{ $feature->id }})"
                        wire:loading.attr="disabled"
                        class="px-3 py-1 text-sm rounded-full transition
                        {{ $feature->user_voted ? 'bg-blue-100 text-blue-700' : 'bg-white text-gray-600 border hover:bg-gray-50' }}">
                    @if($feature->user_voted)
                        <span class="flex items-center gap-1">
                            <x-icon-check class="w-4 h-4"/> Voted
                        </span>
                    @else
                        Vote
                    @endif
                </button>
            </div>
        @endforeach
    </div>
    
    <form wire:submit="suggestFeature" class="mt-4 pt-4 border-t">
        <div class="flex gap-2">
            <input wire:model="suggestion" type="text" 
                   class="flex-1 rounded-lg border-gray-300 text-sm"
                   placeholder="Suggest a feature..."/>
            <button type="submit" class="px-4 py-2 bg-gray-900 text-white text-sm rounded-lg hover:bg-gray-800">
                Suggest
            </button>
        </div>
    </form>
</div>
```

## Pitfalls

Respect the 30-day survey cooldown strictly—over-surveying GHL agency users damages trust and can trigger uninstalls. Never require feedback submission to proceed with OAuth or widget installation; always provide a "Skip" option. Store feedback responses in a separate table from audit logs (`ghl_audit_logs` is for MLS compliance, not product feedback). Be aware that some agencies have strict data policies; clearly label optional vs required fields and provide a privacy notice link.

=====