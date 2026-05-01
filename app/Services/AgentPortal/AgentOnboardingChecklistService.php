<?php

namespace App\Services\AgentPortal;

use App\Filament\Pages\AgentAlertsWorkspace;
use App\Filament\Pages\AgentContactsWorkspace;
use App\Filament\Pages\AgentMarketingWorkspace;
use App\Filament\Pages\AgentSearchWorkspace;
use App\Filament\Pages\AgentSettingsWorkspace;
use App\Ghl\Sync\Models\QuantyraLead;
use App\Models\AgentAlert;
use App\Models\AgentSearch;
use App\Models\AgentShareLink;
use App\Models\User;
use App\Services\SubscriberLeadScopeService;

final class AgentOnboardingChecklistService
{
    private const PANEL = 'dashboard';

    public function __construct(
        private readonly SubscriberFeedAccessService $feeds,
        private readonly SubscriberLeadScopeService $leads,
        private readonly FeatureFlagService $flags,
    ) {}

    /**
     * @return list<array{id: string, title: string, description: string, done: bool, blocked: bool, href: string, cta_label: string}>
     */
    public function checklistForUser(User $user): array
    {
        $scopes = $this->feeds->resolvedScopesForUser($user);
        $feedReady = $scopes !== [];

        $hasSearch = AgentSearch::query()->where('user_id', $user->id)->exists();
        $hasActiveAlert = AgentAlert::query()
            ->where('user_id', $user->id)
            ->where('status', 'active')
            ->exists();

        $locationIds = $this->leads->locationIdsForUser($user);
        $hasLead = $locationIds !== [] && QuantyraLead::query()
            ->whereIn('ghl_location_id', $locationIds)
            ->exists();

        $hasShareLink = AgentShareLink::query()->where('user_id', $user->id)->exists();

        $searchOn = $this->flags->isEnabled($user, 'search');
        $alertsOn = $this->flags->isEnabled($user, 'alerts');
        $contactsOn = $this->flags->isEnabled($user, 'contacts');
        $marketingOn = $this->flags->isEnabled($user, 'marketing');

        return [
            [
                'id' => 'mls_feed_access',
                'title' => 'Confirm MLS feed access',
                'description' => 'Resolved scopes (below) should list at least one active dataset — includes your account default when none is explicitly assigned.',
                'done' => $feedReady,
                'blocked' => false,
                'href' => $feedReady ? '' : url('/dashboard?panel=onboarding'),
                'cta_label' => $feedReady ? '' : 'Subscriber setup',
            ],
            [
                'id' => 'saved_search',
                'title' => 'Save a search',
                'description' => 'Persist filters and map geometry as a reusable saved search for alerts and share links.',
                'done' => $hasSearch,
                'blocked' => ! $searchOn,
                'href' => $searchOn ? AgentSearchWorkspace::getUrl(panel: self::PANEL) : AgentSettingsWorkspace::getUrl(panel: self::PANEL),
                'cta_label' => $searchOn ? 'Open Search' : 'Enable in Settings',
            ],
            [
                'id' => 'active_alert',
                'title' => 'Create an active alert',
                'description' => 'Turn saved criteria into listing, market, or home value alerts on a cadence.',
                'done' => $hasActiveAlert,
                'blocked' => ! $alertsOn,
                'href' => $alertsOn ? AgentAlertsWorkspace::getUrl(panel: self::PANEL) : AgentSettingsWorkspace::getUrl(panel: self::PANEL),
                'cta_label' => $alertsOn ? 'Open Alerts' : 'Enable in Settings',
            ],
            [
                'id' => 'first_contact',
                'title' => 'Review contacts',
                'description' => 'Open the contacts workspace to segment leads and attach alerts.',
                'done' => $hasLead,
                'blocked' => ! $contactsOn,
                'href' => $contactsOn ? AgentContactsWorkspace::getUrl(panel: self::PANEL) : AgentSettingsWorkspace::getUrl(panel: self::PANEL),
                'cta_label' => $contactsOn ? 'Open Contacts' : 'Enable in Settings',
            ],
            [
                'id' => 'share_link',
                'title' => 'Create a share link',
                'description' => 'Publish a tracked share or SEO landing link from Marketing.',
                'done' => $hasShareLink,
                'blocked' => ! $marketingOn,
                'href' => $marketingOn ? AgentMarketingWorkspace::getUrl(panel: self::PANEL) : AgentSettingsWorkspace::getUrl(panel: self::PANEL),
                'cta_label' => $marketingOn ? 'Open Marketing' : 'Enable in Settings',
            ],
        ];
    }
}
