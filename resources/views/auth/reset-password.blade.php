<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Set new password | GeoIDX by Quantyra Labs</title>
    <meta name="robots" content="noindex, nofollow">
    @vite(['resources/css/app.css', 'resources/js/app.js'])
</head>
<body class="min-h-screen bg-slate-950 text-slate-100 antialiased">
    <div class="relative min-h-screen overflow-hidden">
        <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(16,185,129,0.22),transparent_50%)]"></div>

        <header class="relative z-10 border-b border-white/10 bg-slate-950/80 backdrop-blur">
            <div class="mx-auto flex max-w-6xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
                <a href="{{ route('login', [], false) }}" class="text-sm font-semibold text-slate-200 hover:text-white">← Back to log in</a>
                <span class="text-sm font-semibold tracking-tight text-slate-100">GeoIDX by Quantyra Labs</span>
            </div>
        </header>

        <main class="relative z-10 mx-auto flex min-h-[calc(100vh-4.5rem)] max-w-6xl items-center justify-center px-4 py-12 sm:px-6 lg:px-8">
            <div class="w-full max-w-md rounded-2xl border border-white/10 bg-slate-900/90 p-8 shadow-2xl backdrop-blur">
                <div class="space-y-6">
                    <div>
                        <h2 class="text-xl font-bold tracking-tight text-slate-100">Set a new password</h2>
                    </div>

                    <form method="POST" action="{{ route('password.update', [], false) }}" class="space-y-5">
                        @csrf

                        <input type="hidden" name="token" value="{{ $request->route('token') }}">

                        <div>
                            <label for="reset-email" class="block text-sm font-medium text-slate-200">Email</label>
                            <input
                                id="reset-email"
                                name="email"
                                type="email"
                                value="{{ old('email', $request->email) }}"
                                required
                                autofocus
                                autocomplete="email"
                                class="mt-2 block w-full rounded-xl border border-white/15 bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:border-emerald-400/60 focus:outline-none focus:ring-2 focus:ring-emerald-400/30 focus:ring-offset-0"
                            >
                            @error('email')
                                <p class="mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
                            @enderror
                        </div>

                        <div>
                            <label for="reset-password" class="block text-sm font-medium text-slate-200">Password</label>
                            <input
                                id="reset-password"
                                name="password"
                                type="password"
                                required
                                autocomplete="new-password"
                                class="mt-2 block w-full rounded-xl border border-white/15 bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:border-emerald-400/60 focus:outline-none focus:ring-2 focus:ring-emerald-400/30 focus:ring-offset-0"
                            >
                            @error('password')
                                <p class="mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
                            @enderror
                        </div>

                        <div>
                            <label for="reset-password-confirmation" class="block text-sm font-medium text-slate-200">Confirm password</label>
                            <input
                                id="reset-password-confirmation"
                                name="password_confirmation"
                                type="password"
                                required
                                autocomplete="new-password"
                                class="mt-2 block w-full rounded-xl border border-white/15 bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:border-emerald-400/60 focus:outline-none focus:ring-2 focus:ring-emerald-400/30 focus:ring-offset-0"
                            >
                        </div>

                        <x-auth.turnstile-widget />

                        <button
                            type="submit"
                            class="w-full rounded-xl bg-emerald-400 px-4 py-3 text-sm font-semibold text-slate-950 shadow-sm transition hover:bg-emerald-300 focus:outline-none focus:ring-2 focus:ring-emerald-400/50 focus:ring-offset-2 focus:ring-offset-slate-900"
                        >
                            Reset password
                        </button>
                    </form>
                </div>
            </div>
        </main>
    </div>
</body>
</html>
