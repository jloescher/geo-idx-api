<?php

namespace App\Support;

final class DashboardUrl
{
    public static function panel(string $panel = 'setup'): string
    {
        if ($panel === 'setup' || $panel === 'dashboard') {
            return '/dashboard';
        }

        return '/dashboard?panel='.urlencode($panel);
    }
}
