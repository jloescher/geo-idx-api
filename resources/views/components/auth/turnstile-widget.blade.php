@php
    $siteKey = config('turnstile.site_key');
@endphp

@if (is_string($siteKey) && $siteKey !== '')
    <div>
        <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
        <div
            class="cf-turnstile"
            data-sitekey="{{ $siteKey }}"
            data-theme="dark"
        ></div>
        @error('cf-turnstile-response')
            <p class="mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
        @enderror
    </div>
@endif
