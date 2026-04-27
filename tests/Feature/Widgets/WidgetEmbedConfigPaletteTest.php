<?php

namespace Tests\Feature\Widgets;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Ghl\Widgets\Models\GhlWidgetConfig;
use App\Models\Domain;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class WidgetEmbedConfigPaletteTest extends TestCase
{
    use RefreshDatabase;

    private function createGhlToken(string $companyId, string $locationId): GhlOAuthToken
    {
        $access = 'access-pal-'.uniqid('', true);
        $refresh = 'refresh-pal-'.uniqid('', true);

        $t = new GhlOAuthToken([
            'ghl_company_id' => $companyId,
            'ghl_location_id' => $locationId,
            'ghl_user_id' => 'u1',
            'user_type' => 'Location',
            'scopes' => 'contacts.write',
            'is_bulk_install' => false,
            'expires_at' => now()->addDay(),
            'status' => 'active',
            'access_token_hash' => hash('sha256', $access),
        ]);
        $t->access_token = $access;
        $t->refresh_token = $refresh;
        $t->save();

        return $t->fresh();
    }

    public function test_direct_site_config_reflects_user_widget_palette(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'widget_palette' => [
                'primary' => '#c0ffee',
                'secondary' => '#bada55',
                'text' => '#222222',
                'background' => '#fafafa',
                'theme' => 'dark',
            ],
        ]);
        $key = $user->ensureWidgetEmbedSiteKey();

        Domain::query()->create([
            'domain_slug' => 'embed-client.test',
            'is_active' => true,
        ]);

        $this->withHeaders([
            'Origin' => 'https://embed-client.test',
        ])->get('http://localhost/widget/config/'.$key)
            ->assertOk()
            ->assertJsonPath('primary_color', '#c0ffee')
            ->assertJsonPath('secondary_color', '#bada55')
            ->assertJsonPath('theme', 'dark')
            ->assertJsonPath('listings_per_page', 20);
    }

    public function test_linked_ghl_install_global_palette_overrides_row_primary_color(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'widget_palette' => [
                'primary' => '#00ff99',
                'secondary' => '#003322',
                'text' => '#111111',
                'background' => '#eeeeee',
                'theme' => 'light',
            ],
        ]);

        Domain::query()->create([
            'domain_slug' => 'customer-palette.example',
            'is_active' => true,
        ]);

        $oauth = $this->createGhlToken('co-pal', 'loc-pal');
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $oauth->id,
            'ghl_location_id' => 'loc-pal',
            'ghl_company_id' => 'co-pal',
            'primary_url' => 'https://customer-palette.example',
            'widget_api_key' => 'qh_palettekey1234567890123456789012gh',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
            'quantyra_user_id' => $user->id,
        ]);

        GhlWidgetConfig::query()->create([
            'ghl_location_id' => 'loc-pal',
            'ghl_registered_url_id' => $row->id,
            'primary_color' => '#000000',
            'secondary_color' => '#111111',
            'listings_per_page' => 9,
        ]);

        $this->withHeaders([
            'Origin' => 'https://customer-palette.example',
        ])->get('http://localhost/widget/config/'.$row->widget_api_key)
            ->assertOk()
            ->assertJsonPath('primary_color', '#00ff99')
            ->assertJsonPath('listings_per_page', 9);
    }
}
