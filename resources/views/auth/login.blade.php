<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Log in | GeoIDX by Quantyra Labs</title>
    <meta name="robots" content="noindex, nofollow">
    @vite(['resources/css/app.css', 'resources/js/app.js'])
</head>
<body class="min-h-screen bg-slate-950 text-slate-100 antialiased">
    <div class="relative min-h-screen overflow-hidden">
        <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(16,185,129,0.22),transparent_50%)]"></div>

        <header class="relative z-10 border-b border-white/10 bg-slate-950/80 backdrop-blur">
            <div class="mx-auto flex max-w-6xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
                <a href="/" class="text-sm font-semibold text-slate-200 hover:text-white">← Back to home</a>
                <span class="text-sm font-semibold tracking-tight text-slate-100">GeoIDX by Quantyra Labs</span>
            </div>
        </header>

        <main class="relative z-10 mx-auto flex min-h-[calc(100vh-4.5rem)] max-w-6xl items-center justify-center px-4 py-12 sm:px-6 lg:px-8">
            <div class="w-full max-w-md rounded-2xl border border-white/10 bg-slate-900/90 p-8 shadow-2xl backdrop-blur">
                <x-auth.login-form :autofocus-email="true" :show-intro="true" />
                @if (Route::has('register'))
                    <p class="mt-6 border-t border-white/10 pt-4 text-center text-xs text-slate-300">
                        New to Quantyra IDX?
                        <a href="{{ route('register', [], false) }}" class="font-medium text-emerald-300 underline decoration-emerald-400/50 hover:text-emerald-200">
                            Create account
                        </a>
                    </p>
                @endif
            </div>
        </main>
    </div>
</body>
</html>
