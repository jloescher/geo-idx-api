<?php

namespace Tests\Unit\Services\Bridge;

use App\Models\Domain;
use App\Services\Bridge\MlsDatasetResolver;
use Illuminate\Http\Request;
use Symfony\Component\HttpKernel\Exception\HttpException;
use Tests\TestCase;

class MlsDatasetResolverTest extends TestCase
{
    private MlsDatasetResolver $resolver;

    protected function setUp(): void
    {
        parent::setUp();
        $this->resolver = app(MlsDatasetResolver::class);
    }

    public function test_resolve_dataset_from_query_param(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami']]);

        $request = new Request;
        $request->query->set('dataset', 'miami');

        $this->assertSame('miami', $this->resolver->resolveDataset($request));
    }

    public function test_resolve_dataset_from_domain(): void
    {
        config(['bridge.datasets' => ['stellar']]);
        config(['bridge.dataset' => 'stellar']);

        $domain = new Domain;
        $domain->domain_slug = 'testdomain.com';
        $domain->is_active = true;
        $domain->mls_dataset = 'stellar';

        $request = new Request;
        $request->attributes->set('bridge.domain', $domain);

        $this->assertSame('stellar', $this->resolver->resolveDataset($request));
    }

    public function test_resolve_dataset_from_domain_with_null_falls_back_to_global(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami']]);
        config(['bridge.dataset' => 'miami']);

        $domain = new Domain;
        $domain->domain_slug = 'testdomain.com';
        $domain->is_active = true;
        $domain->mls_dataset = null;

        $request = new Request;
        $request->attributes->set('bridge.domain', $domain);

        $this->assertSame('miami', $this->resolver->resolveDataset($request));
    }

    public function test_resolve_dataset_falls_back_to_global_default(): void
    {
        config(['bridge.dataset' => 'stellar']);

        $request = new Request;

        $this->assertSame('stellar', $this->resolver->resolveDataset($request));
    }

    public function test_validate_dataset_accepts_valid_dataset(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami']]);

        $this->resolver->validateDataset('stellar');
        $this->resolver->validateDataset('miami');

        $this->assertTrue(true); // No exception thrown
    }

    public function test_validate_dataset_rejects_invalid_dataset(): void
    {
        config(['bridge.datasets' => ['stellar']]);

        $this->expectException(HttpException::class);
        $this->expectExceptionMessage("MLS feed 'miami' is not available");

        $this->resolver->validateDataset('miami');
    }

    public function test_get_available_datasets_returns_config_values(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami', '  phoenix  ']]);

        $datasets = $this->resolver->getAvailableDatasets();

        $this->assertSame(['stellar', 'miami', 'phoenix'], $datasets);
    }

    public function test_get_available_datasets_defaults_to_stellar(): void
    {
        config(['bridge.datasets' => null]);

        $datasets = $this->resolver->getAvailableDatasets();

        $this->assertSame(['stellar'], $datasets);
    }

    public function test_query_param_takes_priority_over_domain(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami']]);
        config(['bridge.dataset' => 'stellar']);

        $domain = new Domain;
        $domain->domain_slug = 'testdomain.com';
        $domain->is_active = true;
        $domain->mls_dataset = 'miami';

        $request = new Request;
        $request->query->set('dataset', 'stellar');
        $request->attributes->set('bridge.domain', $domain);

        $this->assertSame('stellar', $this->resolver->resolveDataset($request));
    }

    public function test_validate_dataset_for_site_rejects_disabled_dataset(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami']]);

        $this->expectException(HttpException::class);
        $this->expectExceptionMessage('not enabled');

        $this->resolver->validateDatasetForSite('miami', ['stellar']);
    }

    public function test_datasets_enabled_for_domain_intersects_with_global_catalog(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami']]);

        $domain = new Domain;
        $domain->allowed_mls_datasets = ['stellar', 'phoenix'];

        $enabled = $this->resolver->datasetsEnabledForDomain($domain);

        $this->assertSame(['stellar'], $enabled);
    }

    public function test_resolve_dataset_falls_back_when_global_default_not_enabled_for_domain(): void
    {
        config(['bridge.datasets' => ['stellar', 'miami']]);
        config(['bridge.dataset' => 'stellar']);

        $domain = new Domain;
        $domain->allowed_mls_datasets = ['miami'];

        $request = new Request;
        $request->attributes->set('bridge.domain', $domain);

        $this->assertSame('miami', $this->resolver->resolveDataset($request));
    }
}
