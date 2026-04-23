<?php

namespace App\Ghl\Widgets\Controllers;

use Illuminate\Http\Response;

/**
 * Revenue Impact: One script tag → IDX surfaces on GHL sites → more inventory views → more gated leads.
 */
class WidgetLoaderController
{
    public function __invoke(): Response
    {
        $base = json_encode(rtrim((string) config('ghl.urls.api_public'), '/'), JSON_THROW_ON_ERROR);

        $js = <<<JS
(function () {
  var s = document.currentScript;
  if (!s) { return; }
  var apiKey = s.getAttribute('data-api-key') || '';
  var loc = s.getAttribute('data-location-id') || '';
  var widget = s.getAttribute('data-widget') || 'search';
  if (!apiKey) { console.error('Quantyra GeoIDX: data-api-key is required'); return; }
  var base = {$base};
  var url = base + '/widget/' + widget + '/' + encodeURIComponent(apiKey) + '?location_id=' + encodeURIComponent(loc);
  fetch(url, { credentials: 'omit', mode: 'cors' })
    .then(function (r) { return r.text(); })
    .then(function (html) {
      var wrap = document.createElement('div');
      wrap.setAttribute('data-quantyra-widget', widget);
      wrap.innerHTML = html;
      (s.parentNode || document.body).insertBefore(wrap, s.nextSibling);
    })
    .catch(function (e) { console.error('Quantyra widget load failed', e); });
})();
JS;

        return response($js, 200, [
            'Content-Type' => 'application/javascript; charset=UTF-8',
        ]);
    }
}
