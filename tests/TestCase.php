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
        $isInMemorySqlite = $defaultConnection === 'sqlite' && $database === ':memory:';

        if (! $isInMemorySqlite) {
            throw new RuntimeException(
                sprintf(
                    'Refusing to run tests with non-ephemeral DB configuration [%s:%s]. Set ALLOW_DESTRUCTIVE_TEST_DB=true to override intentionally.',
                    $defaultConnection,
                    $database
                )
            );
        }
    }
}
