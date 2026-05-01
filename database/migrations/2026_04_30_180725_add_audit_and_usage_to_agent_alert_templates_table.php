<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('agent_alert_templates', function (Blueprint $table) {
            $table->json('audit_json')->nullable()->after('body_json');
            $table->unsignedInteger('usage_count')->default(0)->after('audit_json');
            $table->timestamp('last_used_at')->nullable()->after('usage_count');
            $table->json('schedule_json')->nullable()->after('last_used_at');
        });
    }

    public function down(): void
    {
        Schema::table('agent_alert_templates', function (Blueprint $table) {
            $table->dropColumn(['audit_json', 'usage_count', 'last_used_at', 'schedule_json']);
        });
    }
};
