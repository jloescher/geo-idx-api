<?php

namespace App\Ghl\Widgets\Services;

use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Ghl\Widgets\Support\OriginMatcher;
use App\Models\User;
use App\Widgets\DirectSiteWidgetContext;
use Illuminate\Http\JsonResponse;

/**
 * Shared widget hostname + token validation for public loader and dashboard preview.
 */
final class WidgetEmbedValidationService
{
    /**
     * @param  array{token: string, hostname?: string|null, referrer?: string|null, requireFooter?: bool|null}  $validated
     */
    public function respond(array $validated, bool $trustQuantyraDashboardHost): JsonResponse
    {
        $token = (string) $validated['token'];
        $ghlRow = GhlRegisteredUrl::query()
            ->where('widget_api_key', $token)
            ->where('widget_access_enabled', true)
            ->first();

        if ($ghlRow !== null) {
            return $this->respondForAllowlist(
                $ghlRow->ghl_location_id,
                $ghlRow->allowedOrigins(),
                $validated,
                $trustQuantyraDashboardHost,
            );
        }

        $user = User::query()->where('widget_embed_site_key', $token)->first();
        if ($user !== null) {
            $ctx = new DirectSiteWidgetContext($user);

            return $this->respondForAllowlist(
                $ctx->ghl_location_id,
                $ctx->allowedOrigins(),
                $validated,
                $trustQuantyraDashboardHost,
            );
        }

        return response()->json([
            'ok' => false,
            'reason' => 'invalid_token',
            'message' => 'Invalid subscriber token for widget runtime.',
        ], 401);
    }

    /**
     * @param  list<string>  $allowedOrigins
     * @param  array{token: string, hostname?: string|null, referrer?: string|null, requireFooter?: bool|null}  $validated
     */
    private function respondForAllowlist(
        string $locationId,
        array $allowedOrigins,
        array $validated,
        bool $trustQuantyraDashboardHost,
    ): JsonResponse {
        $hostname = isset($validated['hostname']) && is_string($validated['hostname']) && $validated['hostname'] !== ''
            ? strtolower((string) $validated['hostname'])
            : '';

        if ($trustQuantyraDashboardHost && $hostname !== '' && $this->isTrustedQuantyraDashboardHostname($hostname)) {
            return $this->successJson($locationId, 'https://'.$hostname, (bool) ($validated['requireFooter'] ?? true));
        }

        $candidateOrigins = [];
        if ($hostname !== '') {
            $candidateOrigins[] = 'https://'.$hostname;
            $candidateOrigins[] = 'http://'.$hostname;
        }
        if (! empty($validated['referrer'])) {
            $candidateOrigins[] = (string) $validated['referrer'];
        }

        $matchedOrigin = null;
        foreach ($candidateOrigins as $candidateOrigin) {
            $matchedOrigin = OriginMatcher::allowedOrigin($candidateOrigin, $allowedOrigins);
            if ($matchedOrigin !== null) {
                break;
            }
        }

        if ($matchedOrigin === null) {
            return response()->json([
                'ok' => false,
                'reason' => 'domain_not_whitelisted',
                'message' => 'This domain is not approved for IDX widget usage.',
            ], 403);
        }

        return $this->successJson($locationId, $matchedOrigin, (bool) ($validated['requireFooter'] ?? true));
    }

    private function successJson(string $locationId, string $origin, bool $requireFooter): JsonResponse
    {
        return response()->json([
            'ok' => true,
            'reason' => null,
            'locationId' => $locationId,
            'origin' => $origin,
            'requiresFooter' => $requireFooter,
            'branding' => [
                'brokerage' => 'Realty Of America, LLC',
                'sourceAttribution' => 'Listings courtesy of Stellar MLS as distributed by MLS GRID',
                'dmcaEmail' => 'support@quantyralabs.cc',
                'consumerDisclaimer' => 'Based on information submitted to the MLS GRID as of [timestamp]. All data is obtained from various sources and may not have been verified by broker or MLS GRID. Supplied Open House Information is subject to change without notice. All information should be independently reviewed and verified for accuracy. Properties may or may not be listed by the office/agent presenting the information.',
            ],
        ]);
    }

    private function isTrustedQuantyraDashboardHostname(string $hostname): bool
    {
        /** @var list<string> $hosts */
        $hosts = config('idx_urls.widget_preview_hostnames', []);

        return in_array(strtolower($hostname), $hosts, true);
    }
}
