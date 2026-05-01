<?php

namespace Tests\Feature;

use App\Models\AgentSearch;
use App\Models\AgentSearchFilter;
use App\Models\AgentShareLink;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class AgentShareLinkPublicTest extends TestCase
{
    use RefreshDatabase;

    public function test_public_shared_link_page_loads_and_records_visit_attribution(): void
    {
        config([
            'idx.platform_url' => 'https://idx.quantyralabs.cc',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Shared Tampa search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'token' => 'sharedtoken123',
            'attribution_json' => [
                'utm_source' => 'newsletter',
            ],
        ]);

        $response = $this->get('https://localhost/shared/'.$link->token.'/shared-tampa-search?utm_campaign=spring');

        $response->assertOk();
        $response->assertViewIs('marketing.shared-link');
        $response->assertViewHas('shareLink');
        $response->assertViewHas('search');
        $response->assertSee('Shared Tampa search');

        /** @var AgentShareLink $fresh */
        $fresh = AgentShareLink::query()->findOrFail($link->id);
        $this->assertSame('newsletter', $fresh->attribution_json['utm_source'] ?? null);
        $this->assertSame('spring', $fresh->attribution_json['utm_campaign'] ?? null);
        $this->assertSame(1, $fresh->attribution_json['visit_count'] ?? null);
        $this->assertArrayHasKey('last_visited_at', (array) $fresh->attribution_json);
    }

    public function test_expired_shared_link_returns_not_found(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'expiredtoken123',
            'expires_at' => now()->subMinute(),
        ]);

        $this->get('https://localhost/shared/'.$link->token)
            ->assertNotFound();
    }

    public function test_public_execute_uses_saved_search_filters(): void
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
                        'ListingKey' => 'stellar:sample',
                        'StandardStatus' => 'Active',
                        'ListPrice' => 520000,
                        'Coordinates' => ['coordinates' => [-82.45, 27.95]],
                    ],
                ],
            ], 200),
        ]);

        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Public execution search',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        AgentSearchFilter::query()->create([
            'agent_search_id' => $search->id,
            'canonical_field_key' => 'property.list_price',
            'operator' => 'gte',
            'value_json' => 500000,
        ]);

        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'token' => 'publicexecutetoken',
        ]);

        $response = $this->postJson('https://localhost/shared/'.$link->token.'/execute');

        $response->assertOk();
        $response->assertJsonPath('data.meta.total_items', 1);
    }

    public function test_shared_link_redirects_to_canonical_slug_when_incorrect(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Luxury Tampa Waterfront',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'token' => 'canonicaltoken123',
        ]);

        $response = $this->get('https://localhost/shared/'.$link->token.'/wrong-slug');
        $response->assertRedirect('https://localhost/shared/'.$link->token.'/luxury-tampa-waterfront');
    }

    public function test_shared_link_view_contains_canonical_tag_for_slugged_url(): void
    {
        config([
            'idx.platform_url' => 'https://idx.quantyralabs.cc',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Downtown Condos',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'token' => 'canonicalviewtoken',
        ]);

        $response = $this->get('https://localhost/shared/'.$link->token.'/downtown-condos');
        $response->assertOk();
        $response->assertSee('<link rel="canonical" href="https://idx.quantyralabs.cc/shared/canonicalviewtoken/downtown-condos"', false);
    }

    public function test_shared_link_view_uses_noindex_for_non_utm_query_variants(): void
    {
        config([
            'idx.platform_url' => 'https://idx.quantyralabs.cc',
        ]);

        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'Seminole Heights Homes',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);

        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'token' => 'noindextoken',
        ]);

        $response = $this->get('https://localhost/shared/'.$link->token.'/seminole-heights-homes?page=2');
        $response->assertOk();
        $response->assertSee('name="robots" content="noindex,follow"', false);
        $response->assertSee('<link rel="canonical" href="https://idx.quantyralabs.cc/shared/noindextoken/seminole-heights-homes"', false);
    }

    public function test_shared_link_view_contains_seo_title_and_description(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $search = AgentSearch::query()->create([
            'user_id' => $user->id,
            'name' => 'South Tampa New Construction',
            'search_state_json' => [],
            'mls_scope_json' => [],
            'is_template' => false,
            'source' => 'manual',
        ]);
        AgentSearchFilter::query()->create([
            'agent_search_id' => $search->id,
            'canonical_field_key' => 'location.city',
            'operator' => 'eq',
            'value_json' => 'Tampa',
        ]);

        $link = AgentShareLink::query()->create([
            'user_id' => $user->id,
            'agent_search_id' => $search->id,
            'token' => 'seometatoken',
        ]);

        $response = $this->get('https://localhost/shared/'.$link->token.'/south-tampa-new-construction');
        $response->assertOk();
        $response->assertSee('South Tampa New Construction | Quantyra IDX');
        $response->assertSee('<meta name="description" content="', false);
        $response->assertSee('Browse shared listings', false);
    }
}
