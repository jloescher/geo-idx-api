<?php

namespace Database\Seeders;

use App\Models\User;
use Illuminate\Database\Seeder;
use Illuminate\Support\Facades\Hash;

class AdminUserSeeder extends Seeder
{
    /**
     * Seed a bootstrap administrator from environment variables (never commit secrets).
     */
    public function run(): void
    {
        $email = env('ADMIN_SEED_EMAIL');
        $password = env('ADMIN_SEED_PASSWORD');

        if (! is_string($email) || trim($email) === '' || ! is_string($password) || $password === '') {
            if ($this->command !== null) {
                $this->command->warn('ADMIN_SEED_EMAIL / ADMIN_SEED_PASSWORD not set; skipping AdminUserSeeder.');
            }

            return;
        }

        $normalized = mb_strtolower(trim($email));
        $name = is_string(env('ADMIN_SEED_NAME')) && trim((string) env('ADMIN_SEED_NAME')) !== ''
            ? trim((string) env('ADMIN_SEED_NAME'))
            : 'Administrator';

        User::query()->updateOrCreate(
            ['email' => $normalized],
            [
                'name' => $name,
                'password' => Hash::make($password),
                'is_admin' => true,
            ],
        );
    }
}
