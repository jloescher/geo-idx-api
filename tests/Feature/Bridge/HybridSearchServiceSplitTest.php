<?php

namespace Tests\Feature\Bridge;

use App\Http\Requests\Search\SearchRequest;
use App\Models\Listing;
use App\Services\Bridge\BridgeSearchTranslator;
use App\Services\Bridge\HybridSearchService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Validator;
use Tests\TestCase;

class HybridSearchServiceSplitTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'bridge.server_token' => 'test-token',
        ]);
    }

    public function test_closed_only_search_uses_bridge_not_local_mirror(): void
    {
        $propertyUrl = 'https://bridge.test/OData/stellar/Property*';

        Http::fake([
            $propertyUrl => Http::response([
                'value' => [
                    [
                        'ListingKey' => 'STELLAR-CLOSED-1',
                        'StandardStatus' => 'Closed',
                        'ListPrice' => 400000,
                    ],
                ],
            ], 200),
        ]);

        $request = $this->makeSearchRequest([
            'statuses' => ['Closed'],
            'page' => ['limit' => 24, 'skip' => 0],
        ]);

        $translated = app(BridgeSearchTranslator::class)->translate($request, 'stellar');
        $result = app(HybridSearchService::class)->fetchSearchResultPayload($request, 'stellar', $translated);

        $this->assertCount(1, $result['value']);
        $this->assertSame('STELLAR-CLOSED-1', $result['value'][0]['ListingKey']);
        Http::assertSentCount(1);
        $this->assertSame(0, Listing::query()->count());
    }

    public function test_mixed_status_search_merges_local_active_and_bridge_closed(): void
    {
        $propertyUrl = 'https://bridge.test/OData/stellar/Property*';

        Listing::query()->create([
            'dataset_slug' => 'stellar',
            'listing_key' => 'STELLAR-ACTIVE-1',
            'standard_status' => 'Active',
            'list_price' => 500000,
            'city' => 'Tampa',
            'state_or_province' => 'FL',
            'modification_timestamp' => now(),
            'raw_data' => [
                'ListingKey' => 'STELLAR-ACTIVE-1',
                'StandardStatus' => 'Active',
                'ListPrice' => 500000,
                'City' => 'Tampa',
            ],
        ]);

        Http::fake([
            $propertyUrl => Http::response([
                'value' => [
                    [
                        'ListingKey' => 'STELLAR-CLOSED-2',
                        'StandardStatus' => 'Closed',
                        'ListPrice' => 350000,
                        'City' => 'Tampa',
                    ],
                ],
            ], 200),
        ]);

        $request = $this->makeSearchRequest([
            'statuses' => ['Active', 'Pending', 'Closed'],
            'page' => ['limit' => 24, 'skip' => 0],
        ]);

        $translated = app(BridgeSearchTranslator::class)->translate($request, 'stellar');
        $result = app(HybridSearchService::class)->fetchSearchResultPayload($request, 'stellar', $translated);

        $keys = array_column($result['value'], 'ListingKey');
        $this->assertContains('STELLAR-ACTIVE-1', $keys);
        $this->assertContains('STELLAR-CLOSED-2', $keys);
        Http::assertSent(function (Request $request): bool {
            $filter = $request->data()['$filter'] ?? '';

            return str_contains($filter, "tolower(StandardStatus) eq 'closed'");
        });
    }

    /**
     * @param  array<string, mixed>  $input
     */
    private function makeSearchRequest(array $input): SearchRequest
    {
        $rules = (new SearchRequest)->rules();
        $validator = Validator::make($input, $rules);
        $validated = $validator->validated();

        $request = SearchRequest::create('/api/v1/search', 'POST', $input);
        $request->setContainer(app());
        $request->setRedirector(app('redirect'));
        $request->merge($validated);
        $request->setValidator($validator);

        return $request;
    }
}
