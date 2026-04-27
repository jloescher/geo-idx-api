<?php

namespace App\Services;

use App\Ghl\Widgets\Models\GhlRegisteredUrl;
use App\Models\User;

final class SubscriberLeadScopeService
{
    /**
     * @return list<string>
     */
    public function locationIdsForUser(User $user): array
    {
        $ids = GhlRegisteredUrl::query()
            ->where('quantyra_user_id', $user->id)
            ->whereNotNull('ghl_location_id')
            ->pluck('ghl_location_id')
            ->filter(static fn (mixed $v): bool => is_string($v) && trim($v) !== '')
            ->map(static fn (string $v): string => trim($v))
            ->values()
            ->all();

        $ids[] = 'direct-'.$user->id;

        return array_values(array_unique($ids));
    }
}
