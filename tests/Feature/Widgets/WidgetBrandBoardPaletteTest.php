<?php

namespace Tests\Feature\Widgets;

use App\Ghl\OAuth\Models\GhlOAuthToken;
use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Ghl\Widgets\Models\GhlWidgetConfig;
use App\Models\Domain;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class WidgetBrandBoardPaletteTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();
        Cache::flush();
    }

    private function createActiveToken(string $locationId = 'loc-bb'): GhlOAuthToken
    {
        $access = 'access-bb-'.uniqid('', true);
        $refresh = 'refresh-bb-'.uniqid('', true);

        $t = new GhlOAuthToken([
            'ghl_company_id' => 'co-bb',
            'ghl_location_id' => $locationId,
            'ghl_user_id' => 'u1',
            'user_type' => 'Location',
            'scopes' => 'brand-boards/design-kit.readonly',
            'is_bulk_install' => false,
            'expires_at' => now()->addHour(),
            'status' => 'active',
            'access_token_hash' => hash('sha256', $access),
        ]);
        $t->access_token = $access;
        $t->refresh_token = $refresh;
        $t->save();

        return $t->fresh();
    }

    public function test_widget_config_merges_brand_board_colors_after_widget_config_row(): void
    {
        Http::fake(function (Request $request) {
            $url = $request->url();
            if (str_contains($url, '/brand-boards/loc-bb/bb-detail-1')) {
                return Http::response([
                    '_id' => 'bb-detail-1',
                    'name' => 'Main',
                    'default' => true,
                    'colors' => [
                        ['hex' => '#AABBCC', 'label' => 'Primary Brand'],
                        ['hex' => '#223344', 'label' => 'Secondary'],
                        ['hex' => '#FFEEDD', 'label' => 'Accent'],
                    ],
                    'fonts' => [
                        ['font' => 'Montserrat', 'fallback' => 'sans-serif', 'label' => 'Heading'],
                    ],
                ], 200);
            }
            if (str_contains($url, '/brand-boards/loc-bb') && ! preg_match('#/brand-boards/loc-bb/[^/]+$#', (string) parse_url($url, PHP_URL_PATH))) {
                return Http::response([
                    'brandBoards' => [
                        [
                            '_id' => 'bb-detail-1',
                            'name' => 'Main',
                            'default' => true,
                            'updatedAt' => '2024-01-01T00:00:00.000Z',
                        ],
                    ],
                    'totalCount' => 1,
                ], 200);
            }

            return Http::response(['message' => 'unexpected'], 500);
        });

        Domain::query()->create([
            'domain_slug' => 'brand-board.example',
            'is_active' => true,
        ]);

        $oauth = $this->createActiveToken('loc-bb');
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $oauth->id,
            'ghl_location_id' => 'loc-bb',
            'ghl_company_id' => 'co-bb',
            'primary_url' => 'https://brand-board.example',
            'widget_api_key' => 'qh_bbkey123456789012345678901234ab',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);

        GhlWidgetConfig::query()->create([
            'ghl_location_id' => 'loc-bb',
            'ghl_registered_url_id' => $row->id,
            'primary_color' => '#000000',
            'secondary_color' => '#111111',
        ]);

        $this->withHeaders([
            'Origin' => 'https://brand-board.example',
        ])->get('http://localhost/widget/config/'.$row->widget_api_key)
            ->assertOk()
            ->assertJsonPath('primary_color', '#aabbcc')
            ->assertJsonPath('secondary_color', '#223344')
            ->assertJsonPath('accent_color', '#ffeedd')
            ->assertJsonPath('font_family', 'Montserrat, sans-serif');

        Http::assertSentCount(2);
    }

    public function test_widget_config_survives_brand_board_list_forbidden(): void
    {
        Http::fake([
            '*' => Http::response(['message' => 'Forbidden'], 403),
        ]);

        Domain::query()->create([
            'domain_slug' => 'bb-forbidden.example',
            'is_active' => true,
        ]);

        $oauth = $this->createActiveToken('loc-bb-403');
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $oauth->id,
            'ghl_location_id' => 'loc-bb-403',
            'ghl_company_id' => 'co-bb',
            'primary_url' => 'https://bb-forbidden.example',
            'widget_api_key' => 'qh_bbkey403456789012345678901234cd',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);

        GhlWidgetConfig::query()->create([
            'ghl_location_id' => 'loc-bb-403',
            'ghl_registered_url_id' => $row->id,
            'primary_color' => '#123456',
            'secondary_color' => '#654321',
        ]);

        $this->withHeaders([
            'Origin' => 'https://bb-forbidden.example',
        ])->get('http://localhost/widget/config/'.$row->widget_api_key)
            ->assertOk()
            ->assertJsonPath('primary_color', '#123456')
            ->assertJsonPath('secondary_color', '#654321');

        Http::assertSentCount(1);
    }

    public function test_brand_board_fetch_is_cached_between_widget_config_requests(): void
    {
        Http::fake(function (Request $request) {
            $url = $request->url();
            if (str_contains($url, '/brand-boards/loc-bb-cache/cache-board')) {
                return Http::response([
                    '_id' => 'cache-board',
                    'colors' => [
                        ['hex' => '#CAFEC0', 'label' => 'Primary'],
                    ],
                    'fonts' => [],
                ], 200);
            }
            if (str_contains($url, '/brand-boards/loc-bb-cache')) {
                return Http::response([
                    'brandBoards' => [
                        [
                            '_id' => 'cache-board',
                            'name' => 'Cached',
                            'default' => true,
                        ],
                    ],
                    'totalCount' => 1,
                ], 200);
            }

            return Http::response(['message' => 'unexpected'], 500);
        });

        Domain::query()->create([
            'domain_slug' => 'bb-cache.example',
            'is_active' => true,
        ]);

        $oauth = $this->createActiveToken('loc-bb-cache');
        $row = GhlRegisteredUrl::query()->create([
            'ghl_oauth_token_id' => $oauth->id,
            'ghl_location_id' => 'loc-bb-cache',
            'ghl_company_id' => 'co-bb',
            'primary_url' => 'https://bb-cache.example',
            'widget_api_key' => 'qh_bbkeycache789012345678901234ef',
            'integration_type' => 'external_website',
            'mls_agreement_acknowledged' => true,
            'urls_validated' => true,
            'widget_access_enabled' => true,
        ]);

        GhlWidgetConfig::query()->create([
            'ghl_location_id' => 'loc-bb-cache',
            'ghl_registered_url_id' => $row->id,
            'primary_color' => '#000000',
        ]);

        $headers = ['Origin' => 'https://bb-cache.example'];
        $this->withHeaders($headers)->get('http://localhost/widget/config/'.$row->widget_api_key)->assertOk();
        $this->withHeaders($headers)->get('http://localhost/widget/config/'.$row->widget_api_key)->assertOk();

        Http::assertSentCount(2);
    }
}
