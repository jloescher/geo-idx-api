<?php

declare(strict_types=1);

/*
| PHPUnit runs this file before merging <env> from phpunit.xml. Loading .env here ensures
| DB_* (e.g. PostgreSQL + PostGIS staging) is available for tests/TestCase.php guards and
| matches how developers run: php artisan test with database credentials from .env.
*/

$projectRoot = dirname(__DIR__);

require $projectRoot.'/vendor/autoload.php';

if (is_readable($projectRoot.'/.env')) {
    Dotenv\Dotenv::createImmutable($projectRoot)->safeLoad();
}
