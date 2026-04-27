<?php

namespace App\Notifications;

use App\Models\LeadSavedSearch;
use Illuminate\Bus\Queueable;
use Illuminate\Notifications\Messages\MailMessage;
use Illuminate\Notifications\Notification;

class LeadDigestNotification extends Notification
{
    use Queueable;

    /**
     * @param  list<LeadSavedSearch>  $searches
     */
    public function __construct(
        private readonly array $searches,
        private readonly string $frequency,
    ) {}

    /**
     * @return list<string>
     */
    public function via(object $notifiable): array
    {
        return ['mail'];
    }

    public function toMail(object $notifiable): MailMessage
    {
        $message = (new MailMessage)
            ->subject('GeoIDX lead alerts')
            ->greeting('Lead alerts are active')
            ->line('Frequency: '.strtoupper($this->frequency))
            ->line('Saved searches included: '.count($this->searches));

        foreach ($this->searches as $search) {
            $message->line('- '.$search->name);
        }

        return $message
            ->line('You can disable alerts from your dashboard settings.')
            ->line('All alert emails include unsubscribe support and respect your notification settings.');
    }
}
