<?php

use App\Ghl\Sync\Models\QuantyraLead;
use App\Services\SubscriberLeadScopeService;
use Illuminate\Database\Eloquent\Builder;
use Livewire\Volt\Component;
use Livewire\WithPagination;

new class extends Component {
    use WithPagination;

    public string $tab = 'all';
    public string $search = '';
    public string $status = '';
    public string $domain = '';
    public string $stage = '';
    public string $activity = '';
    public string $dateFrom = '';
    public string $dateTo = '';
    public array $selected = [];

    protected $queryString = [
        'tab' => ['except' => 'all'],
        'search' => ['except' => ''],
        'status' => ['except' => ''],
        'domain' => ['except' => ''],
        'stage' => ['except' => ''],
        'activity' => ['except' => ''],
        'dateFrom' => ['except' => ''],
        'dateTo' => ['except' => ''],
    ];

    public function updatingTab(): void { $this->resetPage(); }
    public function updatingSearch(): void { $this->resetPage(); }
    public function updatingStatus(): void { $this->resetPage(); }
    public function updatingDomain(): void { $this->resetPage(); }
    public function updatingStage(): void { $this->resetPage(); }
    public function updatingActivity(): void { $this->resetPage(); }
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
        if ($this->tab === 'smart') {
            $query->where(function (Builder $smart): void {
                $smart->whereRaw("coalesce((payload->>'lead_score')::int, 0) >= 80")
                    ->orWhere('payload->status', 'hot');
            });
        }

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
        if ($this->stage !== '') {
            $query->whereRaw('LOWER(payload->>\'stage\') = ?', [strtolower($this->stage)]);
        }
        if ($this->activity === 'recent') {
            $query->where('created_at', '>=', now()->subDays(7));
        }
        if ($this->activity === 'stale') {
            $query->where('created_at', '<', now()->subDays(30));
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

<div class="mt-4 space-y-4">
    <div class="idx-leads-tabs">
        <button type="button" wire:click="$set('tab', 'all')" class="idx-leads-tab {{ $tab === 'all' ? 'is-active' : '' }}">All Leads</button>
        <button type="button" wire:click="$set('tab', 'smart')" class="idx-leads-tab {{ $tab === 'smart' ? 'is-active' : '' }}">Smart Lists</button>
        <button type="button" wire:click="$set('tab', 'domain')" class="idx-leads-tab {{ $tab === 'domain' ? 'is-active' : '' }}">Filter by Domain</button>
        <button type="button" wire:click="$set('tab', 'stage')" class="idx-leads-tab {{ $tab === 'stage' ? 'is-active' : '' }}">Filter by Stage</button>
    </div>

    <div class="idx-toolbar grid gap-3 p-3 md:grid-cols-12">
        <input type="text" wire:model.live.debounce.300ms="search" placeholder="Search leads..." class="idx-input px-3 py-2 text-sm md:col-span-5">
        <select wire:model.live="status" class="idx-input px-3 py-2 text-sm md:col-span-2">
            <option value="">All statuses</option>
            <option value="new">New</option>
            <option value="hot">Hot</option>
            <option value="contacted">Contacted</option>
            <option value="converted">Converted</option>
        </select>
        <select wire:model.live="domain" class="idx-input px-3 py-2 text-sm md:col-span-2">
            <option value="">All domains</option>
            @foreach ($domains as $domainOpt)
                <option value="{{ $domainOpt }}">{{ $domainOpt }}</option>
            @endforeach
        </select>
        <input type="date" wire:model.live="dateFrom" class="idx-input px-2 py-2 text-xs md:col-span-1">
        <input type="date" wire:model.live="dateTo" class="idx-input px-2 py-2 text-xs md:col-span-1">
        <select wire:model.live="stage" class="idx-input px-3 py-2 text-sm md:col-span-1">
            <option value="">Stage</option>
            <option value="new">New</option>
            <option value="qualified">Qualified</option>
            <option value="nurture">Nurture</option>
            <option value="closed">Closed</option>
        </select>
    </div>

    <div class="flex flex-wrap items-center gap-2">
        <button type="button" wire:click="markSelectedContacted" class="idx-btn-primary px-3 py-2 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-50" @disabled($selected === [])>Mark Contacted ▾</button>
        <button type="button" wire:click="deleteSelected" class="idx-btn-danger px-3 py-2 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-50" @disabled($selected === [])>Delete</button>
        <select wire:model.live="activity" class="idx-input ms-auto px-3 py-2 text-xs">
            <option value="">Filter Activity</option>
            <option value="recent">Recent (7d)</option>
            <option value="stale">Stale (30d+)</option>
        </select>
    </div>

    <div class="idx-table-shell overflow-x-auto">
        <table class="min-w-full text-sm">
            <thead class="idx-leads-table-head border-b border-white/10 bg-slate-950/70">
                <tr>
                    <th>SELECT</th>
                    <th>Name</th>
                    <th>Email</th>
                    <th>Last Activity</th>
                    <th>Domain</th>
                    <th>Score</th>
                    <th>Status</th>
                    <th>Created</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-white/5 bg-slate-900/60 text-slate-200">
                @forelse ($leads as $lead)
                    @php($payload = is_array($lead->payload) ? $lead->payload : [])
                    @php($leadName = $payload['name'] ?? trim(($payload['first_name'] ?? '').' '.($payload['last_name'] ?? '')) ?: 'Lead #'.$lead->id)
                    @php($initials = strtoupper(substr(preg_replace('/[^A-Za-z]/', '', (string) $leadName), 0, 2)))
                    @php($avatarClass = match($lead->id % 4) {0 => 'idx-avatar-red', 1 => 'idx-avatar-blue', 2 => 'idx-avatar-amber', default => 'idx-avatar-indigo'})
                    <tr class="idx-leads-row">
                        <td class="px-4 py-3">
                            <input type="checkbox" value="{{ $lead->id }}" wire:model.live="selected" class="rounded border-white/30 bg-slate-950">
                        </td>
                        <td class="px-4 py-3">
                            <div class="flex items-center gap-3">
                                <span class="idx-avatar {{ $avatarClass }}">{{ $initials !== '' ? $initials : 'LD' }}</span>
                                <div>
                                    <p class="font-semibold text-slate-100">{{ $leadName }}</p>
                                    <p class="text-xs text-slate-400">{{ $payload['phone'] ?? 'No phone' }}</p>
                                </div>
                            </div>
                        </td>
                        <td class="px-4 py-3 text-xs">{{ $payload['email'] ?? '—' }}</td>
                        <td class="px-4 py-3 text-xs">{{ $payload['property_interest'] ?? $lead->listing_id ?? '—' }}</td>
                        <td class="px-4 py-3 text-xs">{{ $lead->quantyra_domain ?? '—' }}</td>
                        <td class="px-4 py-3 text-xs font-semibold">{{ $payload['lead_score'] ?? '—' }}</td>
                        <td class="px-4 py-3 text-xs">
                            <span class="rounded-full border border-white/10 bg-slate-950/70 px-2 py-1">{{ strtoupper((string) ($payload['status'] ?? 'new')) }}</span>
                        </td>
                        <td class="px-4 py-3 text-xs">{{ optional($lead->created_at)->diffForHumans() }}</td>
                    </tr>
                @empty
                    <tr>
                        <td colspan="8" class="px-4 py-12 text-center text-sm text-slate-400">No leads found for current filters.</td>
                    </tr>
                @endforelse
            </tbody>
        </table>
    </div>

    <div>
        {{ $leads->links() }}
    </div>
</div>

