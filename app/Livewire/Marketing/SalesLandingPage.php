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
     * @var array<int, array<string, string>>
     */
    public array $proofChips = [
        [
            'label' => 'Built for multiple MLS environments',
        ],
        [
            'label' => 'Geography-first search experiences',
        ],
        [
            'label' => 'Embedded JS widgets for GHL funnels',
        ],
        [
            'label' => 'Demo sandbox trial before paid live activation',
        ],
    ];

    /**
     * @var array<int, array<string, string>>
     */
    public array $geographyHighlights = [
        [
            'title' => 'Boundary-aware search',
            'description' => 'Let buyers explore by neighborhoods, service areas, and local boundaries that match how they actually shop.',
        ],
        [
            'title' => 'Map-first relevance',
            'description' => 'Prioritize discovery by location context so lead intent is clearer before a form submission.',
        ],
        [
            'title' => 'Local funnel precision',
            'description' => 'Keep campaigns aligned to the exact geographies your team serves instead of broad generic pages.',
        ],
    ];

    /**
     * @var array<int, array<string, string>>
     */
    public array $marketIntelHighlights = [
        [
            'title' => 'Context-rich listing experiences',
            'description' => 'Layer market signals into discovery surfaces so buyers understand the story behind price movement.',
        ],
        [
            'title' => 'Portfolio-ready insights',
            'description' => 'Support investor-focused funnels with digestible market-intelligence overlays for faster qualification.',
        ],
        [
            'title' => 'Future-ready intelligence stack',
            'description' => 'Designed to expand with additional market overlays, including digital-asset-aware trend context.',
        ],
    ];

    /**
     * @var array<int, array<string, string>>
     */
    public array $workflowSteps = [
        [
            'title' => 'Start in demo sandbox',
            'description' => 'Launch your 14-day demo-data trial to test widget placement, forms, and messaging.',
        ],
        [
            'title' => 'Embed and route leads',
            'description' => 'Install JS widgets on approved domains and connect submissions to your GHL workflows.',
        ],
        [
            'title' => 'Activate paid live data',
            'description' => 'Move to a paid plan when ready to unlock live data on your production funnels.',
        ],
    ];

    /**
     * @var array<int, array<string, string>>
     */
    public array $comparisonRows = [
        [
            'category' => 'Best for',
            'pro' => 'Solo GHL operators',
            'smart' => 'Teams running multiple funnels',
        ],
        [
            'category' => 'Widget and domain scope',
            'pro' => 'Up to 3 domains',
            'smart' => 'Up to 5 domains with broader campaign coverage',
        ],
        [
            'category' => 'Geography intelligence',
            'pro' => 'Core geography-aware discovery',
            'smart' => 'Advanced geography-led search and lead capture',
        ],
        [
            'category' => 'Market-intelligence overlays',
            'pro' => 'Core market context blocks',
            'smart' => 'Extended market-intel overlays for team workflows',
        ],
        [
            'category' => 'Trial and live data policy',
            'pro' => '14-day demo-data trial, paid live activation',
            'smart' => '14-day demo-data trial, paid live activation',
        ],
    ];

    /**
     * @var array<int, array<string, string>>
     */
    public array $faqs = [
        [
            'question' => 'Does this page show live MLS listings?',
            'answer' => 'No. GeoIDX marketing pages use static and demo visuals only. Live data is available only after paid activation on approved domains.',
        ],
        [
            'question' => 'What does the 14-day trial include?',
            'answer' => 'The trial includes demo data so you can test widget placement, forms, and workflow routing. Live data is unlocked only after paid plan activation.',
        ],
        [
            'question' => 'How are leads protected and routed?',
            'answer' => 'Visitors can preview teaser experiences, then complete verification steps. Qualified leads are routed into your configured workflow paths.',
        ],
        [
            'question' => 'Can I use this with GoHighLevel or LeadConnector?',
            'answer' => 'Yes. GeoIDX is built for GHL-style embedded widget funnels and supports both self-serve and team deployment patterns.',
        ],
        [
            'question' => 'How does geography intelligence improve conversion?',
            'answer' => 'Geography-aware experiences help buyers discover inventory by relevant local boundaries, increasing listing relevance and improving lead quality before handoff.',
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
