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

    public function test_dashboard_controller_returns_expected_view_data_without_subscription(): void
    {
        $user = User::factory()->createOne();
        $request = Request::create('/dashboard', 'GET');
        $request->setUserResolver(static fn (): User => $user);

        $response = app(DashboardController::class)->__invoke($request);

        $this->assertInstanceOf(View::class, $response);
        $this->assertSame('dashboard.index', $response->name());
        $this->assertFalse($response->getData()['hasApiAccess']);
        $this->assertFalse($response->getData()['hasWidgetAccess']);
        $this->assertNull($response->getData()['subscription']);
        $this->assertSame(0, $response->getData()['apiOverageCount']);
        $this->assertIsIterable($response->getData()['apiTokens']);
        $this->assertFalse($response->getData()['leadsMetricAvailable']);
        $this->assertNull($response->getData()['leadsThisMonth']);
        $this->assertFalse($response->getData()['canPurchaseExtraDomainSlots']);
        $this->assertIsArray($response->getData()['widgetPaletteForm']);
    }
}
