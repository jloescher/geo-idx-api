<?php

namespace App\Console\Commands;

use App\Models\LeadAlertSetting;
use App\Models\LeadSavedSearch;
use App\Notifications\LeadDigestNotification;
use Illuminate\Console\Command;

class SendLeadAlertsCommand extends Command
{
    protected $signature = 'leads:send-alerts';

    protected $description = 'Send lead alert notifications for saved searches';

    public function handle(): int
    {
        $sent = 0;

        LeadAlertSetting::query()
            ->where('enabled', true)
            ->with('user')
            ->chunkById(100, function ($settings) use (&$sent): void {
                foreach ($settings as $setting) {
                    $user = $setting->user;
                    if ($user === null) {
                        continue;
                    }

                    /** @var list<LeadSavedSearch> $searches */
                    $searches = LeadSavedSearch::query()
                        ->where('user_id', $user->id)
                        ->where('is_alert_enabled', true)
                        ->get()
                        ->all();

                    if ($searches === []) {
                        continue;
                    }

                    $user->notify(new LeadDigestNotification($searches, (string) $setting->frequency));
                    $setting->forceFill(['last_sent_at' => now()])->save();
                    $sent++;
                }
            });

        $this->info("Lead alert notifications sent: {$sent}");

        return self::SUCCESS;
    }
}
