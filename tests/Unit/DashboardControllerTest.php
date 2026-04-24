<?php

namespace Tests\Unit;

use App\Http\Controllers\DashboardController;
use Illuminate\Http\Request;
use Illuminate\View\View;
use Tests\TestCase;

class DashboardControllerTest extends TestCase
{
    public function test_dashboard_controller_returns_expected_view_data_without_subscription(): void
    {
        $request = Request::create('/dashboard', 'GET');
        $request->setUserResolver(static fn (): object => new class
        {
            public function subscription(string $name): mixed
            {
                return null;
            }
        });

        $response = app(DashboardController::class)->__invoke($request);

        $this->assertInstanceOf(View::class, $response);
        $this->assertSame('dashboard.index', $response->name());
        $this->assertFalse($response->getData()['hasApiAccess']);
        $this->assertFalse($response->getData()['hasWidgetAccess']);
        $this->assertNull($response->getData()['subscription']);
        $this->assertSame(0, $response->getData()['apiOverageCount']);
        $this->assertIsIterable($response->getData()['apiTokens']);
    }
}
