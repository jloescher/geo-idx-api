<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Allow destructive DB schema commands on protected databases
    |--------------------------------------------------------------------------
    |
    | When false, commands like migrate:fresh and db:wipe are refused when
    | APP_ENV=production OR when the current database name appears protected
    | (for example staging/prod names).
    |
    */
    'allow_destructive_db_commands' => filter_var(env('ALLOW_DESTRUCTIVE_DB_COMMANDS', false), FILTER_VALIDATE_BOOLEAN),

    /*
    |--------------------------------------------------------------------------
    | Protected database name fragments
    |--------------------------------------------------------------------------
    |
    | Case-insensitive fragments matched against config("database.connections.*.database").
    | If any fragment matches, destructive commands are refused.
    |
    */
    'protected_database_name_fragments' => array_values(array_filter(array_map(
        static fn (string $value): string => trim($value),
        explode(',', (string) env('PROTECTED_DATABASE_NAME_FRAGMENTS', 'prod,production,staging'))
    ))),

    /*
    |--------------------------------------------------------------------------
    | Allow deleting User models in production
    |--------------------------------------------------------------------------
    |
    | Revenue / safety: production should not drop subscriber accounts from code.
    | Set true only for a deliberate maintenance window.
    |
    */
    'allow_user_deletion_in_production' => filter_var(env('ALLOW_USER_DELETION_IN_PRODUCTION', false), FILTER_VALIDATE_BOOLEAN),

];
