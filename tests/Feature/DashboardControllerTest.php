<?php

namespace Tests\Feature;

use App\Models\User;
use App\Services\Dashboard\DashboardViewData;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Request;
use Tests\TestCase;

class DashboardControllerTest extends TestCase
{
    use RefreshDatabase;

    public function test_dashboard_view_data_returns_expected_keys(): void
    {
        $user = User::factory()->createOne();
        $request = Request::create('/dashboard', 'GET');
        $request->setUserResolver(static fn (): User => $user);

        $data = app(DashboardViewData::class)->forRequest($request);

        $this->assertTrue($data['hasApiAccess']);
        $this->assertIsIterable($data['apiTokens']);
        $this->assertFalse($data['canPurchaseExtraDomainSlots']);
        $this->assertFalse($data['canInviteUsers']);
        $this->assertSame(0, $data['verifiedDomainCount']);
        $this->assertIsArray($data['mlsCatalogFeedCodes']);
        $this->assertSame('dashboard', $data['activePanel']);
    }

    public function test_dashboard_view_data_normalizes_unknown_panel(): void
    {
        $user = User::factory()->createOne();
        $request = Request::create('/dashboard?panel=unknown', 'GET');
        $request->setUserResolver(static fn (): User => $user);

        $data = app(DashboardViewData::class)->forRequest($request);

        $this->assertSame('dashboard', $data['activePanel']);
    }
}
