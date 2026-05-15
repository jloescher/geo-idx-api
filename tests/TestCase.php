<?php

namespace Tests;

use Illuminate\Foundation\Testing\TestCase as BaseTestCase;
use RuntimeException;

abstract class TestCase extends BaseTestCase
{
    protected function setUp(): void
    {
        $this->guardAgainstNonTestDatabase();

        parent::setUp();
    }

    private function guardAgainstNonTestDatabase(): void
    {
        $appEnv = (string) ($_ENV['APP_ENV'] ?? $_SERVER['APP_ENV'] ?? getenv('APP_ENV') ?: '');
        if ($appEnv !== 'testing') {
            return;
        }

        $allowDestructive = (string) ($_ENV['ALLOW_DESTRUCTIVE_TEST_DB'] ?? $_SERVER['ALLOW_DESTRUCTIVE_TEST_DB'] ?? getenv('ALLOW_DESTRUCTIVE_TEST_DB') ?: '');
        if (in_array(strtolower($allowDestructive), ['1', 'true', 'yes', 'on'], true)) {
            return;
        }

        $defaultConnection = (string) ($_ENV['DB_CONNECTION'] ?? $_SERVER['DB_CONNECTION'] ?? getenv('DB_CONNECTION') ?: '');
        $database = (string) ($_ENV['DB_DATABASE'] ?? $_SERVER['DB_DATABASE'] ?? getenv('DB_DATABASE') ?: '');

        if ($this->isAllowedEphemeralTestDatabase($defaultConnection, $database)) {
            return;
        }

        throw new RuntimeException(
            sprintf(
                'Refusing to run tests with DB configuration [%s:%s]. Use a dedicated PostgreSQL database named "testing" or "idx_api_testing" (set DB_* in .env; phpunit does not override them—see tests/bootstrap.php), or set ALLOW_DESTRUCTIVE_TEST_DB=true when you intentionally target another database (e.g. staging with PostGIS). Warning: many tests use RefreshDatabase, which reapplies migrations and can destroy data—never enable this against a database you cannot reset.',
                $defaultConnection,
                $database
            )
        );
    }

    private function isAllowedEphemeralTestDatabase(string $connection, string $database): bool
    {
        if ($connection !== 'pgsql') {
            return false;
        }

        $normalized = strtolower(trim($database));

        return in_array($normalized, ['testing', 'idx_api_testing'], true);
    }
}
