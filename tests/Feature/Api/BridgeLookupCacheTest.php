<?php

namespace Tests\Feature\Api;

use App\Models\Domain;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Http;
use Tests\TestCase;

class BridgeLookupCacheTest extends TestCase
{
    use RefreshDatabase;

    protected function setUp(): void
    {
        parent::setUp();

        config([
            'bridge.host' => 'https://bridge.test',
            'bridge.server_token' => 'test-bridge-key',
            'bridge.dataset' => 'stellar',
            'bridge.datasets' => ['stellar'],
            'bridge.path_prefix' => '',
            'bridge.reso_root' => '',
            'bridge.images_public_base' => 'https://idx-images.test',
            'bridge.lookups_cache_ttl_seconds' => 2_592_000,
            'coingecko.cache_key' => 'coingecko.pricing.matrix',
            'coingecko.asset_ids' => [],
            'coingecko.vs_currencies' => [],
        ]);
    }

    private function createDomain(): Domain
    {
        return Domain::query()->create([
            'domain_slug' => 'searchtampabayhouses.com',
            'is_active' => true,
        ]);
    }

    private function fakeLookupResponse(): array
    {
        return [
            'value' => [
                [
                    'LookupName' => 'PropertySubType',
                    'LookupValue' => 'Single Family Residence',
                    'StandardLookupValue' => 'Single Family Residence',
                    'ResourceName' => 'Property',
                ],
                [
                    'LookupName' => 'PropertySubType',
                    'LookupValue' => 'Condominium',
                    'StandardLookupValue' => 'Condominium',
                    'ResourceName' => 'Property',
                ],
                [
                    'LookupName' => 'PropertyCondition',
                    'LookupValue' => 'Existing',
                    'StandardLookupValue' => '',
                    'ResourceName' => 'Property',
                ],
            ],
            '@odata.context' => 'https://bridge.test/stellar/$metadata#Lookup',
        ];
    }

