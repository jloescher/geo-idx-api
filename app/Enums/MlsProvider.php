<?php

namespace App\Enums;

enum MlsProvider: string
{
    case STELLAR = 'stellar';
    case SPARK = 'spark';

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
