<?php

namespace App\Http\Controllers\Api;

use App\Ghl\Widgets\Services\WidgetEmbedValidationService;
use App\Ghl\Widgets\Support\TrustedWidgetPreviewOrigins;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class WidgetValidationController
{
    public function __construct(
        private readonly WidgetEmbedValidationService $widgetEmbedValidation,
    ) {}

    public function __invoke(Request $request): Response
    {
        if ($request->isMethod('OPTIONS')) {
            return $this->withWidgetValidateCors($request, response('', 204));
        }

        $validated = $request->validate([
            'token' => ['required', 'string', 'max:64'],
            'hostname' => ['nullable', 'string', 'max:255'],
            'referrer' => ['nullable', 'string', 'max:1000'],
            'requireFooter' => ['nullable', 'boolean'],
        ]);

        $trustHostnameForPreview = TrustedWidgetPreviewOrigins::originIfTrusted((string) $request->header('Origin', '')) !== null;

        $response = $this->widgetEmbedValidation->respond($validated, $trustHostnameForPreview);

        return $this->withWidgetValidateCors($request, $response);
    }

    private function withWidgetValidateCors(Request $request, Response $response): Response
    {
        $origin = (string) $request->headers->get('Origin', '');
        if ($origin !== '') {
            $response->headers->set('Access-Control-Allow-Origin', $origin);
            $response->headers->set('Access-Control-Allow-Methods', 'POST, OPTIONS');
            $response->headers->set('Access-Control-Allow-Headers', 'Content-Type, Accept');
            $response->headers->set('Vary', 'Origin');
        }

        return $response;
    }
}
