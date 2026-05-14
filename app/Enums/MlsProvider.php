<?php

namespace App\Enums;

enum MlsProvider: string
{
    case STELLAR = 'stellar';

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
