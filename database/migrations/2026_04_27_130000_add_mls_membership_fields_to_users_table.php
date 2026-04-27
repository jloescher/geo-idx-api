<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('users', function (Blueprint $table): void {
            $table->string('mls_id')->nullable()->after('widget_embed_site_key');
            $table->string('mls_email')->nullable()->after('mls_id');
            $table->json('assigned_mls_datasets')->nullable()->after('mls_email');
            $table->string('mls_membership_status', 32)->default('pending')->after('assigned_mls_datasets');
            $table->timestamp('mls_membership_verified_at')->nullable()->after('mls_membership_status');
            $table->timestamp('mls_membership_next_reverify_at')->nullable()->after('mls_membership_verified_at');
            $table->text('mls_membership_last_error')->nullable()->after('mls_membership_next_reverify_at');
        });
    }

    public function down(): void
    {
        Schema::table('users', function (Blueprint $table): void {
            $table->dropColumn([
                'mls_id',
                'mls_email',
                'assigned_mls_datasets',
                'mls_membership_status',
                'mls_membership_verified_at',
                'mls_membership_next_reverify_at',
                'mls_membership_last_error',
            ]);
        });
    }
};
