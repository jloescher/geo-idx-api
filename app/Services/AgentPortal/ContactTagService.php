<?php

namespace App\Services\AgentPortal;

use App\Models\AgentContactTag;
use App\Models\User;
use Illuminate\Support\Collection;

final class ContactTagService
{
    /**
     * Sync tags for a specific lead/contact, replacing existing tags.
     *
     * @param  list<string>  $tags
     * @return list<string> The final set of tag names
     */
    public function syncTags(User $user, string $leadId, array $tags): array
    {
        AgentContactTag::query()
            ->where('user_id', $user->id)
            ->where('lead_id', $leadId)
            ->delete();

        $uniqueTags = collect($tags)
            ->map(static fn (string $tag): string => trim($tag))
            ->filter(static fn (string $tag): bool => $tag !== '')
            ->unique()
            ->values();

        foreach ($uniqueTags as $tag) {
            AgentContactTag::query()->create([
                'user_id' => $user->id,
                'lead_id' => $leadId,
                'tag' => $tag,
            ]);
        }

        return $uniqueTags->all();
    }

    /**
     * @return list<string>
     */
    public function getTagsForLead(User $user, string $leadId): array
    {
        return AgentContactTag::query()
            ->where('user_id', $user->id)
            ->where('lead_id', $leadId)
            ->pluck('tag')
            ->values()
            ->all();
    }

    /**
     * Get all unique tag names used by a user.
     *
     * @return list<string>
     */
    public function getAllTagNames(User $user): array
    {
        return AgentContactTag::query()
            ->where('user_id', $user->id)
            ->pluck('tag')
            ->unique()
            ->sort()
            ->values()
            ->all();
    }

    /**
     * Find lead IDs that have ALL the specified tags.
     *
     * @param  list<string>  $tags
     * @return Collection<int, string>
     */
    public function findLeadsByTags(User $user, array $tags): Collection
    {
        if (empty($tags)) {
            return collect();
        }

        $query = AgentContactTag::query()
            ->where('user_id', $user->id)
            ->whereIn('tag', $tags);

        return $query
            ->selectRaw('lead_id, COUNT(DISTINCT tag) as matched_tags')
            ->groupBy('lead_id')
            ->havingRaw('COUNT(DISTINCT tag) = ?', [count($tags)])
            ->pluck('lead_id');
    }
}
