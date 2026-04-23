<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Installation complete — Quantyra</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>body{font-family:system-ui,sans-serif;max-width:800px;margin:2rem auto;padding:0 1rem}code{background:#f3f4f6;padding:.2rem .4rem;display:block;white-space:pre-wrap;margin:.5rem 0}</style>
</head>
<body>
{{-- Revenue Impact: Immediate embed snippets → faster time-to-first-lead → higher attach rate. --}}
<h1>You're connected</h1>
@if($widgetApiKey)
    <p><strong>Widget API key:</strong> <code>{{ $widgetApiKey }}</code></p>
    <p><strong>Example embed (search):</strong></p>
    <code>&lt;script src="{{ rtrim(config('ghl.urls.api_public'),'/') }}/widget/loader.js"
  data-widget="search"
  data-api-key="{{ $widgetApiKey }}"
  data-location-id="{{ $ghlLocationId }}"
  data-theme="light"&gt;&lt;/script&gt;</code>
@else
    <p>Bookmark this page after completing registration to retrieve your embed snippets.</p>
@endif
<p><a href="{{ config('ghl.urls.idx_platform') }}/subscribe?ghl_location={{ urlencode($ghlLocationId ?? '') }}">Upgrade on IDX App</a></p>
</body>
</html>
