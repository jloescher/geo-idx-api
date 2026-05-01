<?php

namespace Tests\Feature;

use App\Ghl\Sync\Models\QuantyraLead;
use App\Models\AgentActivityEvent;
use App\Models\AgentAlert;
use App\Models\AgentAlertRun;
use App\Models\AgentAlertTemplate;
use App\Models\AgentPortalSetting;
use App\Models\AgentSearch;
use App\Models\AgentSearchFilter;
use App\Models\AgentSeoLandingPage;
use App\Models\AgentShareLink;
use App\Models\MlsFieldCatalog;
use App\Models\SubscriberFeedAccess;
use App\Models\User;
use App\Services\AgentPortal\AlertSchedulerService;
use App\Services\AgentPortal\FeatureFlagService;
use App\Services\AgentPortal\FieldCatalogService;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use Carbon\CarbonImmutable;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Str;
use Tests\TestCase;

class AgentPortalCrudTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();
        $this->withoutVite();
    }

    public function test_user_can_create_and_execute_agent_search(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches', [
            'name' => 'Waterfront buyers',
            'is_template' => false,
            'filters' => [
                [
                    'canonical_field_key' => 'property.list_price',
                    'operator' => 'gte',
                    'value_json' => 500000,
                ],
            ],
        ]);

        $response->assertCreated();
        $searchId = (int) $response->json('data.id');
        $this->assertGreaterThan(0, $searchId);

        $execute = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'agent_search_id' => $searchId,
        ]);

        $execute->assertOk();
        $execute->assertJsonPath('meta.compiled_query_count', 1);
        $execute->assertJsonStructure(['data' => ['meta' => ['by_source']]]);
        $this->assertDatabaseHas('agent_activity_events', [
            'user_id' => $user->id,
            'event_type' => 'search_execute',
        ]);

        $event = AgentActivityEvent::query()
            ->where('user_id', $user->id)
            ->where('event_type', 'search_execute')
            ->latest('id')
            ->first();
        $this->assertNotNull($event);
        $execMeta = is_array($event->metadata_json) ? $event->metadata_json : [];
        $this->assertSame('saved_loaded', $execMeta['trigger'] ?? null);
        $this->assertSame($searchId, $execMeta['agent_search_id'] ?? null);
    }

    public function test_agent_searches_geocode_returns_json_and_sends_nominatim_user_agent(): void
    {
        config([
            'geocoding.nominatim_user_agent' => 'QuantyraIDXTest/1.0 (+https://test.example)',
        ]);

        Http::fake([
            'nominatim.openstreetmap.org/*' => Http::response([
                ['lat' => '27.946', 'lon' => '-82.458', 'display_name' => 'Tampa, FL'],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/searches/geocode?q='.urlencode('Tampa'));

        $response->assertOk();
        $response->assertJsonPath('data.0.display_name', 'Tampa, FL');
        $response->assertJsonPath('meta.cached', false);

        Http::assertSent(function (Request $request): bool {
            if (! str_contains($request->url(), 'nominatim.openstreetmap.org/search')) {
                return false;
            }
            $ua = $request->header('User-Agent');

            return is_array($ua) && ($ua[0] ?? null) === 'QuantyraIDXTest/1.0 (+https://test.example)';
        });
    }

    public function test_agent_searches_geocode_validates_empty_query(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $this->actingAs($user)->getJson('https://localhost/agent/searches/geocode')->assertStatus(422);
    }

    public function test_agent_searches_geocode_reuses_cache_without_second_upstream_request(): void
    {
        config([
            'geocoding.nominatim_user_agent' => 'QuantyraIDXTest/1.0 (+https://test.example)',
            'geocoding.geocode_cache_ttl_seconds' => 300,
        ]);

        Http::fake([
            'nominatim.openstreetmap.org/*' => Http::response([
                ['lat' => '27.1', 'lon' => '-82.2', 'display_name' => 'Sarasota, FL'],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();

        $first = $this->actingAs($user)->getJson('https://localhost/agent/searches/geocode?q='.urlencode('Sarasota'));
        $first->assertOk();
        $first->assertJsonPath('meta.cached', false);

        $second = $this->actingAs($user)->getJson('https://localhost/agent/searches/geocode?q='.urlencode('Sarasota'));
        $second->assertOk();
        $second->assertJsonPath('meta.cached', true);
        $second->assertJsonPath('data.0.display_name', 'Sarasota, FL');

        Http::assertSentCount(1);
    }

    public function test_user_can_create_and_delete_alert(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Starter search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $create = $this->actingAs($user)->postJson('https://localhost/agent/alerts', [
            'agent_search_id' => $search->id,
            'name' => 'Morning listing alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
        ]);

        $create->assertCreated();
        $alertId = (int) $create->json('data.id');
        $this->assertGreaterThan(0, $alertId);
        $this->assertDatabaseHas('agent_alerts', ['id' => $alertId, 'user_id' => $user->id]);

        $delete = $this->actingAs($user)->deleteJson('https://localhost/agent/alerts/'.$alertId);
        $delete->assertNoContent();
        $this->assertDatabaseMissing('agent_alerts', ['id' => $alertId]);
    }

    public function test_user_can_pause_and_resume_alert(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Status test alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
        ]);

        $pause = $this->actingAs($user)->putJson('https://localhost/agent/alerts/'.$alert->id, [
            'name' => 'Status test alert',
            'alert_type' => 'listing',
            'status' => 'paused',
            'schedule_json' => ['cadence' => 'daily'],
        ]);
        $pause->assertOk();
        $this->assertDatabaseHas('agent_alerts', ['id' => $alert->id, 'status' => 'paused']);

        $resume = $this->actingAs($user)->putJson('https://localhost/agent/alerts/'.$alert->id, [
            'name' => 'Status test alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
        ]);
        $resume->assertOk();
        $this->assertDatabaseHas('agent_alerts', ['id' => $alert->id, 'status' => 'active']);
    }

    public function test_user_cannot_access_another_users_search_or_alert(): void
    {
        /** @var User $owner */
        $owner = User::factory()->createOne();
        /** @var User $other */
        $other = User::factory()->createOne();

        $search = AgentSearch::query()->create([
            'user_id' => $owner->id,
            'name' => 'Owner search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $alert = AgentAlert::query()->create([
            'user_id' => $owner->id,
            'agent_search_id' => $search->id,
            'name' => 'Owner alert',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);

        $this->actingAs($other)
            ->getJson('https://localhost/agent/searches/'.$search->id)
            ->assertNotFound();

        $this->actingAs($other)
            ->getJson('https://localhost/agent/alerts/'.$alert->id)
            ->assertNotFound();

        $this->actingAs($other)
            ->getJson('https://localhost/agent/alerts/'.$alert->id.'/runs')
            ->assertNotFound();
    }

    public function test_alert_runs_endpoint_returns_ordered_runs_without_listing_ids_snapshot(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Runs API alert',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);

        AgentAlertRun::query()->create([
            'agent_alert_id' => $alert->id,
            'status' => 'sent',
            'metadata_json' => [
                'listing_query' => [
                    'success' => true,
                    'listing_ids_snapshot' => ['a', 'b'],
                    'new_match_count' => 2,
                ],
            ],
            'ran_at' => now()->subHours(2),
        ]);
        AgentAlertRun::query()->create([
            'agent_alert_id' => $alert->id,
            'status' => 'skipped',
            'metadata_json' => [
                'listing_query' => [
                    'success' => false,
                    'listing_ids_snapshot' => ['x'],
                    'reason' => 'empty_criteria',
                ],
            ],
            'ran_at' => now()->subMinute(),
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/alerts/'.$alert->id.'/runs?limit=10');
        $response->assertOk();
        $response->assertJsonCount(2, 'data');
        $newest = $response->json('data.0.metadata.listing_query');
        $this->assertIsArray($newest);
        $this->assertArrayNotHasKey('listing_ids_snapshot', $newest);
        $this->assertFalse($newest['success']);
        $this->assertSame('skipped', $response->json('data.0.status'));
    }

    public function test_alert_runs_limit_query_is_clamped_between_1_and_50(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Limit clamp alert',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);

        for ($i = 0; $i < 3; $i++) {
            AgentAlertRun::query()->create([
                'agent_alert_id' => $alert->id,
                'status' => 'sent',
                'metadata_json' => [],
                'ran_at' => now()->subMinutes($i),
            ]);
        }

        $this->actingAs($user)
            ->getJson('https://localhost/agent/alerts/'.$alert->id.'/runs?limit=0')
            ->assertOk()
            ->assertJsonCount(1, 'data');

        $this->actingAs($user)
            ->getJson('https://localhost/agent/alerts/'.$alert->id.'/runs?limit=999')
            ->assertOk()
            ->assertJsonCount(3, 'data');
    }

    public function test_alerts_index_includes_total_runs_count_and_caps_embedded_runs(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Many runs alert',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);

        for ($i = 0; $i < 12; $i++) {
            AgentAlertRun::query()->create([
                'agent_alert_id' => $alert->id,
                'status' => 'sent',
                'metadata_json' => [],
                'ran_at' => now()->subMinutes($i),
            ]);
        }

        $response = $this->actingAs($user)->getJson('https://localhost/agent/alerts');
        $response->assertOk();
        $row = collect($response->json('data'))->firstWhere('id', $alert->id);
        $this->assertIsArray($row);
        $this->assertSame(12, $row['runs_count']);
        $this->assertCount(8, $row['runs']);
    }

    public function test_lookup_options_returns_cached_or_live_values_for_user_scope(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.dataset' => 'stellar',
            'bridge.lookups_cache_ttl_seconds' => 3600,
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['LookupName' => 'PropertySubType', 'LookupValue' => 'Condominium'],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/searches/lookups/options?field=PropertySubType');

        $response->assertOk();
        $response->assertJsonPath('data.0.dataset_code', 'stellar');
        $response->assertJsonPath('data.0.values.0.LookupValue', 'Condominium');
    }

    public function test_lookup_options_without_field_returns_field_catalog_entries(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.dataset' => 'stellar',
            'bridge.lookups_cache_ttl_seconds' => 3600,
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['LookupName' => 'ListPrice', 'LookupValue' => 'List Price'],
                    ['LookupName' => 'BedroomsTotal', 'LookupValue' => 'Bedrooms Total'],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/searches/lookups/options');

        $response->assertOk();
        $response->assertJsonPath('data.0.dataset_code', 'stellar');
        $response->assertJsonPath('data.0.values.0.LookupName', 'ListPrice');
    }

    public function test_execute_supports_request_geometries(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    [
                        'ListingKey' => 'stellar:inside',
                        'StandardStatus' => 'Active',
                        'ListPrice' => 450000,
                        'Coordinates' => ['coordinates' => [-82.45, 27.95]],
                    ],
                    [
                        'ListingKey' => 'stellar:outside',
                        'StandardStatus' => 'Active',
                        'ListPrice' => 470000,
                        'Coordinates' => ['coordinates' => [-81.20, 28.40]],
                    ],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 100000],
            ],
            'geometries' => [
                [
                    'geometry_type' => 'polygon',
                    'mode' => 'include',
                    'geojson' => [
                        'coordinates' => [[
                            [-82.80, 27.70],
                            [-82.10, 27.70],
                            [-82.10, 28.20],
                            [-82.80, 28.20],
                            [-82.80, 27.70],
                        ]],
                    ],
                ],
            ],
        ]);

        $response->assertOk();
        $response->assertJsonPath('data.meta.total_items', 1);
        $this->assertDatabaseHas('agent_activity_events', [
            'user_id' => $user->id,
            'event_type' => 'search_execute',
        ]);

        $event = AgentActivityEvent::query()
            ->where('user_id', $user->id)
            ->where('event_type', 'search_execute')
            ->latest('id')
            ->first();
        $this->assertNotNull($event);
        $meta = is_array($event->metadata_json) ? $event->metadata_json : [];
        $this->assertSame(1, (int) ($meta['geometry_count'] ?? 0));
        $this->assertSame('manual', (string) ($meta['trigger'] ?? ''));
        $this->assertSame('workspace', (string) ($meta['surface'] ?? ''));
    }

    public function test_execute_rejects_too_many_geometries(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $geometries = [];
        for ($i = 0; $i < 7; $i++) {
            $geometries[] = [
                'geometry_type' => 'circle',
                'mode' => 'include',
                'geojson' => [
                    'center' => ['lat' => 27.9, 'lng' => -82.4],
                    'radius_m' => 1000,
                ],
            ];
        }

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 100000],
            ],
            'geometries' => $geometries,
        ]);

        $response->assertStatus(422);
        $response->assertJsonStructure(['errors' => ['geometries']]);
    }

    public function test_execute_rejects_polygon_with_too_many_vertices(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $ring = [];
        for ($i = 0; $i < 205; $i++) {
            $ring[] = [-82.5 + ($i * 0.0001), 27.8 + ($i * 0.0001)];
        }
        $ring[] = $ring[0];

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 100000],
            ],
            'geometries' => [
                [
                    'geometry_type' => 'polygon',
                    'mode' => 'include',
                    'geojson' => [
                        'coordinates' => [$ring],
                    ],
                ],
            ],
        ]);

        $response->assertStatus(422);
        $response->assertJsonStructure(['errors' => ['geometries']]);
    }

    public function test_execute_rejects_polygon_when_bbox_span_exceeds_configured_limit(): void
    {
        config(['agent_search.max_polygon_bbox_span_deg' => 0.05]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 100000],
            ],
            'geometries' => [
                [
                    'geometry_type' => 'polygon',
                    'mode' => 'include',
                    'geojson' => [
                        'coordinates' => [[
                            [-82.80, 27.70],
                            [-82.10, 27.70],
                            [-82.10, 28.20],
                            [-82.80, 28.20],
                            [-82.80, 27.70],
                        ]],
                    ],
                ],
            ],
        ]);

        $response->assertStatus(422);
        $response->assertJsonStructure(['errors' => ['geometries']]);
    }

    public function test_execute_rejects_circle_when_radius_exceeds_configured_limit(): void
    {
        config(['agent_search.max_circle_radius_m' => 5000]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 100000],
            ],
            'geometries' => [
                [
                    'geometry_type' => 'circle',
                    'mode' => 'include',
                    'geojson' => [
                        'center' => ['lat' => 27.9, 'lng' => -82.4],
                        'radius_m' => 6000,
                    ],
                ],
            ],
        ]);

        $response->assertStatus(422);
        $response->assertJsonStructure(['errors' => ['geometries']]);
    }

    public function test_user_can_save_and_load_automation_settings(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'automations', true);

        $update = $this->actingAs($user)->putJson('https://localhost/agent/automations/settings', [
            'nurture_enabled' => true,
            'nurture_mode' => 'basic',
            'dry_run' => false,
            'eligibility_tags' => ['buyer', 'vip'],
            'integration_health' => 'connected',
            'notes' => 'Auto follow-up enabled.',
        ]);

        $update->assertOk();
        $this->assertDatabaseHas('agent_automation_settings', ['user_id' => $user->id]);

        $show = $this->actingAs($user)->getJson('https://localhost/agent/automations/settings');
        $show->assertOk();
        $show->assertJsonPath('data.nurture_enabled', true);
        $show->assertJsonPath('data.nurture_mode', 'basic');
        $show->assertJsonPath('data.integration_health', 'connected');
    }

    public function test_user_can_create_and_delete_share_link(): void
    {
        config([
            'idx.platform_url' => 'https://idx.quantyralabs.cc',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Promo search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $create = $this->actingAs($user)->postJson('https://localhost/agent/share-links', [
            'agent_search_id' => $search->id,
            'attribution_json' => [
                'utm_source' => 'newsletter',
                'utm_campaign' => 'may-launch',
            ],
        ]);

        $create->assertCreated();
        $create->assertJsonPath('data.agent_search_id', $search->id);
        $url = (string) $create->json('data.url');
        $this->assertStringContainsString('https://idx.quantyralabs.cc/shared/', $url);

        $linkId = (int) $create->json('data.id');
        $this->assertDatabaseHas('agent_share_links', ['id' => $linkId, 'user_id' => $user->id]);

        $list = $this->actingAs($user)->getJson('https://localhost/agent/share-links');
        $list->assertOk();
        $list->assertJsonPath('data.0.id', $linkId);

        $delete = $this->actingAs($user)->deleteJson('https://localhost/agent/share-links/'.$linkId);
        $delete->assertNoContent();
        $this->assertDatabaseMissing('agent_share_links', ['id' => $linkId]);
    }

    public function test_user_cannot_attach_share_link_to_another_users_search(): void
    {
        /** @var User $owner */
        $owner = User::factory()->createOne();
        /** @var User $other */
        $other = User::factory()->createOne();

        $search = AgentSearch::query()->create([
            'user_id' => $owner->id,
            'name' => 'Owner-only search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $create = $this->actingAs($other)->postJson('https://localhost/agent/share-links', [
            'agent_search_id' => $search->id,
        ]);
        $create->assertNotFound();

        $this->assertDatabaseCount('agent_share_links', 0);
        $this->assertSame(0, AgentShareLink::query()->count('*'));
    }

    public function test_seo_template_share_link_is_deduplicated_per_user_and_search(): void
    {
        config([
            'idx.platform_url' => 'https://idx.quantyralabs.cc',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'seo_landing_pages', true);
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'SEO Template Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $first = $this->actingAs($user)->postJson('https://localhost/agent/share-links', [
            'agent_search_id' => $search->id,
            'template_kind' => 'seo_landing',
        ]);
        $first->assertCreated();
        $firstId = (int) $first->json('data.id');

        $second = $this->actingAs($user)->postJson('https://localhost/agent/share-links', [
            'agent_search_id' => $search->id,
            'template_kind' => 'seo_landing',
        ]);
        $second->assertOk();
        $second->assertJsonPath('data.id', $firstId);
        $second->assertJsonPath('data.reused_existing', true);
        $second->assertJsonPath('data.canonical_path', '/shared/'.$second->json('data.token').'/seo-template-search');

        $this->assertDatabaseCount('agent_share_links', 1);
    }

    public function test_user_can_deactivate_and_reactivate_share_link(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'seo_landing_pages', true);
        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'toggle-link-token',
            'attribution_json' => ['template_kind' => 'seo_landing'],
        ]);

        $deactivate = $this->actingAs($user)->putJson('https://localhost/agent/share-links/'.$link->id, [
            'status' => 'inactive',
        ]);
        $deactivate->assertOk();
        $deactivate->assertJsonPath('data.status', 'inactive');
        $this->assertNotNull(AgentShareLink::query()->findOrFail($link->id)->expires_at);

        $reactivate = $this->actingAs($user)->putJson('https://localhost/agent/share-links/'.$link->id, [
            'status' => 'active',
        ]);
        $reactivate->assertOk();
        $reactivate->assertJsonPath('data.status', 'active');
        $this->assertNull(AgentShareLink::query()->findOrFail($link->id)->expires_at);
    }

    public function test_user_can_filter_share_links_by_type_status_and_search(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'seo_landing_pages', true);

        $first = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'filter-token-1',
            'attribution_json' => ['template_kind' => 'seo_landing'],
        ]);
        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'filter-token-2',
            'attribution_json' => ['template_kind' => 'standard'],
            'expires_at' => now()->subMinute(),
        ]);

        $seoOnly = $this->actingAs($user)->getJson('https://localhost/agent/share-links?template_kind=seo_landing');
        $seoOnly->assertOk();
        $seoOnly->assertJsonCount(1, 'data');
        $seoOnly->assertJsonPath('data.0.id', $first->id);

        $inactiveOnly = $this->actingAs($user)->getJson('https://localhost/agent/share-links?status=inactive');
        $inactiveOnly->assertOk();
        $inactiveOnly->assertJsonCount(1, 'data');
        $inactiveOnly->assertJsonPath('data.0.token', 'filter-token-2');

        $search = $this->actingAs($user)->getJson('https://localhost/agent/share-links?q=token-1');
        $search->assertOk();
        $search->assertJsonCount(1, 'data');
        $search->assertJsonPath('data.0.token', 'filter-token-1');
    }

    public function test_share_link_payload_contains_canonical_path_and_url(): void
    {
        config([
            'idx.platform_url' => 'https://idx.quantyralabs.cc',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'seo_landing_pages', true);
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Canonical Payload Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $create = $this->actingAs($user)->postJson('https://localhost/agent/share-links', [
            'agent_search_id' => $search->id,
            'template_kind' => 'seo_landing',
        ]);
        $create->assertCreated();
        $create->assertJsonPath('data.canonical_path', '/shared/'.$create->json('data.token').'/canonical-payload-search');
        $create->assertJsonPath('data.canonical_url', 'https://idx.quantyralabs.cc/shared/'.$create->json('data.token').'/canonical-payload-search');
    }

    public function test_seo_template_share_link_creates_or_reuses_seo_landing_page_record(): void
    {
        config([
            'idx.platform_url' => 'https://idx.quantyralabs.cc',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'seo_landing_pages', true);
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'SEO Landing Source Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $first = $this->actingAs($user)->postJson('https://localhost/agent/share-links', [
            'agent_search_id' => $search->id,
            'template_kind' => 'seo_landing',
        ]);
        $first->assertCreated();

        $this->assertDatabaseHas('agent_seo_landing_pages', [
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'canonical_path' => '/shared/'.$first->json('data.token').'/seo-landing-source-search',
            'status' => 'active',
        ]);

        $second = $this->actingAs($user)->postJson('https://localhost/agent/share-links', [
            'agent_search_id' => $search->id,
            'template_kind' => 'seo_landing',
        ]);
        $second->assertOk();
        $second->assertJsonPath('data.reused_existing', true);
        $this->assertDatabaseCount('agent_seo_landing_pages', 1);

        $page = AgentSeoLandingPage::query()->where('user_id', $user->id)->first();
        $this->assertNotNull($page);
        $this->assertEquals($first->json('data.id'), $page?->agent_share_link_id);
    }

    public function test_seo_landings_endpoint_lists_records_for_user(): void
    {
        config([
            'idx.platform_url' => 'https://idx.quantyralabs.cc',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'seo_landing_pages', true);
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'SEO Landing List Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $create = $this->actingAs($user)->postJson('https://localhost/agent/share-links', [
            'agent_search_id' => $search->id,
            'template_kind' => 'seo_landing',
        ]);
        $create->assertCreated();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/share-links/seo-landings');
        $response->assertOk();
        $response->assertJsonCount(1, 'data');
        $response->assertJsonPath('data.0.agent_search_name', 'SEO Landing List Search');
        $response->assertJsonPath('data.0.status', 'active');
    }

    public function test_share_link_metrics_returns_expected_counts(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'metrics-a',
            'attribution_json' => ['template_kind' => 'seo_landing'],
        ]);
        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'metrics-b',
            'attribution_json' => ['template_kind' => 'seo_landing'],
            'expires_at' => now()->subMinute(),
        ]);
        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'metrics-c',
            'attribution_json' => ['template_kind' => 'standard'],
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/share-links/metrics');
        $response->assertOk();
        $response->assertJsonPath('data.total', 3);
        $response->assertJsonPath('data.seo_total', 2);
        $response->assertJsonPath('data.active_total', 2);
        $response->assertJsonPath('data.inactive_total', 1);
        $response->assertJsonPath('data.seo_active_total', 1);
        $response->assertJsonPath('data.seo_inactive_total', 1);
    }

    public function test_share_link_metrics_supports_time_window_for_created_and_visits(): void
    {
        CarbonImmutable::setTestNow('2026-04-30 12:00:00');

        try {
            /** @var User $user */
            $user = User::factory()->createOne();

            $recent = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'window-recent',
                'attribution_json' => ['visit_count' => 5],
            ]);
            AgentShareLink::query()->whereKey($recent->id)->update([
                'created_at' => now()->subDays(3),
                'updated_at' => now()->subDays(3),
            ]);

            $old = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'window-old',
                'attribution_json' => ['visit_count' => 7],
            ]);
            AgentShareLink::query()->whereKey($old->id)->update([
                'created_at' => now()->subDays(20),
                'updated_at' => now()->subDays(20),
            ]);

            $days7 = $this->actingAs($user)->getJson('https://localhost/agent/share-links/metrics?days=7');
            $days7->assertOk();
            $days7->assertJsonPath('data.created_in_window', 1);
            $days7->assertJsonPath('data.visits_in_window', 5);
            $days7->assertJsonPath('data.window_days', 7);

            $days30 = $this->actingAs($user)->getJson('https://localhost/agent/share-links/metrics?days=30');
            $days30->assertOk();
            $days30->assertJsonPath('data.created_in_window', 2);
            $days30->assertJsonPath('data.visits_in_window', 12);
            $days30->assertJsonPath('data.window_days', 30);
        } finally {
            CarbonImmutable::setTestNow();
        }
    }

    public function test_share_link_metrics_includes_window_over_window_deltas(): void
    {
        CarbonImmutable::setTestNow('2026-04-30 12:00:00');

        try {
            /** @var User $user */
            $user = User::factory()->createOne();

            $currentA = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'delta-current-a',
                'attribution_json' => ['visit_count' => 10],
            ]);
            AgentShareLink::query()->whereKey($currentA->id)->update([
                'created_at' => now()->subDays(2),
                'updated_at' => now()->subDays(2),
            ]);

            $currentB = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'delta-current-b',
                'attribution_json' => ['visit_count' => 5],
            ]);
            AgentShareLink::query()->whereKey($currentB->id)->update([
                'created_at' => now()->subDays(4),
                'updated_at' => now()->subDays(4),
            ]);

            $previous = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'delta-previous',
                'attribution_json' => ['visit_count' => 4],
            ]);
            AgentShareLink::query()->whereKey($previous->id)->update([
                'created_at' => now()->subDays(10),
                'updated_at' => now()->subDays(10),
            ]);

            $response = $this->actingAs($user)->getJson('https://localhost/agent/share-links/metrics?days=7');
            $response->assertOk();
            $response->assertJsonPath('data.created_in_window', 2);
            $response->assertJsonPath('data.created_previous_window', 1);
            $response->assertJsonPath('data.created_delta', 1);
            $response->assertJsonPath('data.visits_in_window', 15);
            $response->assertJsonPath('data.visits_previous_window', 4);
            $response->assertJsonPath('data.visits_delta', 11);
        } finally {
            CarbonImmutable::setTestNow();
        }
    }

    public function test_share_link_metrics_history_returns_daily_buckets(): void
    {
        CarbonImmutable::setTestNow('2026-04-30 12:00:00');

        try {
            /** @var User $user */
            $user = User::factory()->createOne();

            $recentA = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'history-a',
                'attribution_json' => ['visit_count' => 3],
            ]);
            AgentShareLink::query()->whereKey($recentA->id)->update([
                'created_at' => now()->subDays(1),
                'updated_at' => now()->subDays(1),
            ]);

            $recentB = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'history-b',
                'attribution_json' => ['visit_count' => 2],
            ]);
            AgentShareLink::query()->whereKey($recentB->id)->update([
                'created_at' => now()->subDays(1),
                'updated_at' => now()->subDays(1),
            ]);

            $today = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'history-c',
                'attribution_json' => ['visit_count' => 7],
            ]);
            AgentShareLink::query()->whereKey($today->id)->update([
                'created_at' => now(),
                'updated_at' => now(),
            ]);

            $response = $this->actingAs($user)->getJson('https://localhost/agent/share-links/metrics/history?days=7');
            $response->assertOk();
            $response->assertJsonPath('data.window_days', 7);
            $response->assertJsonPath('data.buckets.6.created', 1);
            $response->assertJsonPath('data.buckets.6.visits', 7);
            $response->assertJsonPath('data.buckets.5.created', 2);
            $response->assertJsonPath('data.buckets.5.visits', 5);
        } finally {
            CarbonImmutable::setTestNow();
        }
    }

    public function test_user_can_export_filtered_share_links_csv(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'seo_landing_pages', true);

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'csv-seo-active',
            'attribution_json' => ['template_kind' => 'seo_landing'],
        ]);
        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'csv-standard',
            'attribution_json' => ['template_kind' => 'standard'],
        ]);
        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'csv-seo-inactive',
            'attribution_json' => ['template_kind' => 'seo_landing'],
            'expires_at' => now()->subMinute(),
        ]);

        $response = $this->actingAs($user)->get('https://localhost/agent/share-links/export.csv?template_kind=seo_landing&status=active&q=seo');
        $response->assertOk();
        $response->assertHeader('content-type', 'text/csv; charset=UTF-8');
        $response->assertHeader('content-disposition', 'attachment; filename=agent-share-links.csv');

        $csv = $response->streamedContent();
        $this->assertIsString($csv);
        $this->assertStringContainsString('token,template_kind,status,canonical_path,created_at', $csv);
        $this->assertStringContainsString('csv-seo-active,seo_landing,active', $csv);
        $this->assertFalse(Str::contains($csv, 'csv-standard'));
        $this->assertFalse(Str::contains($csv, 'csv-seo-inactive'));
    }

    public function test_share_link_csv_export_includes_metrics_snapshot_header(): void
    {
        CarbonImmutable::setTestNow('2026-04-30 12:00:00');

        try {
            /** @var User $user */
            $user = User::factory()->createOne();

            $current = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'snapshot-current',
                'attribution_json' => ['visit_count' => 6, 'template_kind' => 'seo_landing'],
            ]);
            AgentShareLink::query()->whereKey($current->id)->update([
                'created_at' => now()->subDays(2),
                'updated_at' => now()->subDays(2),
            ]);

            $previous = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'snapshot-previous',
                'attribution_json' => ['visit_count' => 2, 'template_kind' => 'seo_landing'],
            ]);
            AgentShareLink::query()->whereKey($previous->id)->update([
                'created_at' => now()->subDays(10),
                'updated_at' => now()->subDays(10),
            ]);

            $response = $this->actingAs($user)->get('https://localhost/agent/share-links/export.csv?days=7');
            $response->assertOk();
            $csv = $response->streamedContent();
            $this->assertIsString($csv);
            $this->assertStringContainsString('metric,window_days,created_in_window,created_previous_window,created_delta,visits_in_window,visits_previous_window,visits_delta', $csv);
            $this->assertStringContainsString('summary,7,1,1,0,6,2,4', $csv);
        } finally {
            CarbonImmutable::setTestNow();
        }
    }

    public function test_user_can_export_metrics_history_csv(): void
    {
        CarbonImmutable::setTestNow('2026-04-30 12:00:00');

        try {
            /** @var User $user */
            $user = User::factory()->createOne();

            $one = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'history-export-1',
                'attribution_json' => ['visit_count' => 4],
            ]);
            AgentShareLink::query()->whereKey($one->id)->update([
                'created_at' => now()->subDays(1),
                'updated_at' => now()->subDays(1),
            ]);

            $two = AgentShareLink::query()->create([
                'user_id' => $user->id,
                'token' => 'history-export-2',
                'attribution_json' => ['visit_count' => 2],
            ]);
            AgentShareLink::query()->whereKey($two->id)->update([
                'created_at' => now(),
                'updated_at' => now(),
            ]);

            $response = $this->actingAs($user)->get('https://localhost/agent/share-links/metrics/history.csv?days=7');
            $response->assertOk();
            $response->assertHeader('content-type', 'text/csv; charset=UTF-8');
            $response->assertHeader('content-disposition', 'attachment; filename=agent-share-links-history.csv');

            $csv = $response->streamedContent();
            $this->assertIsString($csv);
            $this->assertStringContainsString('date,created,visits', $csv);
            $this->assertStringContainsString('2026-04-29,1,4', $csv);
            $this->assertStringContainsString('2026-04-30,1,2', $csv);
        } finally {
            CarbonImmutable::setTestNow();
        }
    }

    public function test_share_link_operations_endpoint_reports_prune_config_and_estimate(): void
    {
        config(['agent_portal.share_links.prune_days' => 45]);

        /** @var User $user */
        $user = User::factory()->createOne();

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'ops-keep',
            'expires_at' => now()->subDays(10),
        ]);

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'ops-prune',
            'expires_at' => now()->subDays(100),
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/share-links/operations');
        $response->assertOk();
        $response->assertJsonPath('data.prune_days', 45);
        $response->assertJsonPath('data.prune_candidate_count', 1);
        $response->assertJsonPath('data.schedule', 'daily 03:30');
        $response->assertJsonPath('data.prune_command', 'agent:prune-share-links --days=45');
    }

    public function test_share_link_operations_estimate_endpoint_supports_custom_days(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'estimate-prune',
            'expires_at' => now()->subDays(120),
        ]);
        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'estimate-keep',
            'expires_at' => now()->subDays(10),
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/share-links/operations/estimate?days=90');
        $response->assertOk();
        $response->assertJsonPath('data.days', 90);
        $response->assertJsonPath('data.prune_candidate_count', 1);

        $invalid = $this->actingAs($user)->getJson('https://localhost/agent/share-links/operations/estimate?days=0');
        $invalid->assertStatus(422);
    }

    public function test_user_can_crud_alert_templates(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $create = $this->actingAs($user)->postJson('https://localhost/agent/alert-templates', [
            'name' => 'Weekly digest template',
            'template_type' => 'listing',
            'body_json' => [
                'schedule' => ['cadence' => 'weekly'],
                'status' => 'active',
            ],
        ]);
        $create->assertCreated();
        $templateId = (int) $create->json('data.id');
        $this->assertDatabaseHas('agent_alert_templates', ['id' => $templateId, 'user_id' => $user->id]);

        $list = $this->actingAs($user)->getJson('https://localhost/agent/alert-templates');
        $list->assertOk();
        $list->assertJsonCount(1, 'data');
        $list->assertJsonPath('data.0.id', $templateId);

        $update = $this->actingAs($user)->putJson('https://localhost/agent/alert-templates/'.$templateId, [
            'name' => 'Weekly digest template v2',
            'template_type' => 'listing',
            'body_json' => [
                'schedule' => ['cadence' => 'monthly'],
                'status' => 'paused',
            ],
        ]);
        $update->assertOk();
        $update->assertJsonPath('data.name', 'Weekly digest template v2');

        $delete = $this->actingAs($user)->deleteJson('https://localhost/agent/alert-templates/'.$templateId);
        $delete->assertNoContent();
        $this->assertDatabaseMissing('agent_alert_templates', ['id' => $templateId]);
    }

    public function test_user_can_create_alert_from_template(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $template = AgentAlertTemplate::query()->create([
            'user_id' => $user->id,
            'name' => 'Market monitor',
            'template_type' => 'market_activity',
            'body_json' => [
                'status' => 'paused',
                'schedule' => ['cadence' => 'weekly'],
            ],
        ]);

        $createAlert = $this->actingAs($user)->postJson('https://localhost/agent/alerts/from-template', [
            'template_id' => $template->id,
            'name' => 'Market monitor - Tampa',
        ]);
        $createAlert->assertCreated();
        $createAlert->assertJsonPath('data.alert_type', 'market_activity');
        $createAlert->assertJsonPath('data.status', 'paused');
        $createAlert->assertJsonPath('data.schedule_json.cadence', 'weekly');
    }

    public function test_user_can_list_contacts_with_filters(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => [
                'name' => 'Jane Buyer',
                'email' => 'jane@example.test',
                'status' => 'new',
                'stage' => 'qualified',
                'lead_score' => 88,
            ],
            'quantyra_domain' => 'example.com',
        ]);
        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => [
                'name' => 'John Seller',
                'email' => 'john@example.test',
                'status' => 'contacted',
                'stage' => 'nurture',
                'lead_score' => 45,
            ],
            'quantyra_domain' => 'example.com',
        ]);
        QuantyraLead::query()->create([
            'ghl_location_id' => 'other-location',
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => [
                'name' => 'Outside Scope',
                'status' => 'new',
                'lead_score' => 99,
            ],
            'quantyra_domain' => 'outside.com',
        ]);

        $list = $this->actingAs($user)->getJson('https://localhost/agent/contacts?status=new&search=Jane');
        $list->assertOk();
        $list->assertJsonPath('data.meta.total', 1);
        $list->assertJsonPath('data.items.0.payload.name', 'Jane Buyer');
    }

    public function test_user_can_view_single_contact_within_scope(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Scoped Lead', 'status' => 'new'],
        ]);
        $outside = QuantyraLead::query()->create([
            'ghl_location_id' => 'other-location',
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Outside Lead', 'status' => 'new'],
        ]);

        $show = $this->actingAs($user)->getJson('https://localhost/agent/contacts/'.$lead->id);
        $show->assertOk();
        $show->assertJsonPath('data.id', $lead->id);

        $this->actingAs($user)->getJson('https://localhost/agent/contacts/'.$outside->id)->assertNotFound();
    }

    public function test_user_can_update_scoped_contact_fields(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Lead Update Candidate', 'status' => 'new'],
        ]);

        $update = $this->actingAs($user)->putJson('https://localhost/agent/contacts/'.$lead->id, [
            'status' => 'contacted',
            'stage' => 'qualified',
            'tags' => ['buyer', 'follow-up'],
            'notes' => 'Wants homes near downtown.',
        ]);

        $update->assertOk();
        $update->assertJsonPath('data.payload.status', 'contacted');
        $update->assertJsonPath('data.payload.stage', 'qualified');
        $update->assertJsonPath('data.payload.tags.0', 'buyer');
        $update->assertJsonPath('data.payload.notes', 'Wants homes near downtown.');
    }

    public function test_contact_activity_endpoint_returns_handoff_and_update_events(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Downtown Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);
        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Timeline Contact', 'status' => 'new', 'email' => 'timeline@example.test'],
        ]);

        $this->actingAs($user)->putJson('https://localhost/agent/contacts/'.$lead->id, [
            'status' => 'contacted',
            'notes' => 'Called and qualified.',
        ])->assertOk();

        $this->actingAs($user)->postJson('https://localhost/agent/contacts/'.$lead->id.'/handoff/alert', [
            'agent_search_id' => $search->id,
            'name' => 'Timeline Alert',
            'alert_type' => 'listing',
            'cadence' => 'daily',
        ])->assertCreated();

        $activity = $this->actingAs($user)->getJson('https://localhost/agent/contacts/'.$lead->id.'/activity');
        $activity->assertOk();
        $activity->assertJsonFragment(['type' => 'contact_updated', 'channel' => 'crm']);
        $activity->assertJsonFragment(['type' => 'search_to_alert_handoff', 'channel' => 'alert']);
    }

    public function test_contact_activity_endpoint_tags_email_and_site_channels_from_payload_log(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $at = now()->toIso8601String();
        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => [
                'name' => 'Channel Lead',
                'status' => 'new',
                'activity_log' => [
                    [
                        'type' => 'email_opened',
                        'at' => $at,
                        'campaign' => 'spring',
                    ],
                    [
                        'type' => 'listing_view',
                        'at' => $at,
                        'listing_key' => 'stellar:123',
                    ],
                ],
            ],
        ]);

        $activity = $this->actingAs($user)->getJson('https://localhost/agent/contacts/'.$lead->id.'/activity');
        $activity->assertOk();
        $activity->assertJsonFragment(['type' => 'email_opened', 'channel' => 'email']);
        $activity->assertJsonFragment(['type' => 'listing_view', 'channel' => 'site']);
    }

    public function test_contact_activity_endpoint_includes_contact_scoped_custom_events(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Scoped Event Lead', 'status' => 'new'],
        ]);

        $this->actingAs($user)->postJson('https://localhost/agent/dashboard/events', [
            'event_type' => 'apply_filter',
            'title' => 'Applied max price filter',
            'metadata' => ['contact_id' => $lead->id, 'field' => 'ListPrice'],
        ])->assertCreated();
        $this->actingAs($user)->postJson('https://localhost/agent/dashboard/events', [
            'event_type' => 'map_draw',
            'title' => 'Drew include polygon',
            'metadata' => ['contact_id' => $lead->id],
        ])->assertCreated();

        $activity = $this->actingAs($user)->getJson('https://localhost/agent/contacts/'.$lead->id.'/activity');
        $activity->assertOk();
        $activity->assertJsonFragment(['type' => 'apply_filter', 'channel' => 'crm']);
        $activity->assertJsonFragment(['type' => 'map_draw', 'channel' => 'crm']);
    }

    public function test_user_can_handoff_contact_to_alert(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Waterfront Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);
        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Handoff Contact', 'email' => 'handoff@example.test'],
        ]);

        $handoff = $this->actingAs($user)->postJson('https://localhost/agent/contacts/'.$lead->id.'/handoff/alert', [
            'agent_search_id' => $search->id,
            'name' => 'Handoff Alert',
            'alert_type' => 'listing',
            'cadence' => 'weekly',
        ]);

        $handoff->assertCreated();
        $handoff->assertJsonPath('data.alert.name', 'Handoff Alert');
        $handoff->assertJsonPath('data.alert.agent_search_id', $search->id);
        $handoff->assertJsonPath('data.alert.schedule_json.handoff.contact_lead_id', $lead->id);
    }

    public function test_user_can_create_contact_and_handoff_to_alert(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'New Contact Flow Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $handoff = $this->actingAs($user)->postJson('https://localhost/agent/contacts/handoff/alert', [
            'contact' => [
                'name' => 'New Handoff Contact',
                'email' => 'new-handoff@example.test',
                'phone' => '555-0100',
            ],
            'agent_search_id' => $search->id,
            'name' => 'New Contact Handoff Alert',
            'alert_type' => 'listing',
            'cadence' => 'daily',
        ]);

        $handoff->assertCreated();
        $handoff->assertJsonPath('data.contact.payload.name', 'New Handoff Contact');
        $handoff->assertJsonPath('data.alert.name', 'New Contact Handoff Alert');
        $handoff->assertJsonPath('data.alert.schedule_json.handoff.contact_name', 'New Handoff Contact');
    }

    public function test_agent_dashboard_summary_returns_upcoming_active_alerts_ordered_by_next_run(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $later = now()->addDays(10);
        $sooner = now()->addDay();

        $alertLater = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Later pulse',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'weekly'],
            'next_run_at' => $later,
        ]);
        $alertSooner = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Sooner pulse',
            'alert_type' => 'market_activity',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
            'next_run_at' => $sooner,
        ]);
        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Paused alert',
            'alert_type' => 'home_value',
            'status' => 'paused',
            'schedule_json' => ['cadence' => 'daily'],
            'next_run_at' => now(),
        ]);
        $alertUnscheduled = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Unscheduled active',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'monthly'],
            'next_run_at' => null,
        ]);

        $summary = $this->actingAs($user)->getJson('https://localhost/agent/dashboard/summary');
        $summary->assertOk();
        $summary->assertJsonStructure(['data' => ['upcoming_alerts']]);
        $ids = collect($summary->json('data.upcoming_alerts'))->pluck('id')->all();
        $this->assertSame(
            [$alertSooner->id, $alertLater->id, $alertUnscheduled->id],
            $ids,
            'Active alerts: soonest next_run_at first, null next_run_at last.'
        );
        $pausedId = AgentAlert::query()->where('name', 'Paused alert')->value('id');
        $this->assertNotContains($pausedId, $ids);
    }

    public function test_agent_dashboard_summary_returns_kpis_and_activity(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Summary Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);
        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Summary Alert',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);
        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => null,
            'token' => 'summarytoken',
            'slug' => 'summary-slug',
            'template_kind' => 'standard',
            'attribution_json' => null,
            'expires_at' => null,
            'visits' => 0,
        ]);
        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Summary Lead', 'status' => 'new'],
        ]);

        $summary = $this->actingAs($user)->getJson('https://localhost/agent/dashboard/summary');
        $summary->assertOk();
        $summary->assertJsonPath('data.kpis.contacts', 1);
        $summary->assertJsonPath('data.kpis.active_alerts', 1);
        $summary->assertJsonPath('data.kpis.saved_searches', 1);
        $summary->assertJsonPath('data.kpis.active_share_links', 1);
        $summary->assertJsonCount(4, 'data.activity_feed');
    }

    public function test_alert_summary_returns_type_kpis_and_runs(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $listing = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Listing Alert',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);
        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Market Alert',
            'alert_type' => 'market_activity',
            'status' => 'active',
        ]);
        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Home Value Alert',
            'alert_type' => 'home_value',
            'status' => 'paused',
        ]);

        AgentAlertRun::query()->create([
            'agent_alert_id' => $listing->id,
            'status' => 'sent',
            'metadata_json' => [],
            'ran_at' => now()->subDay(),
        ]);

        $summary = $this->actingAs($user)->getJson('https://localhost/agent/alerts/summary');
        $summary->assertOk();
        $summary->assertJsonPath('data.total', 3);
        $summary->assertJsonPath('data.active', 2);
        $summary->assertJsonPath('data.by_type.listing', 1);
        $summary->assertJsonPath('data.by_type.market_activity', 1);
        $summary->assertJsonPath('data.by_type.home_value', 1);
        $summary->assertJsonPath('data.runs_last_30_days', 1);
    }

    public function test_alert_history_returns_daily_run_buckets_by_type(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $listing = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Listing Alert',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);
        $market = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Market Alert',
            'alert_type' => 'market_activity',
            'status' => 'active',
        ]);

        AgentAlertRun::query()->create([
            'agent_alert_id' => $listing->id,
            'status' => 'sent',
            'metadata_json' => [],
            'ran_at' => now()->subDays(1),
        ]);
        AgentAlertRun::query()->create([
            'agent_alert_id' => $market->id,
            'status' => 'sent',
            'metadata_json' => [],
            'ran_at' => now()->subDays(1),
        ]);
        AgentAlertRun::query()->create([
            'agent_alert_id' => $listing->id,
            'status' => 'sent',
            'metadata_json' => [],
            'ran_at' => now()->subDays(2),
        ]);

        $history = $this->actingAs($user)->getJson('https://localhost/agent/alerts/history?days=7');
        $history->assertOk();
        $history->assertJsonPath('data.meta.days', 7);
        $history->assertJsonPath('data.meta.total_runs', 3);
        $history->assertJsonCount(7, 'data.buckets');
    }

    public function test_alerts_index_filters_by_alert_type_query(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'L1',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);
        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'M1',
            'alert_type' => 'market_activity',
            'status' => 'active',
        ]);
        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'H1',
            'alert_type' => 'home_value',
            'status' => 'paused',
        ]);

        $marketOnly = $this->actingAs($user)->getJson('https://localhost/agent/alerts?alert_type=market_activity');
        $marketOnly->assertOk();
        $marketOnly->assertJsonCount(1, 'data');
        $marketOnly->assertJsonPath('data.0.name', 'M1');

        $all = $this->actingAs($user)->getJson('https://localhost/agent/alerts');
        $all->assertOk();
        $all->assertJsonCount(3, 'data');
    }

    public function test_alerts_index_rejects_invalid_alert_type_query(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/alerts?alert_type=unknown');

        $response->assertStatus(422);
        $response->assertJsonPath('errors.alert_type.0', 'Must be one of: listing, market_activity, home_value.');
    }

    public function test_alert_store_rejects_saved_search_owned_by_another_user(): void
    {
        /** @var User $owner */
        $owner = User::factory()->createOne();
        /** @var User $actor */
        $actor = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $owner->id,
            'name' => 'Owner only',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $response = $this->actingAs($actor)->postJson('https://localhost/agent/alerts', [
            'name' => 'Cross-tenant alert',
            'alert_type' => 'listing',
            'agent_search_id' => $search->id,
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
        ]);

        $response->assertStatus(422);
    }

    public function test_user_can_run_automation_integration_lifecycle_actions(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'automations', true);

        $connect = $this->actingAs($user)->postJson('https://localhost/agent/automations/settings/integrations/connect', [
            'provider' => 'ghl',
        ]);
        $connect->assertOk();
        $connect->assertJsonPath('data.integrations.ghl.status', 'connected');

        $reconnect = $this->actingAs($user)->postJson('https://localhost/agent/automations/settings/integrations/reconnect', [
            'provider' => 'ghl',
        ]);
        $reconnect->assertOk();
        $reconnect->assertJsonPath('data.integrations.ghl.status', 'connected');

        $disconnect = $this->actingAs($user)->postJson('https://localhost/agent/automations/settings/integrations/disconnect', [
            'provider' => 'ghl',
        ]);
        $disconnect->assertOk();
        $disconnect->assertJsonPath('data.integrations.ghl.status', 'disconnected');
    }

    public function test_user_can_save_and_load_agent_portal_settings(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $update = $this->actingAs($user)->putJson('https://localhost/agent/settings', [
            'notification_email_enabled' => true,
            'notification_sms_enabled' => false,
            'weekly_digest_enabled' => true,
            'alert_default_cadence' => 'weekly',
            'timezone' => 'America/Chicago',
            'theme_density' => 'comfortable',
            'onboarding_tips_enabled' => false,
            'feature_flags' => ['beta_search', 'advanced_alerts'],
        ]);
        $update->assertOk();
        $update->assertJsonPath('data.alert_default_cadence', 'weekly');
        $this->assertDatabaseHas('agent_portal_settings', ['user_id' => $user->id]);

        $show = $this->actingAs($user)->getJson('https://localhost/agent/settings');
        $show->assertOk();
        $show->assertJsonPath('data.timezone', 'America/Chicago');
        $show->assertJsonPath('data.theme_density', 'comfortable');
        $show->assertJsonPath('data.feature_flags.0', 'beta_search');

        $row = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $this->assertNotNull($row);
    }

    public function test_put_agent_portal_settings_merges_without_wiping_hide_onboarding_checklist(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $this->actingAs($user)->postJson('https://localhost/agent/settings/onboarding-checklist/dismiss')->assertOk();

        $this->actingAs($user)->putJson('https://localhost/agent/settings', [
            'notification_email_enabled' => true,
            'notification_sms_enabled' => false,
            'weekly_digest_enabled' => true,
            'alert_default_cadence' => 'daily',
            'timezone' => 'America/Los_Angeles',
            'theme_density' => 'compact',
            'onboarding_tips_enabled' => true,
        ])->assertOk();

        $row = AgentPortalSetting::query()->where('user_id', $user->id)->first();
        $this->assertNotNull($row);
        $this->assertSame('America/Los_Angeles', $row->settings_json['timezone'] ?? null);
        $this->assertTrue((bool) ($row->settings_json['hide_agent_onboarding_checklist'] ?? false));
    }

    public function test_lookup_options_returns_multiple_enum_values_for_field(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.dataset' => 'stellar',
            'bridge.lookups_cache_ttl_seconds' => 3600,
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['LookupName' => 'PropertyType', 'LookupValue' => 'Residential'],
                    ['LookupName' => 'PropertyType', 'LookupValue' => 'Condominium'],
                    ['LookupName' => 'PropertyType', 'LookupValue' => 'Commercial'],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/searches/lookups/options?field=PropertyType');

        $response->assertOk();
        $response->assertJsonPath('data.0.dataset_code', 'stellar');
        $response->assertJsonCount(3, 'data.0.values');
        $response->assertJsonPath('data.0.values.0.LookupValue', 'Residential');
        $response->assertJsonPath('data.0.values.1.LookupValue', 'Condominium');
        $response->assertJsonPath('data.0.values.2.LookupValue', 'Commercial');
    }

    public function test_search_execute_coerces_numeric_filter_values(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['ListingKey' => 'abc-123', 'ListPrice' => 350000, 'City' => 'Tampa'],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'property.bedrooms_total', 'operator' => 'gte', 'value' => '3'],
                ['field' => 'property.list_price', 'operator' => 'lte', 'value' => '500000'],
                ['field' => 'location.city', 'operator' => 'eq', 'value' => 'Tampa'],
            ],
        ]);

        $response->assertOk();
        $response->assertJsonStructure(['data' => ['items', 'meta']]);
    }

    public function test_search_execute_supports_viewport_bbox_filter(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    [
                        'ListingKey' => 'vp-1',
                        'ListPrice' => 250000,
                        'Latitude' => 27.95,
                        'Longitude' => -82.45,
                        'City' => 'Tampa',
                    ],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'location.viewport', 'operator' => 'bbox', 'value' => '27.8,-82.6,28.0,-82.3'],
            ],
        ]);

        $response->assertOk();
        $response->assertJsonPath('data.meta.total_items', 1);
        $response->assertJsonPath('data.items.0.listingId', 'vp-1');

        $viewportEvent = AgentActivityEvent::query()
            ->where('user_id', $user->id)
            ->where('event_type', 'search_execute')
            ->latest('id')
            ->first();
        $this->assertNotNull($viewportEvent);
        $vm = is_array($viewportEvent->metadata_json) ? $viewportEvent->metadata_json : [];
        $this->assertSame('viewport', $vm['trigger'] ?? null);
    }

    public function test_search_execute_accepts_telemetry_trigger_override(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['ListingKey' => 'manual-override', 'ListPrice' => 425000],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 400000],
            ],
            'telemetry' => ['trigger' => 'viewport_pan'],
        ]);

        $response->assertOk();

        $event = AgentActivityEvent::query()
            ->where('user_id', $user->id)
            ->where('event_type', 'search_execute')
            ->latest('id')
            ->first();
        $this->assertNotNull($event);
        $meta = is_array($event->metadata_json) ? $event->metadata_json : [];
        $this->assertSame('viewport_pan', $meta['trigger'] ?? null);
    }

    public function test_search_execute_supports_contains_operator(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['ListingKey' => 'c-1', 'City' => 'St. Petersburg'],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'location.city', 'operator' => 'contains', 'value' => 'Petersburg'],
            ],
        ]);

        $response->assertOk();
        $response->assertJsonPath('data.meta.total_items', 1);
    }

    public function test_serialize_search_returns_shareable_url(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->postJson('https://localhost/agent/searches/serialize', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 200000],
                ['field' => 'location.city', 'operator' => 'eq', 'value' => 'Tampa'],
            ],
            'geometries' => [],
        ]);

        $response->assertOk();
        $response->assertJsonStructure(['data' => ['url', 'params']]);
        $url = $response->json('data.url');
        $this->assertStringContainsString('/agent/searches/shared?', $url);
        $this->assertNotEmpty($response->json('data.params.f'));
    }

    public function test_shared_search_executes_from_serialized_url_params(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['ListingKey' => 'shared-1', 'ListPrice' => 350000, 'City' => 'Tampa'],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $serialize = $this->actingAs($user)->postJson('https://localhost/agent/searches/serialize', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 200000],
            ],
            'geometries' => [],
        ]);
        $serialize->assertOk();

        $params = $serialize->json('data.params');
        $sharedResponse = $this->actingAs($user)->getJson('https://localhost/agent/searches/shared?f='.urlencode($params['f']));

        $sharedResponse->assertOk();
        $sharedResponse->assertJsonPath('meta.shared', true);
        $sharedResponse->assertJsonPath('data.meta.total_items', 1);
    }

    public function test_shared_search_records_search_execute_when_geometry_encoded(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['ListingKey' => 'shared-geo', 'ListPrice' => 350000, 'City' => 'Tampa'],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $serialize = $this->actingAs($user)->postJson('https://localhost/agent/searches/serialize', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 200000],
            ],
            'geometries' => [
                [
                    'geometry_type' => 'polygon',
                    'mode' => 'include',
                    'geojson' => [
                        'coordinates' => [[
                            [-82.80, 27.70],
                            [-82.10, 27.70],
                            [-82.10, 28.20],
                            [-82.80, 28.20],
                            [-82.80, 27.70],
                        ]],
                    ],
                ],
            ],
        ]);
        $serialize->assertOk();
        $params = $serialize->json('data.params');
        $this->assertArrayHasKey('g', $params);

        AgentActivityEvent::query()->where('user_id', $user->id)->delete();

        $sharedResponse = $this->actingAs($user)->getJson(
            'https://localhost/agent/searches/shared?f='.urlencode($params['f']).'&g='.urlencode($params['g'])
        );
        $sharedResponse->assertOk();
        $this->assertDatabaseHas('agent_activity_events', [
            'user_id' => $user->id,
            'event_type' => 'search_execute',
        ]);

        $event = AgentActivityEvent::query()
            ->where('user_id', $user->id)
            ->where('event_type', 'search_execute')
            ->latest('id')
            ->first();
        $this->assertNotNull($event);
        $meta = is_array($event->metadata_json) ? $event->metadata_json : [];
        $this->assertSame(1, (int) ($meta['geometry_count'] ?? 0));
        $this->assertSame('shared', (string) ($meta['surface'] ?? ''));
    }

    public function test_shared_search_with_empty_params_returns_empty(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response(['value' => []], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/searches/shared?f=');

        $response->assertOk();
        $response->assertJsonPath('meta.shared', true);
    }

    public function test_alert_template_crud_with_audit_trail(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $create = $this->actingAs($user)->postJson('https://localhost/agent/alert-templates', [
            'name' => 'Weekly Listing Digest',
            'template_type' => 'listing',
            'body_json' => ['status' => 'active', 'schedule' => ['cadence' => 'weekly']],
            'schedule_json' => ['cadence' => 'weekly', 'day_of_week' => 1, 'time_of_day' => '09:00'],
        ]);
        $create->assertCreated();
        $create->assertJsonPath('data.name', 'Weekly Listing Digest');
        $create->assertJsonPath('data.template_type', 'listing');
        $create->assertJsonPath('data.audit_json.0.action', 'created');
        $create->assertJsonPath('data.usage_count', 0);
        $templateId = $create->json('data.id');

        $update = $this->actingAs($user)->putJson("https://localhost/agent/alert-templates/{$templateId}", [
            'name' => 'Weekly Listing Digest Updated',
            'template_type' => 'listing',
            'body_json' => ['status' => 'active'],
            'schedule_json' => ['cadence' => 'daily', 'time_of_day' => '08:00'],
        ]);
        $update->assertOk();
        $update->assertJsonPath('data.name', 'Weekly Listing Digest Updated');
        $update->assertJsonPath('data.audit_json.1.action', 'updated');

        $list = $this->actingAs($user)->getJson('https://localhost/agent/alert-templates');
        $list->assertOk();
        $list->assertJsonPath('data.0.usage_count', 0);
        $list->assertJsonPath('data.0.last_used_at', null);

        $delete = $this->actingAs($user)->deleteJson("https://localhost/agent/alert-templates/{$templateId}");
        $delete->assertNoContent();
    }

    public function test_alert_from_template_increments_usage_count(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $template = AgentAlertTemplate::query()->create([
            'user_id' => $user->id,
            'name' => 'Market Pulse',
            'template_type' => 'market_activity',
            'body_json' => ['status' => 'active'],
            'schedule_json' => ['cadence' => 'daily'],
            'usage_count' => 0,
        ]);

        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Market Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/alerts/from-template', [
            'template_id' => $template->id,
            'name' => 'My Market Alert',
            'agent_search_id' => $search->id,
        ]);
        $response->assertCreated();
        $response->assertJsonPath('data.alert_type', 'market_activity');

        $template->refresh();
        $this->assertEquals(1, $template->usage_count);
        $this->assertNotNull($template->last_used_at);
    }

    public function test_field_catalog_endpoint_returns_categorized_fields(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        MlsFieldCatalog::query()->create([
            'mls_code' => 'stellar',
            'dataset_code' => 'stellar',
            'source_field_key' => 'ListPrice',
            'canonical_field_key' => 'property.list_price',
            'display_label' => 'List Price',
            'field_type' => 'number',
            'category' => 'general',
            'operators_json' => ['eq', 'gte', 'lte', 'between'],
            'is_reso_standard' => true,
            'is_custom_mls_field' => false,
        ]);

        MlsFieldCatalog::query()->create([
            'mls_code' => 'stellar',
            'dataset_code' => 'stellar',
            'source_field_key' => 'City',
            'canonical_field_key' => 'location.city',
            'display_label' => 'City',
            'field_type' => 'string',
            'category' => 'locations',
            'operators_json' => ['eq', 'contains'],
            'is_reso_standard' => true,
            'is_custom_mls_field' => false,
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/searches/fields');
        $response->assertOk();
        $response->assertJsonStructure(['data' => ['fields', 'categories']]);
        $response->assertJsonCount(2, 'data.fields');
        $this->assertContains('general', $response->json('data.categories'));
        $this->assertContains('locations', $response->json('data.categories'));
    }

    public function test_field_catalog_endpoint_filters_by_category(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        MlsFieldCatalog::query()->create([
            'mls_code' => 'stellar',
            'dataset_code' => 'stellar',
            'source_field_key' => 'ListPrice',
            'canonical_field_key' => 'property.list_price',
            'display_label' => 'List Price',
            'field_type' => 'number',
            'category' => 'general',
            'operators_json' => ['eq', 'gte', 'lte'],
            'is_reso_standard' => true,
            'is_custom_mls_field' => false,
        ]);

        MlsFieldCatalog::query()->create([
            'mls_code' => 'stellar',
            'dataset_code' => 'stellar',
            'source_field_key' => 'City',
            'canonical_field_key' => 'location.city',
            'display_label' => 'City',
            'field_type' => 'string',
            'category' => 'locations',
            'operators_json' => ['eq', 'contains'],
            'is_reso_standard' => true,
            'is_custom_mls_field' => false,
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/searches/fields?category=general');
        $response->assertOk();
        $response->assertJsonCount(1, 'data.fields');
        $response->assertJsonPath('data.fields.0.source_field_key', 'ListPrice');
    }

    public function test_field_catalog_service_sync_creates_entries(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.dataset' => 'stellar',
            'bridge.lookups_cache_ttl_seconds' => 3600,
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['LookupName' => 'ListPrice', 'LookupValue' => 'List Price'],
                    ['LookupName' => 'BedroomsTotal', 'LookupValue' => 'Bedrooms Total'],
                    ['LookupName' => 'City', 'LookupValue' => 'City'],
                ],
            ], 200),
        ]);

        $service = $this->app->make(FieldCatalogService::class);
        $count = $service->syncFieldCatalog('stellar', 'stellar');

        $this->assertEquals(3, $count);
        $this->assertDatabaseHas('mls_field_catalog', [
            'mls_code' => 'stellar',
            'dataset_code' => 'stellar',
            'source_field_key' => 'ListPrice',
            'canonical_field_key' => 'property.list_price',
            'category' => 'general',
        ]);
        $this->assertDatabaseHas('mls_field_catalog', [
            'source_field_key' => 'City',
            'canonical_field_key' => 'location.city',
            'category' => 'locations',
        ]);
        $this->assertDatabaseHas('field_mapping_adapters', [
            'mls_code' => 'stellar',
            'canonical_field_key' => 'property.list_price',
            'source_field_key' => 'ListPrice',
        ]);
    }

    public function test_alert_scheduler_computes_next_run_at(): void
    {
        $scheduler = new AlertSchedulerService;

        $immediateNext = $scheduler->computeNextRunAt('immediate');
        $this->assertTrue($immediateNext->isFuture());

        $dailyNext = $scheduler->computeNextRunAt('daily', ['time_of_day' => '23:59']);
        $this->assertTrue($dailyNext->isFuture());

        $weeklyNext = $scheduler->computeNextRunAt('weekly', ['day_of_week' => 1, 'time_of_day' => '09:00']);
        $this->assertTrue($weeklyNext->isFuture());
        $this->assertEquals(1, $weeklyNext->dayOfWeekIso);

        $monthlyNext = $scheduler->computeNextRunAt('monthly', ['time_of_day' => '08:00']);
        $this->assertTrue($monthlyNext->isFuture());
    }

    public function test_alert_scheduler_cooldown_prevents_rerun(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Cooldown Test',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);

        AgentAlertRun::query()->create([
            'agent_alert_id' => $alert->id,
            'status' => 'sent',
            'metadata_json' => [],
            'ran_at' => now()->subMinutes(5),
        ]);

        $scheduler = new AlertSchedulerService;
        $this->assertTrue($scheduler->isWithinCooldown($alert, 60));
    }

    public function test_alert_store_sets_initial_next_run_at(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->postJson('https://localhost/agent/alerts', [
            'name' => 'Scheduled Alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily', 'time_of_day' => '09:00'],
        ]);

        $response->assertCreated();
        $response->assertJsonStructure(['data' => ['next_run_at']]);
        $this->assertNotNull($response->json('data.next_run_at'));
    }

    public function test_process_due_alerts_command_processes_due_alerts(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Due Alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'next_run_at' => now()->subHour(),
        ]);

        AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Future Alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'next_run_at' => now()->addDay(),
        ]);

        $scheduler = app(AlertSchedulerService::class);
        $count = $scheduler->processDueAlerts();

        $this->assertEquals(1, $count);
        $this->assertDatabaseHas('agent_alert_runs', [
            'agent_alert_id' => AgentAlert::query()->where('name', 'Due Alert')->first()->id,
            'status' => 'sent',
        ]);
    }

    public function test_process_due_alerts_executes_linked_saved_search_for_listing_metadata(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['ListingKey' => 'k-a', 'ListPrice' => 400000, 'City' => 'Orlando', 'Latitude' => 28.5, 'Longitude' => -81.4],
                    ['ListingKey' => 'k-b', 'ListPrice' => 500000, 'City' => 'Orlando', 'Latitude' => 28.5, 'Longitude' => -81.4],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Alert linked search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        AgentSearchFilter::query()->create([
            'agent_search_id' => $search->id,
            'canonical_field_key' => 'property.list_price',
            'operator' => 'gte',
            'value_json' => 300000,
        ]);

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'name' => 'Search-driven due',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
            'next_run_at' => now()->subMinute(),
        ]);

        $count = app(AlertSchedulerService::class)->processDueAlerts();
        $this->assertSame(1, $count);

        $run = AgentAlertRun::query()->where('agent_alert_id', $alert->id)->latest('id')->first();
        $this->assertNotNull($run);
        $meta = $run->metadata_json;
        $this->assertIsArray($meta);
        $this->assertArrayHasKey('listing_query', $meta);
        $this->assertTrue($meta['listing_query']['success']);
        $this->assertSame(2, $meta['listing_query']['total_items']);
        $this->assertGreaterThanOrEqual(1, count($meta['listing_query']['sample_listing_ids'] ?? []));
        $this->assertSame(2, $meta['listing_query']['new_match_count']);
        $this->assertCount(2, $meta['listing_query']['listing_ids_snapshot'] ?? []);
    }

    public function test_listing_alert_second_run_counts_only_new_listing_ids(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::sequence()
                ->push([
                    'value' => [
                        ['ListingKey' => 'keep-a', 'ListPrice' => 400000, 'City' => 'Orlando', 'Latitude' => 28.5, 'Longitude' => -81.4],
                        ['ListingKey' => 'keep-b', 'ListPrice' => 410000, 'City' => 'Orlando', 'Latitude' => 28.5, 'Longitude' => -81.4],
                    ],
                ], 200)
                ->push([
                    'value' => [
                        ['ListingKey' => 'keep-b', 'ListPrice' => 410000, 'City' => 'Orlando', 'Latitude' => 28.5, 'Longitude' => -81.4],
                        ['ListingKey' => 'new-c', 'ListPrice' => 420000, 'City' => 'Orlando', 'Latitude' => 28.5, 'Longitude' => -81.4],
                    ],
                ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Diff search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        AgentSearchFilter::query()->create([
            'agent_search_id' => $search->id,
            'canonical_field_key' => 'property.list_price',
            'operator' => 'gte',
            'value_json' => 300000,
        ]);

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'name' => 'Diff alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
            'next_run_at' => now()->subMinute(),
        ]);

        $scheduler = app(AlertSchedulerService::class);
        $this->assertSame(1, $scheduler->processDueAlerts());
        $run1 = AgentAlertRun::query()->where('agent_alert_id', $alert->id)->orderByDesc('id')->first();
        $this->assertNotNull($run1);
        $this->assertSame(2, $run1->metadata_json['listing_query']['new_match_count']);

        AgentAlertRun::query()->where('id', $run1->id)->update(['ran_at' => now()->subHours(2)]);
        $alert->refresh();
        $alert->update(['next_run_at' => now()->subMinute()]);

        $this->assertSame(1, $scheduler->processDueAlerts());
        $run2 = AgentAlertRun::query()->where('agent_alert_id', $alert->id)->orderByDesc('id')->first();
        $this->assertNotNull($run2);
        $this->assertNotSame($run1->id, $run2->id);
        $this->assertSame(1, $run2->metadata_json['listing_query']['new_match_count']);
        $this->assertSame(['new-c'], $run2->metadata_json['listing_query']['new_listing_ids']);
    }

    public function test_process_due_alerts_records_empty_criteria_for_listing_without_filters(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'No filters',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'name' => 'Empty criteria alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
            'next_run_at' => now()->subMinute(),
        ]);

        app(AlertSchedulerService::class)->processDueAlerts();

        $run = AgentAlertRun::query()->where('agent_alert_id', $alert->id)->latest('id')->first();
        $this->assertNotNull($run);
        $lq = $run->metadata_json['listing_query'] ?? null;
        $this->assertIsArray($lq);
        $this->assertFalse($lq['success']);
        $this->assertSame('empty_criteria', $lq['reason']);
    }

    public function test_embed_code_endpoint_returns_valid_snippet(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'widgets', true);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/share-links/embed-code', [
            'widget_type' => 'search',
            'theme' => 'dark',
            'max_listings' => 12,
        ]);

        $response->assertOk();
        $response->assertJsonStructure(['data' => ['embed_code', 'widget_type', 'theme', 'max_listings']]);
        $embedCode = $response->json('data.embed_code');
        $this->assertStringContainsString('data-quantyra-widget="search"', $embedCode);
        $this->assertStringContainsString('data-theme="dark"', $embedCode);
        $this->assertStringContainsString('data-max-listings="12"', $embedCode);
        $this->assertStringContainsString('widget/loader.js', $embedCode);
    }

    public function test_embed_code_with_search_id_includes_data_attribute(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'widgets', true);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/share-links/embed-code', [
            'widget_type' => 'lead_form',
            'agent_search_id' => '42',
        ]);

        $response->assertOk();
        $embedCode = $response->json('data.embed_code');
        $this->assertStringContainsString('data-search-id="42"', $embedCode);
    }

    public function test_embed_code_forbidden_when_widgets_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->postJson('https://localhost/agent/share-links/embed-code', [
            'widget_type' => 'search',
        ]);

        $response->assertForbidden();
        $response->assertJsonPath('module', 'widgets');
    }

    public function test_seo_landings_endpoint_forbidden_when_seo_feature_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/share-links/seo-landings');

        $response->assertForbidden();
        $response->assertJsonPath('module', 'seo_landing_pages');
    }

    public function test_seo_share_link_store_forbidden_when_seo_feature_disabled(): void
    {
        config([
            'idx.platform_url' => 'https://idx.quantyralabs.cc',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Denied SEO Search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/share-links', [
            'agent_search_id' => $search->id,
            'template_kind' => 'seo_landing',
        ]);

        $response->assertForbidden();
        $response->assertJsonPath('module', 'seo_landing_pages');
    }

    public function test_seo_share_link_reactivate_forbidden_when_seo_feature_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'reactivate-seo-denied',
            'attribution_json' => ['template_kind' => 'seo_landing'],
        ]);

        $this->actingAs($user)->putJson('https://localhost/agent/share-links/'.$link->id, [
            'status' => 'inactive',
        ])->assertOk();

        $response = $this->actingAs($user)->putJson('https://localhost/agent/share-links/'.$link->id, [
            'status' => 'active',
        ]);

        $response->assertForbidden();
        $response->assertJsonPath('module', 'seo_landing_pages');
    }

    public function test_seo_share_link_index_forbidden_when_template_kind_seo_and_feature_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/share-links?template_kind=seo_landing');

        $response->assertForbidden();
        $response->assertJsonPath('module', 'seo_landing_pages');
    }

    public function test_seo_share_link_export_csv_forbidden_when_template_kind_seo_and_feature_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/share-links/export.csv?template_kind=seo_landing');

        $response->assertForbidden();
        $response->assertJsonPath('module', 'seo_landing_pages');
    }

    public function test_feature_flags_endpoint_returns_flags(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/settings/feature-flags');
        $response->assertOk();
        $response->assertJsonStructure(['data' => ['flags', 'available_modules', 'global_defaults']]);
        $this->assertContains('search', $response->json('data.available_modules'));
        $this->assertTrue($response->json('data.flags.search'));
    }

    public function test_feature_flags_can_be_updated(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->putJson('https://localhost/agent/settings/feature-flags', [
            'flags' => [
                'search' => true,
                'automations' => true,
                'widgets' => true,
            ],
        ]);

        $response->assertOk();
        $this->assertTrue($response->json('data.flags.search'));
        $this->assertTrue($response->json('data.flags.automations'));
        $this->assertTrue($response->json('data.flags.widgets'));
    }

    public function test_feature_flag_service_is_enabled_check(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $flags = new FeatureFlagService;
        $this->assertTrue($flags->isEnabled($user, 'search'));
        $this->assertFalse($flags->isEnabled($user, 'automations'));

        $flags->setFlag($user, 'automations', true);
        $this->assertTrue($flags->isEnabled($user, 'automations'));
    }

    public function test_module_middleware_blocks_search_routes_when_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        AgentPortalSetting::query()->create([
            'user_id' => $user->id,
            'settings_json' => [
                'feature_flags' => [
                    'search' => false,
                ],
            ],
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/searches');

        $response->assertForbidden();
        $response->assertJsonPath('module', 'search');
    }

    public function test_module_middleware_allows_search_routes_when_enabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        AgentPortalSetting::query()->create([
            'user_id' => $user->id,
            'settings_json' => [
                'feature_flags' => [
                    'search' => true,
                ],
            ],
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/searches');

        $response->assertOk();
        $response->assertJsonStructure(['data']);
    }

    public function test_module_middleware_blocks_automation_routes_when_disabled(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/automations/settings');

        $response->assertForbidden();
        $response->assertJsonPath('module', 'automations');
    }

    public function test_dashboard_summary_respects_period_parameter(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->getJson('https://localhost/agent/dashboard/summary?period=30');
        $response->assertOk();
        $response->assertJsonStructure(['data' => ['kpis', 'activity_feed', 'upcoming_alerts', 'period_days']]);
        $this->assertEquals(30, $response->json('data.period_days'));

        $response7 = $this->actingAs($user)->getJson('https://localhost/agent/dashboard/summary?period=7');
        $response7->assertOk();
        $this->assertEquals(7, $response7->json('data.period_days'));
    }

    public function test_execute_response_includes_cache_hit_meta(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['ListingKey' => 'cache-1', 'ListPrice' => 350000, 'City' => 'Tampa'],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $payload = [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 200000],
            ],
        ];

        $first = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', $payload);
        $first->assertOk();
        $first->assertJsonPath('meta.cache_hit', false);

        $second = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', $payload);
        $second->assertOk();
        $second->assertJsonPath('meta.cache_hit', true);
    }

    public function test_agent_search_execute_returns_422_when_idx_search_revoked_for_all_feeds(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
        ]);

        Http::fake([
            'bridge.test/*' => Http::response([
                'value' => [
                    ['ListingKey' => 'k1', 'ListPrice' => 250000, 'City' => 'Tampa'],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 100000],
            ],
        ])->assertOk();

        SubscriberFeedAccess::query()
            ->where('user_id', $user->id)
            ->update(['permissions_json' => ['idx:search' => false]]);

        $blocked = $this->actingAs($user)->postJson('https://localhost/agent/searches/execute', [
            'filters' => [
                ['field' => 'property.list_price', 'operator' => 'gte', 'value' => 100000],
            ],
        ]);

        $blocked->assertStatus(422);
        $blocked->assertJsonPath('errors.mls_scope.0', 'At least one MLS dataset scope is required.');
    }

    public function test_agent_alert_store_returns_422_when_idx_alerts_revoked(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        app(SubscriberFeedAccessService::class)->resolvedScopesForUser($user);
        SubscriberFeedAccess::query()
            ->where('user_id', $user->id)
            ->update(['permissions_json' => ['idx:search' => true, 'idx:alerts' => false]]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/alerts', [
            'name' => 'Blocked alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
        ]);

        $response->assertStatus(422);
        $response->assertJsonPath('errors.feed_access.0', 'No feeds are enabled for IDX alerts.');
    }

    public function test_agent_alert_resume_returns_422_when_idx_alerts_revoked(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Resume gate',
            'alert_type' => 'listing',
            'status' => 'paused',
            'schedule_json' => ['cadence' => 'daily'],
        ]);

        app(SubscriberFeedAccessService::class)->resolvedScopesForUser($user);
        SubscriberFeedAccess::query()
            ->where('user_id', $user->id)
            ->update(['permissions_json' => ['idx:search' => true, 'idx:alerts' => false]]);

        $resume = $this->actingAs($user)->putJson('https://localhost/agent/alerts/'.$alert->id, [
            'name' => 'Resume gate',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
        ]);

        $resume->assertStatus(422);
        $resume->assertJsonPath('errors.feed_access.0', 'No feeds are enabled for IDX alerts.');
    }

    public function test_process_due_alerts_skips_when_idx_alerts_revoked(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Due but blocked',
            'alert_type' => 'listing',
            'status' => 'active',
            'next_run_at' => now()->subHour(),
        ]);

        app(SubscriberFeedAccessService::class)->resolvedScopesForUser($user);
        SubscriberFeedAccess::query()
            ->where('user_id', $user->id)
            ->update(['permissions_json' => ['idx:search' => true, 'idx:alerts' => false]]);

        $scheduler = app(AlertSchedulerService::class);
        $this->assertSame(0, $scheduler->processDueAlerts());
        $this->assertDatabaseMissing('agent_alert_runs', ['agent_alert_id' => $alert->id]);
    }

    public function test_contacts_bulk_status_updates_multiple_leads(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $lead1 = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Lead A', 'status' => 'new'],
        ]);
        $lead2 = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Lead B', 'status' => 'new'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/contacts/bulk/status', [
            'contact_ids' => [$lead1->id, $lead2->id],
            'status' => 'contacted',
        ]);

        $response->assertOk();
        $this->assertEquals(2, $response->json('data.updated'));

        $lead1->refresh();
        $this->assertEquals('contacted', $lead1->payload['status']);
    }

    public function test_contacts_bulk_delete_removes_leads(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'To Delete'],
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/contacts/bulk/delete', [
            'contact_ids' => [$lead->id],
        ]);

        $response->assertOk();
        $this->assertEquals(1, $response->json('data.deleted'));
        $this->assertDatabaseMissing('quantyra_leads', ['id' => $lead->id]);
    }

    public function test_contacts_export_csv_returns_download(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Export Lead', 'email' => 'export@test.com', 'status' => 'new'],
        ]);

        $response = $this->actingAs($user)->get('https://localhost/agent/contacts/export.csv');
        $response->assertOk();
        $response->assertHeader('content-type', 'text/csv; charset=UTF-8');
    }

    public function test_automation_integration_connect_sets_status(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'automations', true);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/automations/settings/integrations/connect', [
            'provider' => 'ghl',
        ]);

        $response->assertOk();
        $response->assertJsonPath('data.integration_health', 'connected');
        $response->assertJsonPath('data.integrations.ghl.status', 'connected');
    }

    public function test_automation_integration_disconnect_sets_status(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'automations', true);

        $this->actingAs($user)->postJson('https://localhost/agent/automations/settings/integrations/connect', [
            'provider' => 'ghl',
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/automations/settings/integrations/disconnect', [
            'provider' => 'ghl',
        ]);

        $response->assertOk();
        $response->assertJsonPath('data.integration_health', 'disconnected');
        $response->assertJsonPath('data.integrations.ghl.status', 'disconnected');
    }

    public function test_automation_integration_reconnect_updates_timestamp(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'automations', true);

        $this->actingAs($user)->postJson('https://localhost/agent/automations/settings/integrations/connect', [
            'provider' => 'ghl',
        ]);

        $response = $this->actingAs($user)->postJson('https://localhost/agent/automations/settings/integrations/reconnect', [
            'provider' => 'ghl',
        ]);

        $response->assertOk();
        $response->assertJsonPath('data.integrations.ghl.status', 'connected');
        $this->assertNotNull($response->json('data.integrations.ghl.updated_at'));
    }

    public function test_activity_event_is_recorded_on_search_save(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $this->actingAs($user)->postJson('https://localhost/agent/searches', [
            'name' => 'Tracked Search',
            'search_state_json' => ['filters' => []],
        ]);

        $this->assertDatabaseHas('agent_activity_events', [
            'user_id' => $user->id,
            'event_type' => 'save_search',
            'title' => 'Saved search: Tracked Search',
        ]);
    }

    public function test_activity_event_is_recorded_on_alert_create(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $this->actingAs($user)->postJson('https://localhost/agent/alerts', [
            'name' => 'Tracked Alert',
            'alert_type' => 'listing',
            'status' => 'active',
        ]);

        $this->assertDatabaseHas('agent_activity_events', [
            'user_id' => $user->id,
            'event_type' => 'create_alert',
            'title' => 'Created alert: Tracked Alert',
        ]);
    }

    public function test_custom_activity_event_can_be_recorded(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->postJson('https://localhost/agent/dashboard/events', [
            'event_type' => 'apply_filter',
            'title' => 'Applied price filter',
            'metadata' => ['field' => 'ListPrice', 'operator' => 'gte'],
        ]);

        $response->assertCreated();
        $this->assertDatabaseHas('agent_activity_events', [
            'user_id' => $user->id,
            'event_type' => 'apply_filter',
        ]);
    }

    public function test_dashboard_feed_includes_custom_activity_events(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        AgentActivityEvent::query()->create([
            'user_id' => $user->id,
            'event_type' => 'map_draw',
            'title' => 'Drew include polygon',
            'status' => 'completed',
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/dashboard/summary?period=7');
        $response->assertOk();

        $feed = $response->json('data.activity_feed');
        $mapDrawEvents = array_filter($feed, fn ($item) => ($item['type'] ?? '') === 'map_draw');
        $this->assertNotEmpty($mapDrawEvents);
    }

    public function test_dashboard_feed_includes_alert_run_events(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Feed Run Alert',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
            'next_run_at' => now()->addDay(),
        ]);

        AgentAlertRun::query()->create([
            'agent_alert_id' => $alert->id,
            'status' => 'sent',
            'metadata_json' => ['triggered_by' => 'scheduler'],
            'ran_at' => now()->subHours(2),
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/dashboard/summary?period=7');
        $response->assertOk();

        $feed = $response->json('data.activity_feed');
        $runs = array_values(array_filter($feed, fn (array $item): bool => ($item['type'] ?? '') === 'alert_run'));
        $this->assertNotEmpty($runs);
        $this->assertStringContainsString('Alert sent:', (string) ($runs[0]['title'] ?? ''));
        $this->assertSame('sent', $runs[0]['status'] ?? null);
        $this->assertSame('listing', $runs[0]['alert_type'] ?? null);
        $this->assertSame($alert->id, $runs[0]['alert_id'] ?? null);
        $this->assertSame(0, (int) ($runs[0]['new_match_count'] ?? -1));
    }

    public function test_dashboard_feed_shows_new_match_title_when_listing_alert_run_has_new_matches(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $alert = AgentAlert::query()->create([
            'user_id' => $user->id,
            'name' => 'Hot listings',
            'alert_type' => 'listing',
            'status' => 'active',
            'schedule_json' => ['cadence' => 'daily'],
            'next_run_at' => now()->addDay(),
        ]);

        AgentAlertRun::query()->create([
            'agent_alert_id' => $alert->id,
            'status' => 'sent',
            'metadata_json' => [
                'triggered_by' => 'scheduler',
                'cadence' => 'daily',
                'listing_query' => [
                    'success' => true,
                    'new_match_count' => 4,
                ],
            ],
            'ran_at' => now()->subHours(2),
        ]);

        $response = $this->actingAs($user)->getJson('https://localhost/agent/dashboard/summary?period=7');
        $response->assertOk();

        $feed = $response->json('data.activity_feed');
        $runs = array_values(array_filter($feed, fn (array $item): bool => ($item['type'] ?? '') === 'alert_run'));
        $this->assertNotEmpty($runs);
        $this->assertStringContainsString('4 new listing match', (string) ($runs[0]['title'] ?? ''));
        $this->assertSame(4, (int) ($runs[0]['new_match_count'] ?? 0));
    }

    public function test_contact_tags_can_be_synced_and_retrieved(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $lead = QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Tagged Contact'],
        ]);

        $sync = $this->actingAs($user)->putJson("https://localhost/agent/contacts/{$lead->id}/tags", [
            'tags' => ['buyer', 'vip', 'buyer'],
        ]);
        $sync->assertOk();
        $sync->assertJsonPath('data.contact_id', $lead->id);
        $sync->assertJsonCount(2, 'data.tags');
        $this->assertDatabaseHas('agent_contact_tags', [
            'user_id' => $user->id,
            'lead_id' => (string) $lead->id,
            'tag' => 'buyer',
        ]);

        $show = $this->actingAs($user)->getJson("https://localhost/agent/contacts/{$lead->id}/tags");
        $show->assertOk();
        $show->assertJsonPath('data.contact_id', $lead->id);
        $show->assertJsonFragment(['buyer']);
        $show->assertJsonFragment(['vip']);
    }

    public function test_contacts_tab_filters_support_nurtured_awaiting_and_archive(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Nurtured Lead', 'status' => 'hot', 'lead_score' => 95],
        ]);
        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Awaiting Lead', 'status' => 'new'],
        ]);
        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Archived Lead', 'status' => 'archived'],
        ]);

        $nurtured = $this->actingAs($user)->getJson('https://localhost/agent/contacts?tab=nurtured');
        $nurtured->assertOk();
        $nurtured->assertJsonPath('data.meta.total', 1);
        $nurtured->assertJsonPath('data.items.0.payload.name', 'Nurtured Lead');

        $awaiting = $this->actingAs($user)->getJson('https://localhost/agent/contacts?tab=awaiting');
        $awaiting->assertOk();
        $awaiting->assertJsonPath('data.meta.total', 1);
        $awaiting->assertJsonPath('data.items.0.payload.name', 'Awaiting Lead');

        $archive = $this->actingAs($user)->getJson('https://localhost/agent/contacts?tab=archive');
        $archive->assertOk();
        $archive->assertJsonPath('data.meta.total', 1);
        $archive->assertJsonPath('data.items.0.payload.name', 'Archived Lead');
    }

    public function test_contacts_filtered_index_records_activity_only_on_first_page(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        QuantyraLead::query()->create([
            'ghl_location_id' => 'direct-'.$user->id,
            'lead_type' => 'buyer',
            'source' => 'widget',
            'payload' => ['name' => 'Tracked Tab', 'status' => 'hot'],
        ]);

        $this->assertDatabaseCount('agent_activity_events', 0);

        $this->actingAs($user)->getJson('https://localhost/agent/contacts?tab=nurtured')->assertOk();

        $this->assertDatabaseCount('agent_activity_events', 1);
        $this->assertDatabaseHas('agent_activity_events', [
            'user_id' => $user->id,
            'event_type' => 'apply_contacts_filter',
        ]);

        $this->actingAs($user)->getJson('https://localhost/agent/contacts?tab=nurtured&page=2')->assertOk();
        $this->assertDatabaseCount('agent_activity_events', 1);
    }

    public function test_contacts_unfiltered_index_does_not_record_filter_activity(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $this->actingAs($user)->getJson('https://localhost/agent/contacts')->assertOk();

        $this->assertDatabaseMissing('agent_activity_events', [
            'user_id' => $user->id,
            'event_type' => 'apply_contacts_filter',
        ]);
    }

    public function test_agent_portal_home_value_comps_returns_estimate_with_bridge_fakes(): void
    {
        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.dataset' => 'stellar',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'geocoding.google_api_key' => 'test-google-geocoding-key',
        ]);

        $comps = [];
        for ($i = 0; $i < 6; $i++) {
            $row = $this->portalClosedCompListing('stellar:hv'.$i, $i * 0.003);
            $row['LivingArea'] = 1600 + ($i * 100);
            $row['ClosePrice'] = 320000 + ($i * 25000);
            $row['PoolPrivateYN'] = $i % 2 === 0;
            $row['GarageSpaces'] = 2;
            $comps[] = $row;
        }

        Http::fake([
            'maps.googleapis.com/*' => Http::response([
                'status' => 'OK',
                'results' => [[
                    'geometry' => ['location' => ['lat' => 27.95, 'lng' => -82.45]],
                    'formatted_address' => '100 Main St, Tampa, FL 33602, USA',
                    'place_id' => 'test_place_id',
                ]],
            ], 200),
            'bridge.test/*' => Http::response(['value' => $comps], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);
        app(FeatureFlagService::class)->setFlag($user, 'marketing', true);

        $body = [
            'subject' => [
                'address' => '100 Main St, Tampa, FL 33602',
                'condition' => 'good',
                'property_type' => 'sfr',
                'bedrooms' => 3,
                'full_bathrooms' => 2,
                'half_bathrooms' => 0,
                'living_area_sqft' => 1800,
                'year_built' => 2010,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 5],
            'home_value_params' => ['sold_months_back' => 12, 'max_comps' => 6],
        ];

        $response = $this->actingAs($user)->postJson('https://localhost/agent/comps/run?dataset=stellar', $body);

        $response->assertOk();
        $response->assertJsonPath('success', true);
        $this->assertNotNull($response->json('home_value_result'));
        $this->assertNotNull($response->json('home_value_result.point_estimate'));
        $this->assertNotNull($response->json('home_value_result.range.low'));
        $this->assertNotNull($response->json('home_value_result.range.high'));
    }

    public function test_agent_comps_run_rejects_dataset_not_in_subscriber_scope(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);
        app(FeatureFlagService::class)->setFlag($user, 'marketing', true);

        $body = [
            'subject' => [
                'address' => '100 Main St, Tampa, FL',
                'half_bathrooms' => 0,
            ],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 3],
            'filters' => [
                'sold_months_back' => 12,
                'max_sold_comps' => 10,
                'living_area_pct' => 20,
                'beds_tolerance' => 1,
                'baths_tolerance' => 1,
                'year_built_tolerance' => 15,
            ],
            'home_value_params' => ['sold_months_back' => 12, 'max_comps' => 8],
        ];

        Http::fake();

        $response = $this->actingAs($user)->postJson('https://localhost/agent/comps/run?dataset=not-your-dataset', $body);
        $response->assertStatus(422);
        $response->assertJsonPath('errors.dataset.0', 'That MLS dataset is not in your IDX access list.');
        Http::assertNothingSent();
    }

    public function test_agent_comps_run_forbidden_when_marketing_module_disabled(): void
    {
        Http::fake();

        /** @var User $user */
        $user = User::factory()->createOne();
        app(FeatureFlagService::class)->setFlag($user, 'marketing', false);

        $minimal = [
            'subject' => ['address' => 'X', 'half_bathrooms' => 0],
            'mode' => 'home_value',
            'scope' => ['type' => 'radius', 'radius_miles' => 3],
            'filters' => [
                'sold_months_back' => 12,
                'max_sold_comps' => 10,
                'living_area_pct' => 20,
                'beds_tolerance' => 1,
                'baths_tolerance' => 1,
                'year_built_tolerance' => 15,
            ],
            'home_value_params' => ['sold_months_back' => 12, 'max_comps' => 8],
        ];

        $this->actingAs($user)->postJson('https://localhost/agent/comps/run', $minimal)->assertForbidden();
        Http::assertNothingSent();
    }

    /**
     * @return array<string, mixed>
     */
    private function portalClosedCompListing(string $key, float $distLatOffset = 0.01): array
    {
        return [
            'ListingKey' => $key,
            'StandardStatus' => 'Closed',
            'ListPrice' => 400000,
            'ClosePrice' => 395000,
            'OriginalListPrice' => 410000,
            'PreviousListPrice' => 405000,
            'CloseDate' => now()->subMonths(2)->format('Y-m-d'),
            'OnMarketDate' => now()->subMonths(3)->format('Y-m-d'),
            'BedroomsTotal' => 3,
            'BathroomsTotalDecimal' => 2,
            'LivingArea' => 1750,
            'LotSizeAcres' => 0.22,
            'YearBuilt' => 2008,
            'StoriesTotal' => 1,
            'City' => 'Tampa',
            'StateOrProvince' => 'FL',
            'PostalCode' => '33602',
            'CountyOrParish' => 'Hillsborough',
            'PropertyType' => 'Residential',
            'PropertySubType' => 'Single Family Residence',
            'WaterfrontYN' => false,
            'ViewYN' => false,
            'PoolPrivateYN' => true,
            'GarageYN' => true,
            'GarageSpaces' => 1,
            'CarportSpaces' => 0,
            'CoveredSpaces' => 0,
            'OpenParkingSpaces' => null,
            'AssociationYN' => false,
            'SeniorCommunityYN' => false,
            'StreetNumber' => '200',
            'StreetName' => 'Oak',
            'SubdivisionName' => 'Portal Comps Subdivision',
            'MLSAreaMajor' => '33602 - Tampa',
            'Coordinates' => ['coordinates' => [-82.45, 27.95 + $distLatOffset]],
            'DaysOnMarket' => 30,
            'CumulativeDaysOnMarket' => 30,
            'PublicRemarks' => 'Move-in ready.',
            'STELLAR_FloodZoneCode' => 'X',
            'STELLAR_TotalMonthlyFees' => 0,
        ];
    }
}
