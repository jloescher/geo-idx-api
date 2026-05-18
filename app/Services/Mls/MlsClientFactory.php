<?php

namespace App\Services\Mls;

use App\Enums\MlsProvider;
use App\Services\Bridge\BridgeHttpService;
use App\Services\Spark\SparkHttpService;
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
        private SparkHttpService $sparkHttp,
        private MlsFeedResolver $feeds,
    ) {}

    public function bridgeClientForFeed(string $feedCode): BridgeClient
    {
        $def = $this->feeds->feedDefinition($this->feeds->normalizeWireDatasetToCatalogKey($feedCode));
        if (! is_array($def) || ($def['provider'] ?? '') !== MlsProvider::STELLAR->value) {
            throw new HttpException(400, "Feed '{$feedCode}' is not Bridge-backed.");
        }

        $datasetId = (string) ($def['bridge_dataset_id'] ?? '');
        if ($datasetId === '') {
            throw new HttpException(400, "Feed '{$feedCode}' is missing bridge_dataset_id.");
        }

        $catalogKey = $this->feeds->normalizeWireDatasetToCatalogKey($feedCode);

        return new BridgeClient($this->bridgeHttp, $catalogKey, $datasetId);
    }

    public function sparkClientForFeed(string $feedCode): SparkClient
    {
        $catalogKey = $this->feeds->normalizeWireDatasetToCatalogKey($feedCode);
        $def = $this->feeds->feedDefinition($catalogKey);
        if (! is_array($def) || ($def['provider'] ?? '') !== MlsProvider::SPARK->value) {
            throw new HttpException(400, "Feed '{$feedCode}' is not Spark-backed.");
        }

        $datasetId = (string) ($def['spark_dataset_id'] ?? '');
        if ($datasetId === '') {
            throw new HttpException(400, "Feed '{$feedCode}' is missing spark_dataset_id.");
        }

        return new SparkClient($this->sparkHttp, $catalogKey, $datasetId);
    }

    public function bridgeClientFromRequest(Request $request): BridgeClient
    {
        return $this->bridgeClientForFeed($this->resolveFeedCodeFromRequest($request));
    }

    public function sparkClientFromRequest(Request $request): SparkClient
    {
        return $this->sparkClientForFeed($this->resolveFeedCodeFromRequest($request));
    }

    public function providerForRequest(Request $request): MlsProvider
    {
        return $this->feeds->providerForFeedCode($this->resolveFeedCodeFromRequest($request));
    }

    private function resolveFeedCodeFromRequest(Request $request): string
    {
        $code = (string) $request->attributes->get('mls.feed_code', '');
        if ($code === '') {
            $code = $this->feeds->resolveFeedCode($request);
        }

        return $code;
    }
}
