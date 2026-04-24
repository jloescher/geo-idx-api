<?php

namespace Tests\Unit;

use App\Http\Responses\Auth\LoginResponse;
use App\Http\Responses\Auth\RegisterResponse;
use Illuminate\Http\Request;
use Tests\TestCase;

class AuthResponseRedirectTest extends TestCase
{
    private function requestWithSubscribedState(bool $subscribed): Request
    {
        $request = Request::create('/login', 'POST');
        $request->setUserResolver(static fn (): object => new class($subscribed)
        {
            public function __construct(private readonly bool $subscribed) {}

            public function subscribed(string $name): bool
            {
                return $this->subscribed && $name === 'default';
            }
        });

        return $request;
    }

    public function test_login_response_redirects_subscribed_users_to_dashboard(): void
    {
        $response = app(LoginResponse::class)->toResponse($this->requestWithSubscribedState(true));

        $this->assertStringEndsWith('/dashboard', $response->getTargetUrl());
    }

    public function test_login_response_redirects_non_subscribers_to_pricing(): void
    {
        $response = app(LoginResponse::class)->toResponse($this->requestWithSubscribedState(false));

        $this->assertStringEndsWith('/#pricing', $response->getTargetUrl());
    }

    public function test_register_response_redirects_non_subscribers_to_pricing(): void
    {
        $response = app(RegisterResponse::class)->toResponse($this->requestWithSubscribedState(false));

        $this->assertStringEndsWith('/#pricing', $response->getTargetUrl());
    }
}
