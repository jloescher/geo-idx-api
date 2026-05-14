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
        if (! is_array($def) || ($def['provider'] ?? '') !== MlsProvider::Bridge->value) {
            throw new HttpException(400, "Feed '{$feedCode}' is not Bridge-backed.");
        }

        $datasetId = (string) ($def['bridge_dataset_id'] ?? $feedCode);

        return new BridgeClient($this->bridgeHttp, $feedCode, $datasetId);
    }

    public function sparkClientForFeed(string $feedCode): SparkClient
    {
        $def = $this->feeds->feedDefinition($feedCode);
        if (! is_array($def) || ($def['provider'] ?? '') !== MlsProvider::Spark->value) {
            throw new HttpException(400, "Feed '{$feedCode}' is not Spark-backed.");
        }

        return new SparkClient($feedCode, $def);
    }

    public function bridgeClientFromRequest(Request $request): BridgeClient
    {
        $code = (string) $request->attributes->get('mls.feed_code', '');
        if ($code === '') {
            $code = $this->feeds->resolveFeedCode($request);
        }

        return $this->bridgeClientForFeed($code);
    }

    public function sparkClientFromRequest(Request $request): SparkClient
    {
        $code = (string) $request->attributes->get('mls.feed_code', '');
        if ($code === '') {
            $code = $this->feeds->resolveFeedCode($request);
        }

        return $this->sparkClientForFeed($code);
    }

    public function providerForRequest(Request $request): MlsProvider
    {
        $code = (string) $request->attributes->get('mls.feed_code', '');
        if ($code === '') {
            $code = $this->feeds->resolveFeedCode($request);
        }

        $def = $this->feeds->feedDefinition($code);
        $p = is_array($def) ? (string) ($def['provider'] ?? '') : '';

        return MlsProvider::tryFrom($p) ?? MlsProvider::Bridge;
    }
}
