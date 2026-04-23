<?php

namespace Tests\Feature;

use Tests\TestCase;

class DomainRoutingTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        $this->withoutVite();
    }

    public function test_platform_domain_root_shows_sales_page(): void
    {
        $response = $this->get('https://dev-idx.quantyralabs.cc/');

        $response->assertOk();
        $response->assertSee('Quantyra GeoIDX');
    }

    public function test_api_domain_root_redirects_to_matching_sales_host(): void
    {
        $response = $this->get('https://staging-idx-api.quantyralabs.cc/');

        $response->assertRedirect('https://staging-idx.quantyralabs.cc');
    }

    public function test_leadconnector_app_requires_authentication(): void
    {
        $response = $this->get('https://idx.quantyralabs.cc/leadconnectorapp');

        $response->assertRedirect('/login');
    }
}
