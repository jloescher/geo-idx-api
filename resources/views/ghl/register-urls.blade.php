<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Register MLS Domains — Quantyra</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>body{font-family:system-ui,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem}label{display:block;margin:.5rem 0 .25rem}input,select{width:100%;padding:.5rem}button{margin-top:1rem;padding:.75rem 1.25rem}</style>
</head>
<body>
{{-- Revenue Impact: MLS URL gate → compliant widget revenue without broker liability. --}}
<h1>Register your IDX domains</h1>
<p>MLS compliance: list every HTTPS origin that will embed Quantyra widgets or display MLS data under your participant agreements.</p>
<form method="post" action="{{ route('leadconnector.register-urls.store') }}">
    @csrf
    <label for="primary_url">Primary website URL (https)</label>
    <input id="primary_url" name="primary_url" type="url" value="{{ old('primary_url') }}" required>

    <label for="additional_urls">Additional URLs (one per line or comma-separated)</label>
    <textarea id="additional_urls" name="additional_urls" rows="3">{{ old('additional_urls') }}</textarea>

    <label for="integration_type">Integration</label>
    <select id="integration_type" name="integration_type" required>
        <option value="ghl_website" @selected(old('integration_type')==='ghl_website')>GHL-hosted website</option>
        <option value="external_website" @selected(old('integration_type')==='external_website')>External website (JS embed)</option>
        <option value="both" @selected(old('integration_type')==='both')>Both</option>
    </select>

    @if(!$token->ghl_location_id)
        <label for="manual_location_id">GHL Location ID (required for agency token)</label>
        <input id="manual_location_id" name="manual_location_id" value="{{ old('manual_location_id') }}">
    @endif

    <label><input type="checkbox" name="mls_agreement" value="1" required> I confirm these domains comply with applicable MLS IDX display rules for the feeds I enable.</label>

    @if ($errors->any())
        <ul style="color:#b91c1c">
            @foreach ($errors->all() as $e)<li>{{ $e }}</li>@endforeach
        </ul>
    @endif

    <button type="submit">Save &amp; generate widget key</button>
</form>
</body>
</html>
