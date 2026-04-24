<?php

namespace App\Livewire\Marketing;

use App\Billing\SubscriptionCatalog;
use Illuminate\Contracts\View\View;
use Livewire\Component;

class SalesLandingPage extends Component
{
    public bool $showLoginModal = false;

    public string $billingInterval = 'monthly';

    /** @var array<string, array<string, mixed>> */
    public array $plans = [];

    public int $teaserLeads = 0;

    /**
     * Illustrative hyper-local positioning (no live MLS or map data on this page).
     *
     * @var array<int, array<string, string>>
     */
    public array $illustrativeMarkets = [
        ['label' => 'Beach & intracoastal corridor', 'geography' => 'Pinellas County shoreline + municipal limits'],
        ['label' => 'Urban core listings', 'geography' => 'City of St. Petersburg + neighborhood overlays'],
        ['label' => 'Suburban move-up belt', 'geography' => 'Hillsborough County + selected city boundaries'],
        ['label' => 'Waterfront & canal niche', 'geography' => 'City of Clearwater + coastal zoning context'],
        ['label' => 'First-time buyer towns', 'geography' => 'County line to city limits for school-centric search'],
        ['label' => '55+ & lifestyle communities', 'geography' => 'City + census-style sub-areas (illustrative)'],
    ];

    /**
     * @var array<int, array<string, string>>
     */
    public array $faqs = [
        [
            'question' => 'Does this page show live MLS listings?',
            'answer' => 'No. This marketing page uses static and mock visuals only. Live IDX data is shown only on authorized domains under the Stellar MLS agreement.',
        ],
        [
            'question' => 'How are leads protected and routed?',
            'answer' => 'Visitors can preview a short set of teaser listings, then complete email and phone OTP verification. Verified leads are delivered only to the subscribed agent or lender responsible for that geography.',
        ],
        [
            'question' => 'Can I use this with GoHighLevel or LeadConnector?',
            'answer' => 'Yes. Subscribers can use both JS embed widgets and the embedded app workflow, including API options for custom experiences.',
        ],
        [
            'question' => 'How do county and city boundaries improve my IDX?',
            'answer' => 'Boundary-aware geography helps buyers search the way they think—by town, school area, and county—while keeping your brand centered on the markets you actually serve. On authorized IDX domains, maps and search can respect those geographic frames for clearer, more trustworthy discovery.',
        ],
    ];

    public function mount(SubscriptionCatalog $catalog): void
    {
        $this->plans = $catalog->plans();
        $this->teaserLeads = $catalog->teaserLeadsThisMonth();
    }

    public function setBillingInterval(string $interval): void
    {
        if (in_array($interval, ['monthly', 'annual'], true)) {
            $this->billingInterval = $interval;
        }
    }

    public function openLoginModal(): void
    {
        $this->showLoginModal = true;
    }

    public function closeLoginModal(): void
    {
        $this->showLoginModal = false;
    }

    public function render(): View
    {
        return view('livewire.marketing.sales-landing-page');
    }
}
