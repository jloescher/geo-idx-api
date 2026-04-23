<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Quantyra GeoIDX — GHL Install</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>body{font-family:system-ui,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem}a.btn{display:inline-block;background:#2563eb;color:#fff;padding:.75rem 1.25rem;border-radius:.5rem;text-decoration:none;margin-top:1rem}</style>
</head>
<body>
{{-- Revenue Impact: Clear CTA drives OAuth completion → CRM-attached IDX distribution. --}}
<h1>Quantyra GeoIDX for GoHighLevel</h1>
<p>Install the marketplace app, then connect your location or agency. MLS data flows only through the approved idx-api Bridge proxy.</p>
@if(session('error'))
    <p style="color:#b91c1c">{{ session('error') }}</p>
@endif
<p><strong>IDX App</strong> (GHL + dashboard): <a href="{{ config('ghl.urls.idx_platform') }}">{{ config('ghl.urls.idx_platform') }}</a></p>
<p><strong>IDX API</strong> (endpoints + JS widgets): {{ config('ghl.urls.api_public') }}</p>
<p><a class="btn" href="{{ route('leadconnector.oauth.authorize') }}">Connect with GoHighLevel (Location)</a></p>
<p><a class="btn" href="{{ route('leadconnector.oauth.authorize', ['user_type' => 'Company']) }}">Connect as Agency (Company)</a></p>
</body>
</html>
