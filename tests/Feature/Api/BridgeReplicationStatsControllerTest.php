<?php

namespace Tests\Feature\Api;

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Cashier\Subscription;
use Laravel\Cashier\SubscriptionItem;
use Tests\TestCase;

class BridgeReplicationStatsControllerTest extends TestCase
{
    use RefreshDatabase;

    private function tokenForMega(User $user): string
    {
        $subscription = Subscription::query()->create([
            'user_id' => $user->id,
            'type' => 'default',
            'stripe_id' => 'sub_test_'.$user->id,
            'stripe_status' => 'active',
            'stripe_price' => 'price_mega_monthly',
            'quantity' => 1,
            'trial_ends_at' => now()->addDays(7),
            'ends_at' => null,
        ]);

        config(['billing.plans.mega.stripe_price_monthly' => 'price_mega_monthly']);

        SubscriptionItem::query()->create([
            'subscription_id' => $subscription->id,
            'stripe_id' => 'si_test_'.$user->id,
            'stripe_product' => 'prod_test',
            'stripe_price' => 'price_mega_monthly',
            'quantity' => 100,
        ]);

        return $user->createToken('t', ['idx:full'])->plainTextToken;
    }

    public function test_bridge_stats_returns_dataset_payload(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();
        $token = $this->tokenForMega($user);

        config(['bridge.datasets' => ['stellar']]);

        $response = $this->withHeader('Authorization', 'Bearer '.$token)
            ->getJson('/api/v1/bridge/stats');

        $response->assertOk();
        $response->assertJsonStructure([
            'datasets' => [
                [
                    'slug',
                    'listing_count_total',
                    'listing_count_active_pending',
                    'last_bridge_modification_timestamp',
                    'last_sync_finished_at',
                    'replication_in_progress',
                ],
            ],
        ]);
    }
}
