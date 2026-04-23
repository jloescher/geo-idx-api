<?php

namespace App\Console\Commands;

use App\Models\User;
use Illuminate\Console\Command;
use Illuminate\Support\Str;

class IssueGeoWebInternalTokenCommand extends Command
{
    protected $signature = 'idx-api:issue-geo-web-token {--force : Delete existing geo-web-internal tokens first}';

    protected $description = 'Create (or rotate) the geo-web-internal Sanctum token with idx:full and print IDX_API_INTERNAL_TOKEN for .env';

    public function handle(): int
    {
        $user = User::query()->firstOrCreate(
            ['email' => 'geo-web-internal@quantyralabs.internal'],
            [
                'name' => 'Geo Web Internal',
                'password' => bcrypt(Str::random(64)),
            ],
        );

        if ((bool) $this->option('force')) {
            $user->tokens()->where('name', 'geo-web-internal')->delete();
        } elseif ($user->tokens()->where('name', 'geo-web-internal')->exists()) {
            $this->error('A geo-web-internal token already exists. Re-run with --force to rotate.');

            return self::FAILURE;
        }

        $plain = $user->createToken('geo-web-internal', ['idx:full'])->plainTextToken;

        $this->line('Add this to your secrets / .env for geo-web server-to-server calls:');
        $this->line('IDX_API_INTERNAL_TOKEN='.$plain);

        return self::SUCCESS;
    }
}
