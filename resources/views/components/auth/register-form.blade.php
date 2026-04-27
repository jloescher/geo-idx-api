<div class="space-y-6">
    <div>
        <h2 class="text-xl font-bold tracking-tight text-slate-100">Create subscriber account</h2>
        <p class="mt-1 text-sm text-slate-200">
            Set up paid access with MLS verification, then manage GeoIDX widgets, LeadConnector, and billing from your dashboard.
        </p>
    </div>

    <form method="POST" action="{{ route('register', [], false) }}" class="space-y-5">
        @csrf

        <div>
            <label for="register-name" class="block text-sm font-medium text-slate-200">Name</label>
            <input
                id="register-name"
                name="name"
                type="text"
                value="{{ old('name') }}"
                required
                autocomplete="name"
                autofocus
                @class([
                    'mt-2 block w-full rounded-xl border bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:outline-none focus:ring-2 focus:ring-offset-0',
                    'border-red-400/70 focus:border-red-400 focus:ring-red-400/40' => $errors->has('name'),
                    'border-white/15 focus:border-emerald-400/60 focus:ring-emerald-400/30' => ! $errors->has('name'),
                ])
            >
            @error('name')
                <p class="mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
            @enderror
        </div>

        <div>
            <label for="register-email" class="block text-sm font-medium text-slate-200">Email</label>
            <input
                id="register-email"
                name="email"
                type="email"
                value="{{ old('email') }}"
                required
                autocomplete="email"
                @class([
                    'mt-2 block w-full rounded-xl border bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:outline-none focus:ring-2 focus:ring-offset-0',
                    'border-red-400/70 focus:border-red-400 focus:ring-red-400/40' => $errors->has('email'),
                    'border-white/15 focus:border-emerald-400/60 focus:ring-emerald-400/30' => ! $errors->has('email'),
                ])
            >
            @error('email')
                <p class="mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
            @enderror
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
                <label for="register-dataset" class="block text-sm font-medium text-slate-200">MLS dataset</label>
                <select
                    id="register-dataset"
                    name="dataset"
                    class="mt-2 block w-full rounded-xl border border-white/15 bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:border-emerald-400/60 focus:outline-none focus:ring-2 focus:ring-emerald-400/30 focus:ring-offset-0"
                >
                    <option value="stellar" @selected(old('dataset', 'stellar') === 'stellar')>Stellar MLS (stellar)</option>
                </select>
            </div>
            <div>
                <label for="register-domain-slug" class="block text-sm font-medium text-slate-200">Primary domain</label>
                <input
                    id="register-domain-slug"
                    name="domain_slug"
                    type="text"
                    value="{{ old('domain_slug') }}"
                    required
                    placeholder="example.com"
                    class="mt-2 block w-full rounded-xl border border-white/15 bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:border-emerald-400/60 focus:outline-none focus:ring-2 focus:ring-emerald-400/30 focus:ring-offset-0"
                >
            </div>
        </div>
        @error('domain_slug')
            <p class="-mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
        @enderror

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
                <label for="register-mls-id" class="block text-sm font-medium text-slate-200">MLS ID</label>
                <input
                    id="register-mls-id"
                    name="mls_id"
                    type="text"
                    value="{{ old('mls_id') }}"
                    required
                    class="mt-2 block w-full rounded-xl border border-white/15 bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:border-emerald-400/60 focus:outline-none focus:ring-2 focus:ring-emerald-400/30 focus:ring-offset-0"
                >
            </div>
            <div>
                <label for="register-mls-email" class="block text-sm font-medium text-slate-200">MLS member email</label>
                <input
                    id="register-mls-email"
                    name="mls_email"
                    type="email"
                    value="{{ old('mls_email', old('email')) }}"
                    required
                    class="mt-2 block w-full rounded-xl border border-white/15 bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:border-emerald-400/60 focus:outline-none focus:ring-2 focus:ring-emerald-400/30 focus:ring-offset-0"
                >
            </div>
        </div>
        @error('mls_id')
            <p class="-mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
        @enderror
        @error('mls_email')
            <p class="-mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
        @enderror

        <div>
            <label for="register-password" class="block text-sm font-medium text-slate-200">Password</label>
            <input
                id="register-password"
                name="password"
                type="password"
                required
                autocomplete="new-password"
                @class([
                    'mt-2 block w-full rounded-xl border bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:outline-none focus:ring-2 focus:ring-offset-0',
                    'border-red-400/70 focus:border-red-400 focus:ring-red-400/40' => $errors->has('password'),
                    'border-white/15 focus:border-emerald-400/60 focus:ring-emerald-400/30' => ! $errors->has('password'),
                ])
            >
            @error('password')
                <p class="mt-2 text-sm font-medium text-red-300" role="alert">{{ $message }}</p>
            @enderror
        </div>

        <div>
            <label for="register-password-confirmation" class="block text-sm font-medium text-slate-200">Confirm password</label>
            <input
                id="register-password-confirmation"
                name="password_confirmation"
                type="password"
                required
                autocomplete="new-password"
                class="mt-2 block w-full rounded-xl border border-white/15 bg-slate-950/50 px-4 py-3 text-slate-100 shadow-sm transition focus:border-emerald-400/60 focus:outline-none focus:ring-2 focus:ring-emerald-400/30 focus:ring-offset-0"
            >
        </div>

        <button
            type="submit"
            class="w-full rounded-xl bg-emerald-400 px-4 py-3 text-sm font-semibold text-slate-950 shadow-sm transition hover:bg-emerald-300 focus:outline-none focus:ring-2 focus:ring-emerald-400/50 focus:ring-offset-2 focus:ring-offset-slate-900"
        >
            Create account
        </button>
    </form>
</div>
