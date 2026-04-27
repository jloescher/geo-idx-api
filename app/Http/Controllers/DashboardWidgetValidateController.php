<?php

namespace App\Http\Controllers;

use App\Ghl\Widgets\Services\WidgetEmbedValidationService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Same-origin widget validation for the subscriber dashboard (session + CSRF).
 */
class DashboardWidgetValidateController extends Controller
{
    public function __construct(
        private readonly WidgetEmbedValidationService $widgetEmbedValidation,
    ) {}

    public function __invoke(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'token' => ['required', 'string', 'max:64'],
            'hostname' => ['nullable', 'string', 'max:255'],
            'referrer' => ['nullable', 'string', 'max:1000'],
            'requireFooter' => ['nullable', 'boolean'],
        ]);

        return $this->widgetEmbedValidation->respond($validated, true);
    }
}
