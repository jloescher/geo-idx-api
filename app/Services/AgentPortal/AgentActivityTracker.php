<?php

namespace App\Services\AgentPortal;

use App\Models\AgentActivityEvent;
use App\Models\User;

final class AgentActivityTracker
{
    /**
     * Record an activity event for a user.
     *
     * @param  array<string, mixed>|null  $metadata
     */
    public function record(User $user, string $eventType, string $title, string $status = 'completed', ?array $metadata = null): AgentActivityEvent
    {
        return AgentActivityEvent::query()->create([
            'user_id' => $user->id,
            'event_type' => $eventType,
            'title' => $title,
            'status' => $status,
            'metadata_json' => $metadata,
        ]);
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function recentEvents(User $user, int $limit = 15): array
    {
        return AgentActivityEvent::query()
            ->where('user_id', $user->id)
            ->latest()
            ->limit($limit)
            ->get()
            ->map(static fn (AgentActivityEvent $event): array => [
                'type' => $event->event_type,
                'title' => $event->title,
                'status' => $event->status,
                'at' => $event->created_at?->toIso8601String(),
                'metadata' => $event->metadata_json,
            ])
            ->values()
            ->all();
    }
}
