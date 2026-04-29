<?php

use App\Models\LeadAlertSetting;
use App\Notifications\LeadDigestNotification;
use Livewire\Volt\Component;

new class extends Component {
    public bool $enabled = false;
    public string $frequency = 'instant';
    public string $ruleDomain = '';
    public string $ruleMinScore = '';

    public function mount(): void
    {
        $user = auth()->user();
        if ($user === null) {
            return;
        }

        $setting = LeadAlertSetting::query()->firstOrCreate(
            ['user_id' => $user->id],
            ['enabled' => false, 'frequency' => 'instant', 'rules' => []],
        );

        $this->enabled = (bool) $setting->enabled;
        $this->frequency = (string) $setting->frequency;
        $this->ruleDomain = (string) data_get($setting->rules, 'domain', '');
        $this->ruleMinScore = (string) data_get($setting->rules, 'min_score', '');
    }

    public function save(): void
    {
        $user = auth()->user();
        if ($user === null) {
            return;
        }

        LeadAlertSetting::query()->updateOrCreate(
            ['user_id' => $user->id],
            [
                'enabled' => $this->enabled,
                'frequency' => $this->frequency,
                'rules' => [
                    'domain' => trim($this->ruleDomain),
                    'min_score' => trim($this->ruleMinScore),
                ],
            ],
        );

        session()->flash('dashboard_status', 'Lead alert settings saved.');
    }

    public function sendTestEmail(): void
    {
        $user = auth()->user();
        if ($user === null) {
            return;
        }

        $user->notify(new LeadDigestNotification([], 'test'));
        session()->flash('dashboard_status', 'Test alert email sent.');
    }
}; ?>

<div class="idx-card p-4">
    <h3 class="text-sm font-semibold text-white">Email Alerts</h3>
    <p class="mt-1 text-xs text-slate-400">Configure real-time or digest alerts for new matching leads.</p>

    <div class="mt-3 space-y-3">
        <label class="flex items-center gap-2 text-sm text-slate-200">
            <input type="checkbox" wire:model="enabled" class="rounded border-white/30 bg-slate-950">
            Receive lead alerts
        </label>

        <label class="block text-xs text-slate-400">
            Frequency
            <select wire:model="frequency" class="idx-input mt-1 w-full px-3 py-2 text-xs">
                <option value="instant">Instant</option>
                <option value="daily">Daily digest</option>
                <option value="weekly">Weekly</option>
            </select>
        </label>

        <label class="block text-xs text-slate-400">
            Rule: domain
            <input type="text" wire:model="ruleDomain" placeholder="example.com" class="idx-input mt-1 w-full px-3 py-2 text-xs">
        </label>

        <label class="block text-xs text-slate-400">
            Rule: min score
            <input type="number" wire:model="ruleMinScore" placeholder="80" class="idx-input mt-1 w-full px-3 py-2 text-xs">
        </label>

        <div class="flex gap-2">
            <button type="button" wire:click="save" class="idx-btn-primary px-3 py-2 text-xs font-semibold">Save alerts</button>
            <button type="button" wire:click="sendTestEmail" class="idx-btn-secondary px-3 py-2 text-xs font-semibold">Test email</button>
        </div>
        <p class="text-[11px] text-slate-500">Compliance: alert emails include unsubscribe support and respect notification controls.</p>
    </div>
</div>

