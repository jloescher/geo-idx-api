<?php

namespace Tests\Unit\Support;

use App\Support\DestructiveDatabaseCommandGuard;
use PHPUnit\Framework\Attributes\Test;
use RuntimeException;
use Tests\TestCase;

class DestructiveDatabaseCommandGuardTest extends TestCase
{
    #[Test]
    public function it_refuses_destructive_commands_for_production_environment(): void
    {
        config()->set('debug_audit.allow_destructive_db_commands', false);
        config()->set('debug_audit.protected_database_name_fragments', ['prod', 'staging']);

        $this->assertTrue(DestructiveDatabaseCommandGuard::mustRefuse('migrate:fresh', 'production', 'idx_production'));
        $this->assertTrue(DestructiveDatabaseCommandGuard::mustRefuse('db:wipe', 'production', 'idx_prod'));
        $this->assertFalse(DestructiveDatabaseCommandGuard::mustRefuse('migrate', 'production', 'idx_production'));
    }

    #[Test]
    public function it_refuses_destructive_commands_for_protected_database_names_even_in_local(): void
    {
        config()->set('debug_audit.allow_destructive_db_commands', false);
        config()->set('debug_audit.protected_database_name_fragments', ['prod', 'staging']);

        $this->assertTrue(DestructiveDatabaseCommandGuard::mustRefuse('migrate:fresh', 'local', 'geoidxapi_staging'));
        $this->assertTrue(DestructiveDatabaseCommandGuard::mustRefuse('db:wipe', 'staging', 'customer_prod_primary'));
        $this->assertFalse(DestructiveDatabaseCommandGuard::mustRefuse('migrate:fresh', 'local', 'idx_local'));
    }

    #[Test]
    public function it_falls_back_to_default_protected_fragments_when_config_is_null_or_empty(): void
    {
        config()->set('debug_audit.allow_destructive_db_commands', false);
        config()->set('debug_audit.protected_database_name_fragments', null);

        $this->assertTrue(DestructiveDatabaseCommandGuard::mustRefuse('migrate:fresh', 'local', 'geoidxapi_staging'));

        config()->set('debug_audit.protected_database_name_fragments', []);

        $this->assertTrue(DestructiveDatabaseCommandGuard::mustRefuse('db:wipe', 'local', 'myapp_production_copy'));
    }

    #[Test]
    public function it_respects_override_for_intentional_maintenance(): void
    {
        config()->set('debug_audit.allow_destructive_db_commands', true);
        config()->set('debug_audit.protected_database_name_fragments', ['prod', 'staging']);

        $this->assertFalse(DestructiveDatabaseCommandGuard::mustRefuse('migrate:fresh', 'production', 'idx_production'));
        $this->assertFalse(DestructiveDatabaseCommandGuard::mustRefuse('db:wipe', 'local', 'geoidxapi_staging'));
    }

    #[Test]
    public function it_respects_allow_flag_from_process_environment_when_config_is_false(): void
    {
        config()->set('debug_audit.allow_destructive_db_commands', false);
        config()->set('debug_audit.protected_database_name_fragments', ['prod', 'staging']);

        putenv('ALLOW_DESTRUCTIVE_DB_COMMANDS=true');
        $_ENV['ALLOW_DESTRUCTIVE_DB_COMMANDS'] = 'true';

        try {
            $this->assertFalse(DestructiveDatabaseCommandGuard::mustRefuse('migrate:fresh', 'local', 'geoidxapi_staging'));
        } finally {
            putenv('ALLOW_DESTRUCTIVE_DB_COMMANDS=false');
            $_ENV['ALLOW_DESTRUCTIVE_DB_COMMANDS'] = 'false';
            $_SERVER['ALLOW_DESTRUCTIVE_DB_COMMANDS'] = 'false';
            config()->set('debug_audit.allow_destructive_db_commands', false);
        }
    }

    #[Test]
    public function it_throws_when_asserting_a_destructive_command_in_protected_context(): void
    {
        config()->set('debug_audit.allow_destructive_db_commands', false);
        config()->set('debug_audit.protected_database_name_fragments', ['prod', 'staging']);

        $this->expectException(RuntimeException::class);
        DestructiveDatabaseCommandGuard::assertNotRefused('migrate:fresh', 'local', 'geoidxapi_staging');
    }
}
