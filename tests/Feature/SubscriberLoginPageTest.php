<?php

namespace Tests\Feature;

use Tests\TestCase;

class SubscriberLoginPageTest extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        $this->withoutVite();
    }

    public function test_subscriber_login_page_renders_on_platform_host(): void
    {
        $response = $this->get('https://dev-idx.quantyralabs.cc/login');

        $response->assertOk();
        $response->assertSee('Subscriber login');
        $response->assertSee('Sign in');
        $response->assertSee('GeoIDX');
    }
}
