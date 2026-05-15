<?php

namespace Tests\Feature;

use App\Models\User;
use App\Models\UserInvitation;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class TurnstileAuthTest extends TestCase
{
    use RefreshDatabase;

    private const string PLATFORM_BASE = 'https://dev-idx.quantyralabs.cc';

    protected function setUp(): void
    {
        parent::setUp();

        $this->withoutVite();

        config([
            'turnstile.site_key' => 'test-site-key',
            'turnstile.secret_key' => 'test-secret-key',
        ]);
    }

    public function test_login_rejects_missing_turnstile_when_enabled(): void
    {
        $user = User::factory()->createOne();

        $response = $this->post(self::PLATFORM_BASE.'/login', [
            'email' => $user->email,
            'password' => 'password',
        ]);

        $response->assertSessionHasErrors('cf-turnstile-response');
        $this->assertGuest();
    }

    public function test_login_accepts_valid_turnstile_token(): void
    {
        Http::fake([
            'challenges.cloudflare.com/*' => Http::response(['success' => true]),
        ]);

        $user = User::factory()->createOne();

        $response = $this->post(self::PLATFORM_BASE.'/login', [
            'email' => $user->email,
            'password' => 'password',
            'cf-turnstile-response' => 'valid-token',
        ]);

        $response->assertRedirect('/dashboard');
        $this->assertAuthenticatedAs($user);
    }

    public function test_forgot_password_rejects_invalid_turnstile(): void
    {
        Http::fake([
            'challenges.cloudflare.com/*' => Http::response(['success' => false]),
        ]);

        $response = $this->post(self::PLATFORM_BASE.'/forgot-password', [
            'email' => 'someone@example.com',
            'cf-turnstile-response' => 'bad-token',
        ]);

        $response->assertSessionHasErrors('cf-turnstile-response');
    }

    public function test_register_requires_turnstile_when_enabled(): void
    {
        $admin = User::factory()->admin()->createOne();
        $email = 'turnstile-'.uniqid('', true).'@example.com';
        $created = UserInvitation::createWithPlainToken([
            'email' => $email,
            'invited_by' => $admin->id,
            'expires_at' => now()->addDay(),
        ]);

        $response = $this->post(self::PLATFORM_BASE.'/register', [
            'invitation_token' => $created['plain'],
            'name' => 'Invited',
            'email' => $email,
            'domain_slug' => 'slug-'.uniqid('', false).'.example.com',
            'dataset' => 'stellar',
            'mls_id' => 'MLS12345',
            'mls_email' => $email,
            'password' => 'Password-1-Strong!',
            'password_confirmation' => 'Password-1-Strong!',
        ]);

        $response->assertSessionHasErrors('cf-turnstile-response');
    }

    public function test_turnstile_skipped_when_secret_not_configured(): void
    {
        config(['turnstile.secret_key' => null]);

        $user = User::factory()->createOne();

        $response = $this->post(self::PLATFORM_BASE.'/login', [
            'email' => $user->email,
            'password' => 'password',
        ]);

        $response->assertRedirect('/dashboard');
        $this->assertAuthenticatedAs($user);
    }
}
