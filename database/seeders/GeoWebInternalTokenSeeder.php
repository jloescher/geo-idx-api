<?php

namespace Database\Seeders;

use App\Models\User;
use Illuminate\Database\Seeder;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Str;

class GeoWebInternalTokenSeeder extends Seeder
{
    public function run(): void
    {
        $user = User::query()->firstOrCreate(
            ['email' => 'geo-web-internal@quantyralabs.internal'],
            [
                'name' => 'Geo Web Internal',
                'password' => Hash::make(Str::random(64)),
            ],
        );

        if ($user->tokens()->where('name', 'geo-web-internal')->exists()) {
            return;
        }

        $plain = $user->createToken('geo-web-internal', ['idx:full'])->plainTextToken;

        if ($this->command) {
            $this->command->warn('Seeded Sanctum token geo-web-internal. Set IDX_API_INTERNAL_TOKEN in .env:');
            $this->command->line('IDX_API_INTERNAL_TOKEN='.$plain);
        }
    }
}
