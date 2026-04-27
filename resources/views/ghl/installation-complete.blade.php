<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Installation complete — Quantyra</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 2rem auto; padding: 0 1rem; line-height: 1.5; color: #0f172a; }
        code { background: #f1f5f9; padding: .2rem .4rem; border-radius: 4px; font-size: 0.9em; }
        pre, .embed { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 1rem; overflow-x: auto; white-space: pre-wrap; word-break: break-all; }
        .next { margin-top: 2rem; padding-top: 1.5rem; border-top: 1px solid #e2e8f0; }
        .next ul { margin: 0.5rem 0 0; padding-left: 1.25rem; }
        .next a { font-weight: 600; color: #0369a1; }
    </style>
</head>
<body>
{{-- Revenue Impact: Immediate embed snippets → faster time-to-first-lead → higher attach rate. --}}
<h1>You're connected</h1>
<p>GoHighLevel does not always surface third-party marketplace apps on the home screen. Use the links below to open Quantyra on the IDX App (subscriber dashboard) or return here to manage install keys and approved domains.</p>
@if($widgetApiKey)
    <p><strong>Widget site key:</strong> <code>{{ $widgetApiKey }}</code></p>
    <p><strong>Example embed (search):</strong></p>
    <div class="embed">&lt;script src="{{ $apiPublicUrl }}/widget/loader.js"
  ?token={{ $widgetApiKey }}&amp;primaryColor=3b82f6&amp;accentColor=10b981
  data-widget="search"
  data-footer-required="true"
  data-location-id="{{ $ghlLocationId }}"
  data-theme="light"&gt;&lt;/script&gt;</div>
    <p><strong>Required compliance anchor:</strong></p>
    <div class="embed">&lt;div data-quantyragidx-footer="true"&gt;&lt;/div&gt;</div>
@else
    <p>Bookmark this page after completing registration to retrieve your embed snippets.</p>
@endif

<div class="next">
    <h2 style="font-size:1.1rem;margin:0 0 0.5rem;">What to do next</h2>
    <ul>
        <li><a href="{{ $subscriberDashboardUrl }}">Open subscriber dashboard</a> — widget library, previews, approved domains, and billing (sign in with your Quantyra IDX account).</li>
        <li><a href="{{ $manageGhlInstallUrl }}">Manage this GoHighLevel connection</a> — site keys, origins, and re-run OAuth if needed.</li>
        <li><a href="{{ $leadsInsightsUrl }}">Leads &amp; CRM sync</a> — summary in the IDX App (widgets send leads into GHL when configured).</li>
    </ul>
    <p style="font-size:0.9rem;color:#64748b;margin-top:1rem;">In GHL, find installed apps under <strong>Settings → Integrations</strong> (wording varies by account). Quantyra’s marketplace listing is separate from your own website embeds.</p>
</div>

<p style="margin-top:1.5rem;"><a href="{{ $subscribeUrl }}">Upgrade on IDX App</a></p>
</body>
</html>
