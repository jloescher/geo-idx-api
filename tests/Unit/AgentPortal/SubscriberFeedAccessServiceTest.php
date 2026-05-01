<?php

namespace Tests\Unit\AgentPortal;

use App\Models\SubscriberFeedAccess;
use App\Models\User;
use App\Services\AgentPortal\SubscriberFeedAccessService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class SubscriberFeedAccessServiceTest extends TestCase
{
    use RefreshDatabase;

    public function test_permissions_allow_idx_search_defaults_to_true(): void
    {
        $this->assertTrue(SubscriberFeedAccessService::permissionsAllowIdxSearch(null));
        $this->assertTrue(SubscriberFeedAccessService::permissionsAllowIdxSearch([]));
        $this->assertTrue(SubscriberFeedAccessService::permissionsAllowIdxSearch(['idx:search' => true]));
    }

    public function test_permissions_allow_idx_search_denies_explicit_false(): void
    {
        $this->assertFalse(SubscriberFeedAccessService::permissionsAllowIdxSearch(['idx:search' => false]));
    }

    public function test_permissions_allow_idx_alerts_defaults_to_true(): void
    {
        $this->assertTrue(SubscriberFeedAccessService::permissionsAllowIdxAlerts(null));
        $this->assertTrue(SubscriberFeedAccessService::permissionsAllowIdxAlerts([]));
        $this->assertTrue(SubscriberFeedAccessService::permissionsAllowIdxAlerts(['idx:alerts' => true]));
    }

    public function test_permissions_allow_idx_alerts_denies_explicit_false(): void
    {
        $this->assertFalse(SubscriberFeedAccessService::permissionsAllowIdxAlerts(['idx:alerts' => false]));
    }

    public function test_resolved_search_scopes_excludes_feeds_without_idx_search(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $feeds = app(SubscriberFeedAccessService::class);
        $feeds->resolvedScopesForUser($user);

        SubscriberFeedAccess::query()
            ->where('user_id', $user->id)
            ->update(['permissions_json' => ['idx:search' => false, 'idx:alerts' => true]]);

        $searchScopes = $feeds->resolvedSearchScopesForUser($user);
        $this->assertSame([], $searchScopes);

        $alertScopes = $feeds->resolvedAlertScopesForUser($user);
        $this->assertCount(1, $alertScopes);

        SubscriberFeedAccess::query()
            ->where('user_id', $user->id)
            ->update(['permissions_json' => ['idx:search' => true, 'idx:alerts' => false]]);

        $this->assertSame([], $feeds->resolvedAlertScopesForUser($user));

        $allScopes = $feeds->resolvedScopesForUser($user);
        $this->assertCount(1, $allScopes);
        $this->assertSame('stellar', $allScopes[0]['dataset_code']);
    }

    public function test_sync_preserves_permissions_json_per_feed_dataset_key(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne([
            'assigned_mls_datasets' => ['stellar'],
        ]);

        $feeds = app(SubscriberFeedAccessService::class);
        $feeds->resolvedScopesForUser($user);

        SubscriberFeedAccess::query()
            ->where('user_id', $user->id)
            ->update(['permissions_json' => ['idx:search' => false, 'idx:alerts' => true]]);

        $feeds->resolvedScopesForUser($user);

        $row = SubscriberFeedAccess::query()->where('user_id', $user->id)->first();
        $this->assertNotNull($row);
        $this->assertFalse((bool) ($row->permissions_json['idx:search'] ?? true));
        $this->assertTrue((bool) ($row->permissions_json['idx:alerts'] ?? false));
    }
}
