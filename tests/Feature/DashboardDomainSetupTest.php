<?php

namespace Tests\Feature;

use App\Models\Domain;
use App\Models\User;
use App\Services\DomainOwnershipVerifier;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

class DashboardDomainSetupTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();
        $this->withoutVite();
    }

    public function test_store_domain_with_mls_selection_persists_allowed_datasets(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->post('/dashboard/domains', [
            'domain_slug' => 'example.com',
            'allowed_mls_datasets' => ['bridge_stellar'],
            'mls_dataset' => 'bridge_stellar',
        ]);

        $response->assertRedirect('/dashboard');
        $response->assertSessionHas('dashboard_status');

        $domain = Domain::query()->where('domain_slug', 'example.com')->first();
        $this->assertNotNull($domain);
        $this->assertSame(['bridge_stellar'], $domain->getAllowedMlsDatasets());
        $this->assertSame('bridge_stellar', $domain->getMlsDataset());
    }

    public function test_store_domain_requires_at_least_one_mls_feed(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->post('/dashboard/domains', [
            'domain_slug' => 'example.com',
            'allowed_mls_datasets' => [],
        ]);

        $response->assertSessionHasErrors('allowed_mls_datasets');
    }

    public function test_store_domain_defaults_mls_to_first_selected(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->post('/dashboard/domains', [
            'domain_slug' => 'example.com',
            'allowed_mls_datasets' => ['bridge_stellar'],
        ]);

        $response->assertRedirect('/dashboard');

        $domain = Domain::query()->where('domain_slug', 'example.com')->first();
        $this->assertNotNull($domain);
        $this->assertSame('bridge_stellar', $domain->getMlsDataset());
    }

    public function test_verify_txt_auto_creates_production_token_on_first_verification(): void
    {
        $user = User::factory()->createOne();
        $domain = Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'example.com',
            'is_active' => true,
            'verification_status' => 'pending',
            'txt_verification_name' => '_geoidx.example.com',
            'txt_verification_value' => 'geoidx-verify=abc123',
            'mls_dataset' => 'bridge_stellar',
            'allowed_mls_datasets' => ['bridge_stellar'],
        ]);

        $this->bindVerifierSpy(verifyResult: true);

        $response = $this->actingAs($user)
            ->post("/dashboard/domains/{$domain->id}/verify-txt");

        $response->assertRedirect('/dashboard');
        $response->assertSessionHas('dashboard_new_api_token');

        $this->assertEquals(1, $user->fresh()->tokens()->count());
        $token = $user->fresh()->tokens()->first();
        $this->assertSame('Production', $token->name);
    }

    public function test_verify_txt_does_not_create_second_production_token(): void
    {
        $user = User::factory()->createOne();
        $user->createToken('Production', ['idx:full']);

        $domain = Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'example.com',
            'is_active' => true,
            'verification_status' => 'pending',
            'txt_verification_name' => '_geoidx.example.com',
            'txt_verification_value' => 'geoidx-verify=abc123',
            'mls_dataset' => 'bridge_stellar',
            'allowed_mls_datasets' => ['bridge_stellar'],
        ]);

        $this->bindVerifierSpy(verifyResult: true);

        $response = $this->actingAs($user)
            ->post("/dashboard/domains/{$domain->id}/verify-txt");

        $response->assertRedirect('/dashboard');
        $response->assertSessionMissing('dashboard_new_api_token');

        $this->assertEquals(1, $user->fresh()->tokens()->count());
    }

    public function test_verify_txt_does_not_create_token_on_failure(): void
    {
        $user = User::factory()->createOne();
        $domain = Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'example.com',
            'is_active' => true,
            'verification_status' => 'pending',
            'txt_verification_name' => '_geoidx.example.com',
            'txt_verification_value' => 'geoidx-verify=abc123',
            'mls_dataset' => 'bridge_stellar',
            'allowed_mls_datasets' => ['bridge_stellar'],
        ]);

        $this->bindVerifierSpy(verifyResult: false);

        $response = $this->actingAs($user)
            ->post("/dashboard/domains/{$domain->id}/verify-txt");

        $response->assertRedirect('/dashboard');
        $response->assertSessionMissing('dashboard_new_api_token');

        $this->assertEquals(0, $user->fresh()->tokens()->count());
    }

    public function test_staging_api_token_route_creates_staging_token(): void
    {
        $user = User::factory()->createOne();
        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'example.com',
            'is_active' => true,
            'verification_status' => 'verified',
            'verification_method' => 'txt',
            'mls_dataset' => 'bridge_stellar',
            'allowed_mls_datasets' => ['bridge_stellar'],
        ]);

        $response = $this->actingAs($user)
            ->post('/dashboard/api-tokens/staging');

        $response->assertRedirect('/dashboard');
        $response->assertSessionHas('dashboard_new_api_token');

        $token = $user->fresh()->tokens()->first();
        $this->assertSame('Staging', $token->name);
    }

    public function test_staging_api_token_rejects_duplicate_staging_name(): void
    {
        $user = User::factory()->createOne();
        $user->createToken('Staging', ['idx:full']);

        $response = $this->actingAs($user)
            ->post('/dashboard/api-tokens/staging');

        $response->assertRedirect('/dashboard');
        $response->assertSessionMissing('dashboard_new_api_token');
        $response->assertSessionHas('dashboard_status');

        $this->assertEquals(1, $user->fresh()->tokens()->count());
    }

    public function test_setup_page_shows_register_phase_for_new_user(): void
    {
        $user = User::factory()->createOne();

        $response = $this->actingAs($user)->get('/dashboard');

        $response->assertOk();
        $response->assertSee('Add your site', false);
    }

    public function test_setup_page_shows_verify_phase_with_pending_domain(): void
    {
        $user = User::factory()->createOne();
        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'example.com',
            'is_active' => true,
            'verification_status' => 'pending',
            'txt_verification_name' => '_geoidx.example.com',
            'txt_verification_value' => 'geoidx-verify=test',
            'mls_dataset' => 'bridge_stellar',
            'allowed_mls_datasets' => ['bridge_stellar'],
        ]);

        $response = $this->actingAs($user)->get('/dashboard');

        $response->assertOk();
        $response->assertSee('Verify ownership', false);
        $response->assertSee('PENDING', false);
    }

    public function test_setup_page_shows_connect_phase_with_verified_domain(): void
    {
        $user = User::factory()->createOne();
        Domain::query()->create([
            'user_id' => $user->id,
            'domain_slug' => 'example.com',
            'is_active' => true,
            'verification_status' => 'verified',
            'verification_method' => 'txt',
            'mls_dataset' => 'bridge_stellar',
            'allowed_mls_datasets' => ['bridge_stellar'],
        ]);

        $response = $this->actingAs($user)->get('/dashboard');

        $response->assertOk();
        $response->assertSee('Connect your app', false);
        $response->assertSee('Production key', false);
        $response->assertSee('Staging key', false);
    }

    private function bindVerifierSpy(bool $verifyResult): void
    {
        $this->app->instance(DomainOwnershipVerifier::class, new class($verifyResult) extends DomainOwnershipVerifier
        {
            public function __construct(private bool $shouldVerify) {}

            public function issueTxtChallenge(Domain $domain): Domain
            {
                return $domain;
            }

            public function verifyTxtRecord(Domain $domain): bool
            {
                if ($this->shouldVerify) {
                    $domain->forceFill([
                        'verification_status' => 'verified',
                        'verification_method' => 'txt',
                        'txt_verified_at' => now(),
                        'verification_checked_at' => now(),
                    ])->save();
                }

                return $this->shouldVerify;
            }
        });
    }
}
