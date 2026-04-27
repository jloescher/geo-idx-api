<?php

namespace App\Widgets;

use App\Models\Domain;
use App\Models\User;

/**
 * Runtime widget context for subscribers who embed without GoHighLevel (approved domains only).
 *
 * Exposes the same surface GHL-based middleware expects via {@see $ghl_location_id} and {@see allowedOrigins()}.
 */
final class DirectSiteWidgetContext
{
    public readonly string $ghl_location_id;

    public bool $widget_access_enabled = true;

    public function __construct(
        public User $user,
    ) {
        $this->ghl_location_id = 'direct-'.$user->id;
    }

    /**
     * @return list<string>
     */
    public function allowedOrigins(): array
    {
        $origins = [];
        foreach (
            Domain::query()
                ->where('user_id', $this->user->id)
                ->where('is_active', true)
                ->whereIn('verification_status', ['verified', 'verified_ghl'])
                ->orderBy('domain_slug')
                ->cursor() as $domain
        ) {
            $host = trim((string) $domain->domain_slug);
            if ($host === '') {
                continue;
            }
            $origins[] = 'https://'.$host;
            $origins[] = 'http://'.$host;
        }

        return array_values(array_unique($origins));
    }
}
