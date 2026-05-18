<?php

namespace App\Support;

final class DashboardUrl
{
    public static function panel(string $panel = 'dashboard'): string
    {
        if ($panel === 'dashboard') {
            return '/dashboard';
        }

        return '/dashboard?panel='.urlencode($panel);
    }
}
