<?php

namespace Tests\Feature\Mls;

use App\Services\Mls\MlsActivePendingListingsFetcher;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class MlsActivePendingListingsFetcherSparkTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        config([
            'spark.live_reso_base_url' => 'https://sparkapi.test/v1/Reso/OData',
            'spark.access_token' => 'test-spark-token',
            'spark.timeout_seconds' => 30,
            'mls.listings_sync_page_size' => 1,
            'mls.listings_sync_max_pages' => 5,
            'mls.listings_sync_max_rows' => 10,
        ]);
    }

    public function test_spark_feed_fetches_active_pending_from_live_property_api(): void
    {
        $collectionUrl = 'https://sparkapi.test/v1/Reso/OData/Property';
        $nextUrl = 'https://sparkapi.test/v1/Reso/OData/Property?$skip=1';

        Http::fake(function (\Illuminate\Http\Client\Request $request) use ($collectionUrl, $nextUrl) {
            if ($request->url() === $nextUrl) {
                return Http::response([
                    'value' => [
                        [
                            'ListingKey' => 'SPARK-2',
                            'StandardStatus' => 'Pending',
                            'ModificationTimestamp' => '2024-07-14T00:56:59Z',
                        ],
                    ],
                ], 200);
            }

            if (str_starts_with($request->url(), $collectionUrl)) {
                return Http::response([
                    'value' => [
                        [
                            'ListingKey' => 'SPARK-1',
                            'StandardStatus' => 'Active',
                            'ModificationTimestamp' => '2024-07-13T00:56:59Z',
                        ],
                    ],
                    '@odata.nextLink' => $nextUrl,
                ], 200);
            }

            return Http::response([], 404);
        });

        $request = Request::create('/api/v1/listings', 'GET');
        $request->attributes->set('mls.feed_code', 'spark_beaches');

        $result = app(MlsActivePendingListingsFetcher::class)->fetchMergedCollectionForCache($request);

        $body = json_decode($result['body'], true, 512, JSON_THROW_ON_ERROR);
        $this->assertCount(2, $body['value']);
        $this->assertSame('SPARK-1', $body['value'][0]['ListingKey']);
        $this->assertSame('SPARK-2', $body['value'][1]['ListingKey']);

        Http::assertSent(function (\Illuminate\Http\Client\Request $request) use ($collectionUrl): bool {
            if (! str_starts_with($request->url(), $collectionUrl)) {
                return false;
            }

            $filter = $request->data()['$filter'] ?? '';

            return str_contains($filter, "StandardStatus eq 'Active'")
                && str_contains($filter, "StandardStatus eq 'Pending'")
                && str_contains($filter, 'ModificationTimestamp ge datetime');
        });
    }
}
