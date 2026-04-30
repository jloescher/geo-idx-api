<?php

namespace App\Support;

use RuntimeException;

/**
 * Blocks Artisan commands that can wipe or rebuild the database in production.
 */
final class DestructiveDatabaseCommandGuard
{
    /**
     * Commands that must never run against a production database.
     *
     * @var list<string>
     */
    private const REFUSE_IN_PRODUCTION = [
        'migrate:fresh',
        'migrate:refresh',
        'migrate:reset',
        'db:wipe',
    ];

    /**
     * Default fragments when config is missing, null, or empty (e.g. stale `config:cache`
     * built before `config/debug_audit.php` included these keys).
     *
     * @var list<string>
     */
    private const DEFAULT_PROTECTED_DATABASE_FRAGMENTS = ['prod', 'production', 'staging'];

    /**
     * @return list<string>
     */
    private static function protectedDatabaseNameFragments(): array
    {
        $raw = config('debug_audit.protected_database_name_fragments');
        if (is_array($raw) && $raw !== []) {
            return array_values(array_filter(array_map(
                static fn (mixed $v): string => trim((string) $v),
                $raw
            )));
        }

        return self::DEFAULT_PROTECTED_DATABASE_FRAGMENTS;
    }

    /**
     * PHPUnit / in-memory sqlite — never block migrate:fresh (see phpunit.xml DB_*).
     */
    private static function isEphemeralSqliteDatabaseContext(): bool
    {
        $envConnection = (string) (getenv('DB_CONNECTION') ?: $_ENV['DB_CONNECTION'] ?? $_SERVER['DB_CONNECTION'] ?? '');
        $envDatabase = (string) (getenv('DB_DATABASE') ?: $_ENV['DB_DATABASE'] ?? $_SERVER['DB_DATABASE'] ?? '');

        if ($envConnection === 'sqlite' && ($envDatabase === ':memory:' || $envDatabase === '')) {
            return true;
        }

        $name = (string) config('database.default');
        if ((string) config("database.connections.{$name}.driver") !== 'sqlite') {
            return false;
        }
        $database = (string) config("database.connections.{$name}.database", '');

        return $database === ':memory:' || $database === '' || str_starts_with($database, ':memory:');
    }

    public static function mustRefuse(string $command, ?string $environment = null, ?string $databaseName = null): bool
    {
        if (! in_array($command, self::REFUSE_IN_PRODUCTION, true)) {
            return false;
        }

        if ((bool) (config('debug_audit.allow_destructive_db_commands') ?? false)) {
            return false;
        }

        $explicitParams = $environment !== null || $databaseName !== null;

        if (! $explicitParams && self::isEphemeralSqliteDatabaseContext()) {
            return false;
        }

        $environment ??= app()->environment();
        $databaseName ??= (string) config('database.connections.'.config('database.default').'.database');

        if ($environment === 'production') {
            return true;
        }

        $fragments = self::protectedDatabaseNameFragments();
        $lowerDatabase = mb_strtolower($databaseName);

        foreach ($fragments as $fragment) {
            if ($fragment !== '' && str_contains($lowerDatabase, mb_strtolower((string) $fragment))) {
                return true;
            }
        }

        return false;
    }

    public static function assertNotRefused(string $command, ?string $environment = null, ?string $databaseName = null): void
    {
        if (self::mustRefuse($command, $environment, $databaseName)) {
            throw new RuntimeException(
                "Refused to run `{$command}` on a protected database context. ".
                'Set ALLOW_DESTRUCTIVE_DB_COMMANDS=true only for intentional maintenance windows.'
            );
        }
    }
}