    public function test_lookup_caches_response_and_skips_upstream_on_second_call(): void
    {
        $this->createDomain();

        $lookupData = $this->fakeLookupResponse();
        $callCount = 0;

        Http::fake(function () use ($lookupData, &$callCount) {
            $callCount++;

            return Http::response(
                json_encode($lookupData, JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            );
        });

        // First call: should hit Bridge
        $response1 = $this->getJson('/api/v1/lookup', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);
        $response1->assertOk();
        $this->assertSame('Single Family Residence', $response1->json('value.0.LookupValue'));
        $this->assertSame(1, $callCount);

        // Verify row was written to bridge_search_cache
        $this->assertSame(1, DB::table('bridge_search_cache')
            ->where('partition_key', 'lookups:stellar')
            ->count());

        // Second call: should return cached data without hitting Bridge
        $response2 = $this->getJson('/api/v1/lookup', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);
        $response2->assertOk();
        $this->assertSame('Single Family Residence', $response2->json('value.0.LookupValue'));
        $this->assertSame(1, $callCount, 'Bridge should only be called once; second call should use cache.');
    }

    public function test_lookup_caches_different_filter_params_separately(): void
    {
        $this->createDomain();

        $subTypeData = [
            'value' => [['LookupName' => 'PropertySubType', 'LookupValue' => 'Duplex']],
        ];
        $conditionData = [
            'value' => [['LookupName' => 'PropertyCondition', 'LookupValue' => 'Fixer']],
        ];

        $callCount = 0;
        Http::fake(function () use (&$callCount, $subTypeData, $conditionData) {
            $callCount++;

            // Return different data based on the request
            return Http::response(
                json_encode($callCount === 1 ? $subTypeData : $conditionData, JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            );
        });

        // First call: filter by PropertySubType
        $response1 = $this->getJson('/api/v1/lookup?$filter=LookupName eq \'PropertySubType\'', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);
        $response1->assertOk();
        $this->assertSame('Duplex', $response1->json('value.0.LookupValue'));

        // Second call: filter by PropertyCondition (different filter = different cache entry)
        $response2 = $this->getJson('/api/v1/lookup?$filter=LookupName eq \'PropertyCondition\'', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);
        $response2->assertOk();
        $this->assertSame('Fixer', $response2->json('value.0.LookupValue'));

        $this->assertSame(2, $callCount);

        // Both cache entries should exist
        $this->assertSame(2, DB::table('bridge_search_cache')
            ->where('partition_key', 'lookups:stellar')
            ->count());

        // Third call: same filter as first — should be cached
        $response3 = $this->getJson('/api/v1/lookup?$filter=LookupName eq \'PropertySubType\'', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);
        $response3->assertOk();
        $this->assertSame('Duplex', $response3->json('value.0.LookupValue'));
        $this->assertSame(2, $callCount, 'No additional Bridge call for cached filter.');
    }

    public function test_lookup_passes_through_bridge_error_on_cache_miss(): void
    {
        $this->createDomain();

        Http::fake([
            'bridge.test/*' => Http::response(
                json_encode(['error' => ['message' => 'Unauthorized']], JSON_THROW_ON_ERROR),
                401,
                ['Content-Type' => 'application/json']
            ),
        ]);

        $response = $this->getJson('/api/v1/lookup', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ]);

        $response->assertStatus(401);
    }

    public function test_lookup_clear_cache_command_removes_lookups_entries(): void
    {
        $this->createDomain();

        $lookupData = $this->fakeLookupResponse();

        Http::fake([
            'bridge.test/*' => Http::response(
                json_encode($lookupData, JSON_THROW_ON_ERROR),
                200,
                ['Content-Type' => 'application/json']
            ),
        ]);

        // Prime the cache
        $this->getJson('/api/v1/lookup', [
            'X-Domain-Slug' => 'searchtampabayhouses.com',
        ])->assertOk();

        $this->assertSame(1, DB::table('bridge_search_cache')
            ->where('partition_key', 'lookups:stellar')
            ->count());

        // Insert a non-lookup cache entry to verify it's not deleted
        DB::table('bridge_search_cache')->insert([
            'partition_key' => 'user:1',
            'fingerprint' => 'abc123',
            'compressed_data' => gzencode('{}', 9),
            'last_updated' => now(),
        ]);

        // Run the clear command with --dataset=stellar
        $this->artisan('bridge:clear-lookups-cache', ['--dataset' => 'stellar'])
            ->assertSuccessful()
            ->expectsOutputToContain('Deleted 1 cached lookup response(s)');

        // Lookup cache should be gone
        $this->assertSame(0, DB::table('bridge_search_cache')
            ->where('partition_key', 'lookups:stellar')
            ->count());

        // Non-lookup cache entry should still exist
        $this->assertSame(1, DB::table('bridge_search_cache')
            ->where('partition_key', 'user:1')
            ->count());
    }

    public function test_lookup_clear_cache_all_flag_clears_all_datasets(): void
    {
        $this->createDomain();

        // Insert lookup cache entries for two datasets
        DB::table('bridge_search_cache')->insert([
            'partition_key' => 'lookups:stellar',
            'fingerprint' => 'aaa',
            'compressed_data' => gzencode('{}', 9),
            'last_updated' => now(),
        ]);
        DB::table('bridge_search_cache')->insert([
            'partition_key' => 'lookups:mfrmls',
            'fingerprint' => 'bbb',
            'compressed_data' => gzencode('{}', 9),
            'last_updated' => now(),
        ]);

        $this->artisan('bridge:clear-lookups-cache', ['--all' => true])
            ->assertSuccessful()
            ->expectsOutputToContain('Deleted 2 cached lookup response(s)');

        $this->assertSame(0, DB::table('bridge_search_cache')
            ->where('partition_key', 'like', 'lookups:%')
            ->count());
    }

    public function test_lookup_clear_cache_command_requires_flag(): void
    {
        $this->artisan('bridge:clear-lookups-cache')
            ->assertFailed()
            ->expectsOutputToContain('Specify --all or --dataset=');
    }

    public function test_lookup_requires_auth(): void
    {
        Http::fake();

        $this->getJson('/api/v1/lookup')->assertStatus(401);
    }
}
