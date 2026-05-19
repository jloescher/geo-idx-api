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
        $this->assertSame('setup', $data['activePanel']);
        $this->assertSame('register', $data['setupPhase']);
        $this->assertFalse($data['hasVerifiedDomain']);
        $this->assertNull($data['primaryVerifiedDomainSlug']);
        $this->assertFalse($data['hasProductionToken']);
        $this->assertFalse($data['hasStagingToken']);
        $this->assertFalse($data['canCreateStagingToken']);
    }

    public function test_dashboard_view_data_normalizes_unknown_panel(): void
    {
        $user = User::factory()->createOne();
        $request = Request::create('/dashboard?panel=unknown', 'GET');
        $request->setUserResolver(static fn (): User => $user);

        $data = app(DashboardViewData::class)->forRequest($request);

        $this->assertSame('setup', $data['activePanel']);
    }

    public function test_legacy_panels_resolve_to_setup(): void
    {
        $user = User::factory()->createOne();

        foreach (['dashboard', 'onboarding', 'domains'] as $legacy) {
            $request = Request::create('/dashboard?panel='.$legacy, 'GET');
            $request->setUserResolver(static fn (): User => $user);

            $data = app(DashboardViewData::class)->forRequest($request);
            $this->assertSame('setup', $data['activePanel'], "Legacy panel '{$legacy}' should resolve to 'setup'");
        }
    }

    public function test_api_panel_remains_valid(): void
    {
        $user = User::factory()->createOne();
        $request = Request::create('/dashboard?panel=api', 'GET');
        $request->setUserResolver(static fn (): User => $user);

        $data = app(DashboardViewData::class)->forRequest($request);

        $this->assertSame('api', $data['activePanel']);
    }
}
