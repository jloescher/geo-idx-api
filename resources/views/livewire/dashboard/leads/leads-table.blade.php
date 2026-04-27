<?php

use App\Ghl\Sync\Models\QuantyraLead;
use App\Services\SubscriberLeadScopeService;
use Illuminate\Database\Eloquent\Builder;
use Livewire\Volt\Component;
use Livewire\WithPagination;

new class extends Component {
    use WithPagination;

    public string $search = '';
    public string $status = '';
    public string $domain = '';
    public string $dateFrom = '';
    public string $dateTo = '';
    public array $selected = [];

    protected $queryString = [
        'search' => ['except' => ''],
        'status' => ['except' => ''],
        'domain' => ['except' => ''],
        'dateFrom' => ['except' => ''],
        'dateTo' => ['except' => ''],
    ];

    public function updatingSearch(): void { $this->resetPage(); }
    public function updatingStatus(): void { $this->resetPage(); }
    public function updatingDomain(): void { $this->resetPage(); }
    public function updatingDateFrom(): void { $this->resetPage(); }
    public function updatingDateTo(): void { $this->resetPage(); }

    public function markSelectedContacted(): void
    {
        if ($this->selected === []) {
            return;
        }
        QuantyraLead::query()->whereIn('id', $this->selected)->update(['payload->status' => 'contacted']);
        $this->selected = [];
        session()->flash('dashboard_status', 'Selected leads marked as Contacted.');
    }

    public function deleteSelected(): void
    {
        if ($this->selected === []) {
            return;
        }
        QuantyraLead::query()->whereIn('id', $this->selected)->delete();
        $this->selected = [];
        session()->flash('dashboard_status', 'Selected leads deleted.');
    }

    /** @return array{leads: mixed, domains: mixed} */
    public function with(SubscriberLeadScopeService $scope): array
    {
        $user = auth()->user();
        $locationIds = $user !== null ? $scope->locationIdsForUser($user) : [];

        $query = QuantyraLead::query()->whereIn('ghl_location_id', $locationIds)->latest('id');
        $query = $this->applyFilters($query);

        $leads = $query->paginate(20);
        $domains = QuantyraLead::query()
            ->whereIn('ghl_location_id', $locationIds)
            ->whereNotNull('quantyra_domain')
            ->distinct()
            ->pluck('quantyra_domain')
            ->filter()
            ->values();

        return ['leads' => $leads, 'domains' => $domains];
    }

    private function applyFilters(Builder $query): Builder
    {
        if ($this->search !== '') {
            $search = '%'.strtolower(trim($this->search)).'%';
            $query->where(function (Builder $inner) use ($search): void {
                $inner->whereRaw('LOWER(payload->>\'name\') like ?', [$search])
                    ->orWhereRaw('LOWER(payload->>\'first_name\') like ?', [$search])
                    ->orWhereRaw('LOWER(payload->>\'last_name\') like ?', [$search])
                    ->orWhereRaw('LOWER(payload->>\'email\') like ?', [$search])
                    ->orWhereRaw('LOWER(payload->>\'phone\') like ?', [$search])
                    ->orWhereRaw('LOWER(payload->>\'property_interest\') like ?', [$search]);
            });
        }

        if ($this->status !== '') {
            $query->where('payload->status', $this->status);
        }
        if ($this->domain !== '') {
            $query->where('quantyra_domain', $this->domain);
        }
        if ($this->dateFrom !== '') {
            $query->whereDate('created_at', '>=', $this->dateFrom);
        }
        if ($this->dateTo !== '') {
            $query->whereDate('created_at', '<=', $this->dateTo);
        }

        return $query;
    }
}; ?>

