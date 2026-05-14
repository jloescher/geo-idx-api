<?php

declare(strict_types=1);

/**
 * Regenerate Laravel IDE helper stubs for VS Code / Cursor (Intelephense) and PhpStorm.
 * Invoked from Composer post-install / post-update when barryvdh/laravel-ide-helper is present.
 *
 * Skips quietly when the dev package is not installed (--no-dev) or when no .env exists yet.
 */
$root = dirname(__DIR__);

if (! is_dir($root.'/vendor/barryvdh/laravel-ide-helper')) {
    exit(0);
}

if (! is_file($root.'/.env')) {
    fwrite(STDERR, "Skipping IDE helper generation: .env not found (copy .env.example, then run: composer ide-helper).\n");

    exit(0);
}

$artisan = escapeshellarg(PHP_BINARY).' '.escapeshellarg($root.'/artisan');

passthru("{$artisan} ide-helper:generate -M --ansi --no-interaction", $code);
if ($code !== 0) {
    exit($code);
}

passthru("{$artisan} ide-helper:meta --ansi --no-interaction", $code);
exit($code);
