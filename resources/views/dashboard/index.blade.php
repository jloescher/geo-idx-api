<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Subscriber Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 2rem; background: #0f172a; color: #e2e8f0; }
        a { color: #38bdf8; }
        form { margin-top: 1rem; }
        button { background: #ef4444; color: #fff; border: none; padding: .6rem .9rem; border-radius: 6px; cursor: pointer; }
    </style>
</head>
<body>
<h1>Subscriber Dashboard</h1>
<p>Welcome, {{ auth()->user()->name }}.</p>
<p>Access the embedded app at <a href="{{ url('/leadconnectorapp') }}">{{ url('/leadconnectorapp') }}</a>.</p>

<form method="POST" action="{{ route('logout') }}">
    @csrf
    <button type="submit">Logout</button>
</form>
</body>
</html>