<div class="space-y-4">
    <div class="grid gap-3 md:grid-cols-5">
        <input type="text" wire:model.live.debounce.300ms="search" placeholder="Search leads..." class="rounded-xl border border-white/15 bg-slate-950/70 px-3 py-2 text-sm text-slate-100 md:col-span-2">
        <select wire:model.live="status" class="rounded-xl border border-white/15 bg-slate-950/70 px-3 py-2 text-sm text-slate-100">
            <option value="">All statuses</option>
            <option value="new">New</option>
            <option value="hot">Hot</option>
            <option value="contacted">Contacted</option>
            <option value="converted">Converted</option>
        </select>
        <select wire:model.live="domain" class="rounded-xl border border-white/15 bg-slate-950/70 px-3 py-2 text-sm text-slate-100">
            <option value="">All domains</option>
            @foreach ($domains as $domainOpt)
                <option value="{{ $domainOpt }}">{{ $domainOpt }}</option>
            @endforeach
        </select>
        <div class="grid grid-cols-2 gap-2">
            <input type="date" wire:model.live="dateFrom" class="rounded-xl border border-white/15 bg-slate-950/70 px-2 py-2 text-xs text-slate-100">
            <input type="date" wire:model.live="dateTo" class="rounded-xl border border-white/15 bg-slate-950/70 px-2 py-2 text-xs text-slate-100">
        </div>
    </div>

    <div class="flex flex-wrap gap-2">
        <button type="button" wire:click="markSelectedContacted" class="rounded-lg border border-cyan-400/40 px-3 py-2 text-xs font-semibold text-cyan-200 hover:bg-cyan-500/10">Mark Contacted</button>
        <button type="button" wire:click="deleteSelected" class="rounded-lg border border-rose-400/40 px-3 py-2 text-xs font-semibold text-rose-200 hover:bg-rose-500/10">Delete</button>
    </div>

    <div class="overflow-x-auto rounded-2xl border border-white/10">
        <table class="min-w-full divide-y divide-white/10 text-sm">
            <thead class="bg-slate-950/70 text-xs uppercase tracking-wide text-slate-400">
                <tr>
                    <th class="px-3 py-2 text-left">Select</th>
                    <th class="px-3 py-2 text-left">Lead</th>
                    <th class="px-3 py-2 text-left">Email / Phone</th>
                    <th class="px-3 py-2 text-left">Property Interest</th>
                    <th class="px-3 py-2 text-left">Domain</th>
                    <th class="px-3 py-2 text-left">Score</th>
                    <th class="px-3 py-2 text-left">Status</th>
                    <th class="px-3 py-2 text-left">Created</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-white/5 bg-slate-900/50 text-slate-200">
                @forelse ($leads as $lead)
                    @php($payload = is_array($lead->payload) ? $lead->payload : [])
                    <tr>
                        <td class="px-3 py-2">
                            <input type="checkbox" value="{{ $lead->id }}" wire:model.live="selected" class="rounded border-white/30 bg-slate-950">
                        </td>
                        <td class="px-3 py-2">
                            <p class="font-semibold text-slate-100">{{ $payload['name'] ?? trim(($payload['first_name'] ?? '').' '.($payload['last_name'] ?? '')) ?: 'Lead #'.$lead->id }}</p>
                            <p class="text-xs text-slate-400">ID {{ $lead->id }}</p>
                        </td>
                        <td class="px-3 py-2 text-xs">
                            <p>{{ $payload['email'] ?? '—' }}</p>
                            <p class="text-slate-400">{{ $payload['phone'] ?? '—' }}</p>
                        </td>
                        <td class="px-3 py-2 text-xs">{{ $payload['property_interest'] ?? $lead->listing_id ?? '—' }}</td>
                        <td class="px-3 py-2 text-xs">{{ $lead->quantyra_domain ?? '—' }}</td>
                        <td class="px-3 py-2 text-xs">{{ $payload['lead_score'] ?? '—' }}</td>
                        <td class="px-3 py-2 text-xs">
                            <span class="rounded-full bg-slate-800 px-2 py-1">{{ strtoupper((string) ($payload['status'] ?? 'new')) }}</span>
                        </td>
                        <td class="px-3 py-2 text-xs">{{ optional($lead->created_at)->diffForHumans() }}</td>
                    </tr>
                @empty
                    <tr>
                        <td colspan="8" class="px-3 py-8 text-center text-sm text-slate-400">No leads found for current filters.</td>
                    </tr>
                @endforelse
            </tbody>
        </table>
    </div>

    <div>
        {{ $leads->links() }}
    </div>
</div>

