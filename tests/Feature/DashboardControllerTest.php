<?php

namespace Tests\Feature;

use App\Http\Controllers\DashboardController;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Request;
use Illuminate\View\View;
use Tests\TestCase;

class DashboardControllerTest extends TestCase
{
    use RefreshDatabase;

    public function test_dashboard_controller_returns_expected_view_data(): void
    {
        $user = User::factory()->createOne();
        $request = Request::create('/dashboard', 'GET');
        $request->setUserResolver(static fn (): User => $user);

        $response = app(DashboardController::class)->__invoke($request);

        $this->assertInstanceOf(View::class, $response);
        $this->assertSame('dashboard.index', $response->name());
        $this->assertTrue($response->getData()['hasApiAccess']);
        $this->assertIsIterable($response->getData()['apiTokens']);
        $this->assertFalse($response->getData()['canPurchaseExtraDomainSlots']);
        $this->assertFalse($response->getData()['canInviteUsers']);
        $this->assertSame(0, $response->getData()['verifiedDomainCount']);
        $this->assertIsArray($response->getData()['mlsCatalogFeedCodes']);
    }
}
