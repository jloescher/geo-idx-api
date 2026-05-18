<?php

namespace Tests\Unit\Providers;

use App\Providers\TelescopeServiceProvider;
use PHPUnit\Framework\Attributes\DataProvider;
use ReflectionMethod;
use Tests\TestCase;

class TelescopeServiceProviderTest extends TestCase
{
    #[DataProvider('recordAllEnvironmentsProvider')]
    public function test_should_record_all_entries(string $environment, bool $recordAllEnv, bool $expected): void
    {
        $this->app['env'] = $environment;
        putenv($recordAllEnv ? 'TELESCOPE_RECORD_ALL=true' : 'TELESCOPE_RECORD_ALL=false');

        $provider = new TelescopeServiceProvider($this->app);
        $method = new ReflectionMethod(TelescopeServiceProvider::class, 'shouldRecordAllEntries');

        $this->assertSame($expected, $method->invoke($provider));

        putenv('TELESCOPE_RECORD_ALL');
    }

    /**
     * @return array<string, array{0: string, 1: bool, 2: bool}>
     */
    public static function recordAllEnvironmentsProvider(): array
    {
        return [
            'local' => ['local', false, true],
            'staging' => ['staging', false, true],
            'production' => ['production', false, false],
            'production_with_record_all' => ['production', true, true],
        ];
    }
}
