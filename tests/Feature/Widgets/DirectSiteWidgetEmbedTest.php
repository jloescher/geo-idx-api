<?php

namespace Tests\Feature\Widgets;

use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class DirectSiteWidgetEmbedTest extends TestCase
{
    use RefreshDatabase;

    public function test_dashboard_widget_validate_accepts_subscriber_embed_site_key(): void
    {
        $user = User::factory()->createOne();
        $key = $user->ensureWidgetEmbedSiteKey();

        $this->actingAs($user);

        $this->postJson('http://localhost/dashboard/widget-validate', [
            'token' => $key,
            'hostname' => 'localhost',
            'requireFooter' => true,
        ])->assertOk()
            ->assertJsonPath('ok', true)
            ->assertJsonPath('locationId', 'direct-'.$user->id);
    }

    public function test_widget_config_accepts_direct_site_key_with_approved_domain_origin(): void
    {
        $user = User::factory()->createOne();
        $key = $user->ensureWidgetEmbedSiteKey();
        Domain::query()->create([
            'domain_slug' => 'embed-client.test',
            'is_active' => true,
        ]);

        $this->withHeaders([
            'Origin' => 'https://embed-client.test',
        ])->get('http://localhost/widget/config/'.$key)
            ->assertOk()
            ->assertJsonPath('location_id', 'direct-'.$user->id);
    }
}
