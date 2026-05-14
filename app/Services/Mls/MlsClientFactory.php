<?php

namespace App\Services\Mls;

use App\Enums\MlsProvider;
use App\Services\Bridge\BridgeHttpService;
use Illuminate\Http\Request;
use Symfony\Component\HttpKernel\Exception\HttpException;

/**
 * Revenue impact: central client resolution prevents shipping the wrong MLS bill to the wrong subscriber tier.
 *
 * Compliance: resolves credentials strictly from `MlsFeedResolver` + env; never trusts client-provided secrets.
 */
final readonly class MlsClientFactory
{
    public function __construct(
        private BridgeHttpService $bridgeHttp,
        private MlsFeedResolver $feeds,
    ) {}

    public function bridgeClientForFeed(string $feedCode): BridgeClient
    {
        $def = $this->feeds->feedDefinition($feedCode);
        if (! is_array($def) || ($def['provider'] ?? '') !== MlsProvider::STELLAR->value) {
            throw new HttpException(400, "Feed '{$feedCode}' is not Bridge-backed.");
        }

        $datasetId = (string) ($def['bridge_dataset_id'] ?? '');
        if ($datasetId === '') {
            throw new HttpException(400, "Feed '{$feedCode}' is missing bridge_dataset_id.");
        }

        return new BridgeClient($this->bridgeHttp, $feedCode, $datasetId);
    }

    public function bridgeClientFromRequest(Request $request): BridgeClient
    {
        $code = (string) $request->attributes->get('mls.feed_code', '');
        if ($code === '') {
            $code = $this->feeds->resolveFeedCode($request);
        }

        return $this->bridgeClientForFeed($code);
    }

    public function providerForRequest(Request $request): MlsProvider
    {
        return MlsProvider::STELLAR;
    }
}
