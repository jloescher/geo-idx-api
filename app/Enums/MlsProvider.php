<?php

namespace App\Enums;

/**
 * Revenue impact: explicit MLS upstream selection keeps billing + rate limits attributable per feed.
 *
 * Compliance: IDX and participant agreements require clear data lineage; never mix Spark credentials into Bridge calls or vice versa.
 */
enum MlsProvider: string
{
    case Bridge = 'bridge';
    case Spark = 'spark';
}
