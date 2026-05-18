<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>GeoIDX | Quantyra Labs</title>
    <meta name="description" content="Sign in to the GeoIDX dashboard.">
    @vite(['resources/css/app.css', 'resources/js/app.js'])
</head>
<body class="min-h-screen bg-slate-950 px-4 py-16 text-slate-100 antialiased sm:px-6 lg:px-8">
    <div class="mx-auto max-w-2xl text-center">
        <h1 class="text-3xl font-bold tracking-tight text-white sm:text-4xl">GeoIDX</h1>
        <p class="mt-4 text-lg text-slate-300">
            Sign in to manage your account.
        </p>
        <div class="mt-10 flex flex-col items-stretch justify-center gap-3 sm:flex-row sm:items-center">
            @auth
                <a
                    href="{{ \App\Support\DashboardUrl::panel() }}"
                    class="inline-flex min-h-12 items-center justify-center rounded-full bg-cyan-400 px-8 py-3 text-base font-semibold text-slate-950 hover:bg-cyan-300"
                >
                    Open dashboard
                </a>
            @else
                @if (Route::has('login'))
                    <a
                        href="{{ route('login', [], false) }}"
                        class="inline-flex min-h-12 items-center justify-center rounded-full bg-cyan-400 px-8 py-3 text-base font-semibold text-slate-950 hover:bg-cyan-300"
                    >
                        Log in
                    </a>
                @endif
            @endauth
        </div>
    </div>
</body>
</html>
