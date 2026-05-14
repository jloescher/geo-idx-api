<?php

namespace Tests\Feature;

use Symfony\Component\HttpFoundation\BinaryFileResponse;
use Tests\TestCase;

class DomainRoutingTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        $this->withoutVite();
    }

    public function test_platform_domain_root_shows_marketing_home(): void
    {
        $this->withoutVite();

        $response = $this->get('https://dev-idx.quantyralabs.cc/');

        $response->assertOk();
        $response->assertSee('GeoIDX API', false);
        $response->assertSee('Log in');
    }

    public function test_api_domain_root_redirects_to_matching_sales_host(): void
    {
        $response = $this->get('https://staging-idx-api.quantyralabs.cc/');

        $response->assertRedirect('https://staging-idx.quantyralabs.cc');
    }

    public function test_api_domain_exposes_openapi_31_spec_document(): void
    {
        $response = $this->get('https://staging-idx-api.quantyralabs.cc/openapi.json');

        $response->assertOk();
        $response->assertHeader('content-type', 'application/json; charset=UTF-8');

        /** @var BinaryFileResponse $binaryResponse */
        $binaryResponse = $response->baseResponse;
        $servedFile = $binaryResponse->getFile();
        $this->assertNotNull($servedFile);
        $this->assertSame(base_path('docs/yaak-api-collection.json'), $servedFile->getPathname());

        $spec = json_decode((string) file_get_contents($servedFile->getPathname()), true, flags: JSON_THROW_ON_ERROR);
        $this->assertSame('3.1.0', $spec['openapi'] ?? null);
        $this->assertSame('Quantyra GeoIDX API', $spec['info']['title'] ?? null);
    }

    public function test_api_domain_serves_swagger_ui_page(): void
    {
        $response = $this->get('https://staging-idx-api.quantyralabs.cc/swagger');

        $response->assertOk();
        $response->assertSee('SwaggerUIBundle');
        $response->assertSee('/openapi.json');
    }

    public function test_leadconnector_embed_path_is_removed(): void
    {
        $response = $this->get('https://idx.quantyralabs.cc/leadconnectorapp');

        $response->assertNotFound();
    }
}
