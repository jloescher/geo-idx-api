<?php

namespace Tests\Feature;

use App\Models\AgentShareLink;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class AgentShareLinkPruneCommandTest extends TestCase
{
    use RefreshDatabase;

    public function test_prune_command_uses_configured_default_days_when_option_missing(): void
    {
        config(['agent_portal.share_links.prune_days' => 30]);

        /** @var User $user */
        $user = User::factory()->createOne();

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'default-prune-old',
            'expires_at' => now()->subDays(35),
        ]);

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'default-prune-keep',
            'expires_at' => now()->subDays(10),
        ]);

        $this->artisan('agent:prune-share-links')
            ->assertSuccessful();

        $this->assertDatabaseMissing('agent_share_links', ['token' => 'default-prune-old']);
        $this->assertDatabaseHas('agent_share_links', ['token' => 'default-prune-keep']);
    }

    public function test_prune_command_removes_only_old_inactive_share_links(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'active-keep',
            'expires_at' => null,
        ]);

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'inactive-recent-keep',
            'expires_at' => now()->subDays(10),
        ]);

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'inactive-old-prune',
            'expires_at' => now()->subDays(120),
        ]);

        $this->artisan('agent:prune-share-links', ['--days' => 90])
            ->assertSuccessful();

        $this->assertDatabaseHas('agent_share_links', ['token' => 'active-keep']);
        $this->assertDatabaseHas('agent_share_links', ['token' => 'inactive-recent-keep']);
        $this->assertDatabaseMissing('agent_share_links', ['token' => 'inactive-old-prune']);
    }

    public function test_prune_command_dry_run_does_not_delete_and_reports_count(): void
    {
        /** @var User $user */
        $user = User::factory()->createOne();

        AgentShareLink::query()->create([
            'user_id' => $user->id,
            'token' => 'dry-run-old',
            'expires_at' => now()->subDays(120),
        ]);

        $this->artisan('agent:prune-share-links', [
            '--days' => 90,
            '--dry-run' => true,
        ])
            ->assertSuccessful()
            ->expectsOutputToContain('Dry run: would prune 1 inactive share link');

        $this->assertDatabaseHas('agent_share_links', ['token' => 'dry-run-old']);
    }

    public function test_prune_command_fails_when_days_is_not_positive(): void
    {
        $this->artisan('agent:prune-share-links', ['--days' => 0])
            ->assertFailed()
            ->expectsOutputToContain('The --days option must be greater than zero.');
    }
}
