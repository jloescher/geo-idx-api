<?php

namespace Tests\Feature;

use App\Mail\UserInvitationMail;
use App\Models\User;
use App\Models\UserInvitation;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Mail;
use Tests\TestCase;

class InviteOnlyRegistrationTest extends TestCase
{
    use RefreshDatabase;

    private const string PLATFORM_BASE = 'https://dev-idx.quantyralabs.cc';

    protected function setUp(): void
    {
        parent::setUp();

        $this->withoutVite();
    }

    public function test_invalid_invitation_token_redirects_to_login_with_message(): void
    {
        $response = $this->get(self::PLATFORM_BASE.'/register/'.str_repeat('a', 64));

        $response->assertRedirect(self::PLATFORM_BASE.'/login');
        $response->assertSessionHas('status');
    }

    public function test_invited_user_can_register_and_invitation_is_accepted(): void
    {
        $admin = User::factory()->admin()->createOne();
        $email = 'invitee-'.uniqid('', true).'@example.com';
        $domainSlug = 'invited-'.uniqid('', false).'.example.com';

        $created = UserInvitation::createWithPlainToken([
            'email' => $email,
            'invited_by' => $admin->id,
            'expires_at' => now()->addDay(),
        ]);
        $plain = $created['plain'];

        $this->get(self::PLATFORM_BASE.'/register/'.$plain)->assertOk()->assertSee($email, false);

        $response = $this->post(self::PLATFORM_BASE.'/register', [
            'invitation_token' => $plain,
            'name' => 'Invited User',
            'email' => $email,
            'domain_slug' => $domainSlug,
            'dataset' => 'stellar',
            'mls_id' => 'MLS12345',
            'mls_email' => $email,
            'password' => 'Password-1-Strong!',
            'password_confirmation' => 'Password-1-Strong!',
        ]);

        $response->assertRedirect('/dashboard');

        $this->assertDatabaseHas('users', ['email' => $email]);
        $this->assertNotNull(UserInvitation::query()->where('email', $email)->value('accepted_at'));
        $this->assertAuthenticated();
    }

    public function test_register_rejects_invalid_invitation_token(): void
    {
        $response = $this->post(self::PLATFORM_BASE.'/register', [
            'invitation_token' => str_repeat('a', 64),
            'name' => 'X',
            'email' => 'x@example.com',
            'domain_slug' => 'x-'.uniqid('', false).'.example.com',
            'dataset' => 'stellar',
            'mls_id' => 'MLS12345',
            'mls_email' => 'x@example.com',
            'password' => 'Password-1-Strong!',
            'password_confirmation' => 'Password-1-Strong!',
        ]);

        $response->assertSessionHasErrors('invitation_token');
    }

    public function test_user_invite_command_sends_mail(): void
    {
        Mail::fake();

        User::factory()->admin()->createOne();

        $email = 'cli-invite-'.uniqid('', true).'@example.com';

        $exit = Artisan::call('user:invite', ['email' => $email]);

        $this->assertSame(0, $exit);
        Mail::assertQueued(UserInvitationMail::class, fn (UserInvitationMail $mail): bool => $mail->hasTo($email));
    }

    public function test_non_admin_cannot_post_dashboard_invitations(): void
    {
        Mail::fake();

        $user = User::factory()->createOne(['is_admin' => false]);

        $response = $this->actingAs($user)->post(self::PLATFORM_BASE.'/dashboard/invitations', [
            'email' => 'blocked@example.com',
        ]);

        $response->assertForbidden();
        Mail::assertNothingSent();
    }

    public function test_admin_can_send_dashboard_invitation(): void
    {
        Mail::fake();

        $admin = User::factory()->admin()->createOne();
        $email = 'dash-invite-'.uniqid('', true).'@example.com';

        $response = $this->actingAs($admin)->from(self::PLATFORM_BASE.'/dashboard')->post(
            self::PLATFORM_BASE.'/dashboard/invitations',
            ['email' => $email]
        );

        $response->assertRedirect();
        $response->assertSessionHas('dashboard_status');
        Mail::assertQueued(UserInvitationMail::class, fn (UserInvitationMail $mail): bool => $mail->hasTo($email));
    }
}
