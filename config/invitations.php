<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Invitation link lifetime
    |--------------------------------------------------------------------------
    |
    | Hours until a pending user invitation expires. Default: 168 (7 days).
    |
    */

    'ttl_hours' => (int) env('IDX_INVITATION_TTL_HOURS', 168),

];
