<?php

namespace App\Http\Middleware;

use App\Services\TurnstileVerifier;
use Closure;
use Illuminate\Http\Request;
use Illuminate\Validation\ValidationException;
use Symfony\Component\HttpFoundation\Response;

class VerifyTurnstile
{
    /**
     * @var list<string>
     */
    private const array ROUTE_NAMES = [
        'login.store',
        'password.email',
        'password.update',
        'register.store',
    ];

    public function __construct(
        private readonly TurnstileVerifier $turnstile,
    ) {}

    /**
     * @param  Closure(Request): Response  $next
     */
    public function handle(Request $request, Closure $next): Response
    {
        if (! $request->isMethod('POST')) {
            return $next($request);
        }

        $routeName = $request->route()?->getName();

        if (! is_string($routeName) || ! in_array($routeName, self::ROUTE_NAMES, true)) {
            return $next($request);
        }

        if (! $this->turnstile->isEnabled()) {
            return $next($request);
        }

        if ($this->turnstile->verify($request->input('cf-turnstile-response'), $request->ip())) {
            return $next($request);
        }

        throw ValidationException::withMessages([
            'cf-turnstile-response' => [__('Security verification failed. Please try again.')],
        ]);
    }
}
