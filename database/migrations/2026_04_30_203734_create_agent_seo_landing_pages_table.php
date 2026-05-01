<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('agent_seo_landing_pages', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('agent_search_id')->nullable()->constrained('agent_searches')->nullOnDelete();
            $table->foreignId('agent_share_link_id')->nullable()->constrained('agent_share_links')->nullOnDelete();
            $table->string('slug')->nullable();
            $table->string('canonical_path');
            $table->string('canonical_url');
            $table->string('status', 32)->default('active');
            $table->timestamp('published_at')->nullable();
            $table->timestamps();

            $table->index(['user_id', 'status']);
            $table->unique(['user_id', 'canonical_path']);
            $table->unique(['user_id', 'agent_search_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('agent_seo_landing_pages');
    }
};
