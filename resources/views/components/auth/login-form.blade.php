@props([
    'autofocusEmail' => false,
    'showIntro' => true,
])

<div class="space-y-6">
    @if ($showIntro)
        <div>
            <h2 class="text-xl font-bold tracking-tight text-slate-100">Subscriber login</h2>
            <p class="mt-1 text-sm text-slate-200">
                Sign in to manage your GeoIDX subscription, widgets, and dashboard.
            </p>
        </div>
    @endif

    <form method="POST" action="{{ route('login', [], false) }}" class="space-y-5">
        @csrf

        <div>
            <label for="login-email" class="block text-sm font-medium text-slate-200">Email</label>
            <input
                id="login-email"
                name="email"
                type="email"
                value="{{ old('email') }}"
                required
                autocomplete="email"
                @if ($autofocusEmail) autofocus @endif
                @class([
                    'mt-2 block w-full rounded-xl border bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:outline-none focus:ring-2 focus:ring-offset-0',
                    'border-red-400/70 focus:border-red-400 focus:ring-red-400/40' => $errors->has('email'),
                    'border-white/15 focus:border-emerald-400/60 focus:ring-emerald-400/30' => ! $errors->has('email'),
                ])
                aria-invalid="{{ $errors->has('email') ? 'true' : 'false' }}"
                @if ($errors->has('email'))
                    aria-describedby="login-email-error"
                @endif
            >
            @error('email')
                <p id="login-email-error" class="mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
            @enderror
        </div>

        <div>
            <label for="login-password" class="block text-sm font-medium text-slate-200">Password</label>
            <input
                id="login-password"
                name="password"
                type="password"
                required
                autocomplete="current-password"
                @class([
                    'mt-2 block w-full rounded-xl border bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:outline-none focus:ring-2 focus:ring-offset-0',
                    'border-red-400/70 focus:border-red-400 focus:ring-red-400/40' => $errors->has('password'),
                    'border-white/15 focus:border-emerald-400/60 focus:ring-emerald-400/30' => ! $errors->has('password'),
                ])
                aria-invalid="{{ $errors->has('password') ? 'true' : 'false' }}"
            >
            @error('password')
                <p class="mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
            @enderror
        </div>

        <div class="flex items-center justify-between gap-4">
            <label class="flex items-center gap-2 text-sm text-slate-200">
                <input
                    type="checkbox"
                    name="remember"
                    value="1"
                    class="size-4 rounded border-white/25 bg-slate-900 text-emerald-500 focus:ring-emerald-400/40"
                    @checked(old('remember'))
                >
                Remember me
            </label>
        </div>

        <button
            type="submit"
            class="w-full rounded-xl bg-emerald-400 px-4 py-3 text-sm font-semibold text-slate-950 shadow-sm transition hover:bg-emerald-300 focus:outline-none focus:ring-2 focus:ring-emerald-400/50 focus:ring-offset-2 focus:ring-offset-slate-900"
        >
            Sign in
        </button>
    </form>

</div>
