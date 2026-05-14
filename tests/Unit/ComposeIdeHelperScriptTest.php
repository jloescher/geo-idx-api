<?php

declare(strict_types=1);

namespace Tests\Unit;

use PHPUnit\Framework\TestCase;

final class ComposeIdeHelperScriptTest extends TestCase
{
    public function test_compose_ide_helper_script_exists_and_targets_artisan_commands(): void
    {
        $path = dirname(__DIR__, 2).'/scripts/compose-ide-helper.php';

        $this->assertFileExists($path);

        $contents = (string) file_get_contents($path);

        $this->assertStringContainsString('ide-helper:generate', $contents);
        $this->assertStringContainsString('ide-helper:meta', $contents);
        $this->assertStringContainsString('barryvdh/laravel-ide-helper', $contents);
    }
}
