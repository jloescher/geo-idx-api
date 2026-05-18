<?php

declare(strict_types=1);

namespace App\Services\Bridge;

enum HybridSearchRouteMode: string
{
    case PostgresOnly = 'postgres_only';
    case BridgeOnly = 'bridge_only';
    case Split = 'split';
}
