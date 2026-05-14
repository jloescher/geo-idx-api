<?php

namespace App\Enums;

enum MlsProvider: string
{
    case STELLAR = 'stellar';
    case SPARK_SPACE_COAST = 'spark_space_coast';
    case SPARK_BEACHES = 'spark_beaches';
    // Add more MLS as we expand

    public function label(): string
    {
        return match($this) {
            self::STELLAR => 'Stellar MLS',
            self::SPARK_SPACE_COAST => 'Space Coast MLS',
            self::SPARK_BEACHES => 'Beaches MLS',
        };
    }
}
