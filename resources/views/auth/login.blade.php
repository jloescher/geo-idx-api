<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Subscriber Login</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 2rem; background: #f1f5f9; color: #0f172a; }
        .wrapper { max-width: 480px; margin: 2rem auto; background: #fff; border-radius: 8px; padding: 1.5rem; }
        label { display: block; margin-top: 1rem; font-weight: 600; }
        input[type="email"], input[type="password"] { width: 100%; padding: .6rem; margin-top: .4rem; border: 1px solid #cbd5e1; border-radius: 6px; }
        button { margin-top: 1rem; background: #0284c7; color: #fff; border: none; padding: .7rem 1rem; border-radius: 6px; cursor: pointer; }
        .error { color: #dc2626; margin-top: .75rem; }
    </style>
</head>
<body>
<div class="wrapper">
    <h1>Subscriber Login</h1>
    <form method="POST" action="{{ route('login') }}">
        @csrf
        <label for="email">Email</label>
        <input id="email" type="email" name="email" value="{{ old('email') }}" required autocomplete="email">

        <label for="password">Password</label>
        <input id="password" type="password" name="password" required autocomplete="current-password">

        <label style="font-weight: 400; margin-top: .8rem;">
            <input type="checkbox" name="remember" value="1">
            Remember me
        </label>

        <button type="submit">Login</button>
    </form>

    @if ($errors->any())
        <div class="error">{{ $errors->first() }}</div>
    @endif
</div>
</body>
</html>
