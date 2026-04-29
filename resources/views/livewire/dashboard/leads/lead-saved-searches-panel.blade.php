<?php

use App\Models\LeadSavedSearch;
use Livewire\Volt\Component;

new class extends Component {
    public string $name = '';
    public string $status = '';

    public function saveSearch(): void
    {
        $this->validate([
            'name' => ['required', 'string', 'max:120'],
            'status' => ['nullable', 'string', 'max:40'],
        ]);

        $user = auth()->user();
        if ($user === null) {
            return;
        }

        LeadSavedSearch::query()->create([
            'user_id' => $user->id,
            'name' => $this->name,
            'filters' => ['status' => $this->status],
            'is_alert_enabled' => false,
            'alert_frequency' => 'daily',
        ]);

        $this->name = '';
        $this->status = '';
        session()->flash('dashboard_status', 'Saved search created.');
    }

    public function toggleAlert(int $id): void
    {
        $user = auth()->user();
        if ($user === null) {
            return;
        }

        $search = LeadSavedSearch::query()
            ->where('user_id', $user->id)
            ->findOrFail($id);
        $search->is_alert_enabled = ! $search->is_alert_enabled;
        $search->save();
    }

    /** @return array{searches: mixed} */
    public function with(): array
    {
        $user = auth()->user();
        $searches = $user?->leadSavedSearches()->latest('id')->get() ?? collect();

        return ['searches' => $searches];
    }
}; ?>

<div class="idx-card p-4">
    <h3 class="text-sm font-semibold text-white">Saved Searches</h3>
    <p class="mt-1 text-xs text-slate-400">Create quick filters like “Hot leads” or “New this week”.</p>

    <form wire:submit="saveSearch" class="mt-3 grid gap-2 md:grid-cols-3">
        <input type="text" wire:model="name" placeholder="Search name" class="idx-input px-3 py-2 text-xs md:col-span-2">
        <select wire:model="status" class="idx-input px-3 py-2 text-xs">
            <option value="">Any status</option>
            <option value="new">New</option>
            <option value="hot">Hot</option>
            <option value="contacted">Contacted</option>
            <option value="converted">Converted</option>
        </select>
        <button type="submit" class="idx-btn-primary px-3 py-2 text-xs font-semibold md:col-span-3">+ New Custom Search</button>
    </form>

    <div class="mt-3 flex flex-wrap gap-2">
        @foreach ($searches as $search)
            <div class="inline-flex items-center gap-2 rounded-full border border-white/15 bg-slate-950/70 px-3 py-1 text-xs">
                <span>{{ $search->name }}</span>
                <button type="button" wire:click="toggleAlert({{ $search->id }})" class="rounded-full px-2 py-0.5 {{ $search->is_alert_enabled ? 'bg-emerald-500/30 text-emerald-200' : 'bg-slate-700 text-slate-200' }}">
                    {{ $search->is_alert_enabled ? 'Alert on' : 'Alert off' }}
                </button>
            </div>
        @endforeach
    </div>
</div>

