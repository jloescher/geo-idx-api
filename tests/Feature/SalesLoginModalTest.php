<?php

namespace Tests\Feature;

use Livewire\Livewire;
use Tests\TestCase;

class SalesLoginModalTest extends TestCase
{
    public function test_subscriber_login_modal_can_open_and_close(): void
    {
        Livewire::test('marketing.sales-landing-page')
            ->assertSet('showLoginModal', false)
            ->call('openLoginModal')
            ->assertSet('showLoginModal', true)
            ->call('closeLoginModal')
            ->assertSet('showLoginModal', false);
    }
}
