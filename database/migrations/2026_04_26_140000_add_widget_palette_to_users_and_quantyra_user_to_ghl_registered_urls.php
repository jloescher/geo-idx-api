<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('users', function (Blueprint $table) {
            $table->json('widget_palette')->nullable()->after('remember_token');
        });

        if (Schema::hasTable('ghl_registered_urls')) {
            Schema::table('ghl_registered_urls', function (Blueprint $table) {
                $table->foreignId('quantyra_user_id')->nullable()->after('ghl_oauth_token_id')->constrained('users')->nullOnDelete();
            });
        }
    }

    public function down(): void
    {
        if (Schema::hasTable('ghl_registered_urls')) {
            Schema::table('ghl_registered_urls', function (Blueprint $table) {
                $table->dropConstrainedForeignId('quantyra_user_id');
            });
        }

        Schema::table('users', function (Blueprint $table) {
            $table->dropColumn('widget_palette');
        });
    }
};
