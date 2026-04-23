<?php

namespace Database\Seeders;

use App\Models\Domain;
use Illuminate\Database\Seeder;

class DomainSeeder extends Seeder
{
    public function run(): void
    {
        Domain::query()->updateOrCreate(
            ['domain_slug' => 'searchtampabayhouses.com'],
            ['is_active' => true],
        );
    }
}
