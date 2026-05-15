<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;

class TurnstileVerifier
{
    public function isEnabled(): bool
    {
        $secret = config('turnstile.secret_key');

        return is_string($secret) && $secret !== '';
    }

    public function verify(?string $token, ?string $remoteIp = null): bool
    {
        if (! $this->isEnabled()) {
            return true;
        }

        if (! is_string($token) || $token === '') {
            return false;
        }

        $response = Http::asForm()
            ->timeout(10)
            ->post(config('turnstile.verify_url'), array_filter([
                'secret' => config('turnstile.secret_key'),
                'response' => $token,
                'remoteip' => $remoteIp,
            ]));

        if (! $response->successful()) {
            return false;
        }

        return $response->json('success') === true;
    }
}
