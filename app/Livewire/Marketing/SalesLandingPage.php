<?php

namespace App\Livewire\Marketing;

use Illuminate\Contracts\View\View;
use Livewire\Component;

class SalesLandingPage extends Component
{
    /**
     * @var array<int, array<string, string>>
     */
    public array $pilotDomains = [
        ['domain' => 'clearwaterflhouses.com', 'market' => 'Clearwater'],
        ['domain' => 'stpetersburghomesnow.com', 'market' => 'St. Petersburg'],
        ['domain' => 'searchtampabayhouses.com', 'market' => 'Tampa Bay'],
        ['domain' => 'dunedinhomesmarket.com', 'market' => 'Dunedin'],
        ['domain' => 'largoareahomes.com', 'market' => 'Largo'],
        ['domain' => 'seminoleflhomes.com', 'market' => 'Seminole'],
        ['domain' => 'palmharborhouses.com', 'market' => 'Palm Harbor'],
        ['domain' => 'safetyharborproperty.com', 'market' => 'Safety Harbor'],
        ['domain' => 'tarponspringslistings.com', 'market' => 'Tarpon Springs'],
        ['domain' => 'oldsmarhomeshub.com', 'market' => 'Oldsmar'],
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
            'answer' => 'Visitors can preview up to 3 teaser listings, then complete email and phone OTP verification. Verified leads are delivered only to subscribed agents/lenders.',
        ],
        [
            'question' => 'Can I use this with GoHighLevel or LeadConnector?',
            'answer' => 'Yes. Subscribers can use both JS embed widgets and the embedded app workflow, including API options for custom experiences.',
        ],
    ];

    public function render(): View
    {
        return view('livewire.marketing.sales-landing-page');
    }
}
