<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('agent_contact_tags', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->string('lead_id');
            $table->string('tag', 128);
            $table->timestamps();

            $table->index(['user_id', 'tag']);
            $table->index(['lead_id']);
            $table->unique(['user_id', 'lead_id', 'tag']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('agent_contact_tags');
    }
};
