<?php

namespace Tests\Unit\Support;

use App\Support\IdxPublicUrls;
use Illuminate\Http\Request;
use Tests\TestCase;

class IdxPublicUrlsTest extends TestCase
{
    public function test_derives_platform_host_from_dev_idx_api_request(): void
    {
        $request = Request::create(
            'https://dev-idx-api.quantyralabs.cc/openapi.json',
            'GET',
            server: [
                'HTTP_HOST' => 'dev-idx-api.quantyralabs.cc',
                'HTTPS' => 'on',
            ],
        );

        $this->assertSame('https://dev-idx-api.quantyralabs.cc', IdxPublicUrls::apiBaseForRequest($request));
        $this->assertSame('https://dev-idx.quantyralabs.cc', IdxPublicUrls::platformBaseForRequest($request));
    }

    public function test_derives_platform_host_from_staging_idx_api_request(): void
    {
        $request = Request::create(
            'https://staging-idx-api.quantyralabs.cc/x',
            'GET',
            server: [
                'HTTP_HOST' => 'staging-idx-api.quantyralabs.cc',
                'HTTPS' => 'on',
            ],
        );

        $this->assertSame('https://staging-idx.quantyralabs.cc', IdxPublicUrls::platformBaseForRequest($request));
    }

    public function test_falls_back_to_config_when_host_is_not_idx_api_style(): void
    {
        config([
            'idx_urls.platform_url' => 'https://cfg-platform.example',
            'idx_urls.api_public_url' => 'https://cfg-api.example',
        ]);

        $request = Request::create('http://localhost:8000/', 'GET', server: [
            'HTTP_HOST' => 'localhost:8000',
        ]);

        $this->assertSame('https://cfg-api.example', IdxPublicUrls::apiBaseForRequest($request));
        $this->assertSame('https://cfg-platform.example', IdxPublicUrls::platformBaseForRequest($request));
    }
}
